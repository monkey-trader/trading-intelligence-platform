package position

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// fixtureData liest eine Fixture und liefert ihr Feld data roh.
func fixtureDataClosed(t *testing.T, name string) json.RawMessage {
	t.Helper()
	content, err := os.ReadFile(fixtureDir + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return envelope["data"]
}

// fakeGetClient implementiert bingx.GetClient für Tests.
type fakeGetClient struct {
	responses map[string]json.RawMessage
	errors    map[string]error
}

func (f *fakeGetClient) Get(_ context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	key := path + "|" + encodeParamSlice(params)
	if f.errors != nil && f.errors[key] != nil {
		return nil, f.errors[key]
	}
	if f.responses != nil {
		return f.responses[key], nil
	}
	return nil, nil
}

func encodeParamSlice(params []bingx.QueryParam) string {
	if params == nil {
		return ""
	}
	var parts []string
	for _, p := range params {
		parts = append(parts, p.Name+"="+p.Value)
	}
	return strings.Join(parts, "&")
}

// buildSourceKey baut einen konsistenten Key für die Fake-Responses: symbol, startTs, endTs, pageIndex, pageSize.
func buildSourceKey(symbol string, startTs, endTs int64, pageIndex int, pageSize int) string {
	return closedPositionHistoryPath + "|" + encodeParamSlice([]bingx.QueryParam{
		{Name: "symbol", Value: symbol},
		{Name: "startTs", Value: fmt.Sprintf("%d", startTs)},
		{Name: "endTs", Value: fmt.Sprintf("%d", endTs)},
		{Name: "pageIndex", Value: fmt.Sprintf("%d", pageIndex)},
		{Name: "pageSize", Value: fmt.Sprintf("%d", pageSize)},
	})
}

// Tests

func TestBingXClosedPositionSource_HappyPath_FromFixture(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := int64(1780086400000)

	positionHistoryData := fixtureDataClosed(t, "positionHistory.page0.json")

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("ONDO-USDT", from, to, 0, 100): positionHistoryData,
		},
	}

	source := NewBingXClosedPositionSource(fakeClient)
	positions, err := source.FetchClosedPositions(ctx, "ONDO-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	// Fixture hat 3 Positionen.
	if len(positions) != 3 {
		t.Errorf("len(positions) = %d, erwartet 3", len(positions))
	}

	// Prüfe die erste Position.
	if len(positions) > 0 {
		p := positions[0]
		if p.PositionID != "9000000000000000000" || p.Symbol != "ONDO-USDT" {
			t.Errorf("Position[0] = (%s, %s), erwartet (9000000000000000000, ONDO-USDT)",
				p.PositionID, p.Symbol)
		}
		if p.Side != domainposition.Long {
			t.Errorf("Position[0].Side = %s, erwartet LONG", p.Side)
		}
		if p.Leverage != 7 {
			t.Errorf("Position[0].Leverage = %v, erwartet 7", p.Leverage)
		}
	}
}

func TestBingXClosedPositionSource_RangeTooBig_Error(t *testing.T) {
	from := int64(1780000000000)
	to := from + 100*24*60*60*1000 // 100 Tage > 89

	fakeClient := &fakeGetClientWithCapture{
		fn: func(_ context.Context, _ string, _ []bingx.QueryParam) (json.RawMessage, error) {
			t.Errorf("Client darf bei zu großem Bereich nicht aufgerufen werden")
			return nil, nil
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(context.Background(), "BTC-USDT", from, to)

	var tooLarge *domainposition.QueryRangeTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("Fehler = %v, erwartet *QueryRangeTooLargeError", err)
	}
	if tooLarge.From != from || tooLarge.To != to {
		t.Errorf("Bereich = %d–%d, erwartet %d–%d", tooLarge.From, tooLarge.To, from, to)
	}
}

func TestBingXClosedPositionSource_FromAfterTo_ErrorWithoutClientCall(t *testing.T) {
	from := int64(1780000000001)
	to := int64(1780000000000)

	fakeClient := &fakeGetClientWithCapture{
		fn: func(_ context.Context, _ string, _ []bingx.QueryParam) (json.RawMessage, error) {
			t.Errorf("Client darf bei from > to nicht aufgerufen werden")
			return nil, nil
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(context.Background(), "BTC-USDT", from, to)

	var rangeErr *domainposition.InvalidQueryRangeError
	if !errors.As(err, &rangeErr) {
		t.Fatalf("Fehler = %v (%T), erwartet *InvalidQueryRangeError", err, err)
	}
	if rangeErr.From != from || rangeErr.To != to {
		t.Errorf("Fehler = (%d, %d), erwartet (%d, %d)", rangeErr.From, rangeErr.To, from, to)
	}
}

func TestBingXClosedPositionSource_SourceErrors_AreClosedPositionsUnavailable(t *testing.T) {
	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	testCases := []struct {
		name           string
		clientError    error
		expectedReason domainposition.UnavailableReason
	}{
		{"429", &bingx.RateLimitError{}, domainposition.ReasonRateLimited},
		{"Netzwerk", &bingx.NetworkError{}, domainposition.ReasonUnreachable},
		{"code != 0", &bingx.APIError{Code: 109400, Msg: "x"}, domainposition.ReasonUpstreamError},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeClient := &fakeGetClient{
				errors: map[string]error{buildSourceKey("SOL-USDT", from, to, 0, 100): testCase.clientError},
			}
			source := NewBingXClosedPositionSource(fakeClient)

			_, err := source.FetchClosedPositions(context.Background(), "SOL-USDT", from, to)

			var unavailable *domainposition.ClosedPositionsUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("Fehler = %v (%T), erwartet *ClosedPositionsUnavailableError", err, err)
			}
			if unavailable.Reason != testCase.expectedReason {
				t.Errorf("Reason = %s, erwartet %s", unavailable.Reason, testCase.expectedReason)
			}
			if !errors.Is(err, testCase.clientError) {
				t.Errorf("Ursache nicht über Unwrap erreichbar")
			}
			var openUnavailable *domainposition.OpenPositionsUnavailableError
			if errors.As(err, &openUnavailable) {
				t.Errorf("Fehler darf nicht OpenPositionsUnavailableError sein")
			}
		})
	}
}

func TestBingXClosedPositionSource_MappingError_IsClosedPositionsUnavailableInvalidResponse(t *testing.T) {
	from := int64(1780000000000)
	to := from + 50*24*60*60*1000
	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("BAD-USDT", from, to, 0, 100): json.RawMessage(`{"positionHistory": [{"positionId": "invalid"}]}`),
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(context.Background(), "BAD-USDT", from, to)

	var unavailable *domainposition.ClosedPositionsUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("Fehler = %v (%T), erwartet *ClosedPositionsUnavailableError", err, err)
	}
	if unavailable.Reason != domainposition.ReasonInvalidResponse {
		t.Errorf("Reason = %s, erwartet %s", unavailable.Reason, domainposition.ReasonInvalidResponse)
	}
}

func TestBingXClosedPositionSource_ClientError_ReturnsError(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	fakeClient := &fakeGetClient{
		errors: map[string]error{
			buildSourceKey("SOL-USDT", from, to, 0, 100): errors.New("network error"),
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(ctx, "SOL-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
}

func TestBingXClosedPositionSource_EmptyResponse_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	positionHistoryData := fixtureDataClosed(t, "positionHistory.empty.json")

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("XRP-USDT", from, to, 0, 100): positionHistoryData,
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	positions, err := source.FetchClosedPositions(ctx, "XRP-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len(positions) = %d, erwartet 0", len(positions))
	}
}

func TestBingXClosedPositionSource_Paging_FetchesMultiplePages(t *testing.T) {
	// Paging test mit echten 100er-Seiten: Page 0 voll (100) → Page 1 (5) → Stop.
	// Ergebnis: 105 Positionen insgesamt.
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	// Konstruiere Page 0: exakt 100 Einträge mit unterschiedlichen IDs.
	page0Positions := make([]string, 100)
	for i := 0; i < 100; i++ {
		posID := fmt.Sprintf("9000000000010%05d", i)
		page0Positions[i] = fmt.Sprintf(
			`{"positionId":"%s","symbol":"BTC-USDT","positionSide":"LONG","isolated":true,"openTime":%d,"updateTime":%d,"avgPrice":"50000","avgClosePrice":"51000","realisedProfit":"100","netProfit":"95","positionAmt":"1","closePositionAmt":"1","leverage":10,"closeAllPositions":true,"positionCommission":"-1","totalFunding":"-1"}`,
			posID, from+1000+int64(i), from+2000+int64(i),
		)
	}
	page0Data := json.RawMessage(fmt.Sprintf(`{"positionHistory":[%s]}`, strings.Join(page0Positions, ",")))

	// Konstruiere Page 1: 5 Einträge (< 100 → stoppt Paging).
	page1Positions := make([]string, 5)
	for i := 0; i < 5; i++ {
		posID := fmt.Sprintf("9000000000011%05d", i)
		page1Positions[i] = fmt.Sprintf(
			`{"positionId":"%s","symbol":"BTC-USDT","positionSide":"LONG","isolated":true,"openTime":%d,"updateTime":%d,"avgPrice":"50000","avgClosePrice":"51000","realisedProfit":"100","netProfit":"95","positionAmt":"1","closePositionAmt":"1","leverage":10,"closeAllPositions":true,"positionCommission":"-1","totalFunding":"-1"}`,
			posID, from+200000+1000+int64(i), from+200000+2000+int64(i),
		)
	}
	page1Data := json.RawMessage(fmt.Sprintf(`{"positionHistory":[%s]}`, strings.Join(page1Positions, ",")))

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("BTC-USDT", from, to, 0, 100): page0Data,
			buildSourceKey("BTC-USDT", from, to, 1, 100): page1Data,
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	positions, err := source.FetchClosedPositions(ctx, "BTC-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	// 100 (page0) + 5 (page1) = 105.
	if len(positions) != 105 {
		t.Errorf("len(positions) = %d, erwartet 105 (page0=100 voll → page1=5 < 100 → stop)", len(positions))
	}
}

func TestBingXClosedPositionSource_Paging_StopsOnEmptyPage(t *testing.T) {
	// Page 0 voll (100), Page 1 leer → stop nach Page 0.
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	page0Positions := make([]string, 100)
	for i := 0; i < 100; i++ {
		posID := fmt.Sprintf("9000000000020%05d", i)
		page0Positions[i] = fmt.Sprintf(
			`{"positionId":"%s","symbol":"ETH-USDT","positionSide":"LONG","isolated":true,"openTime":%d,"updateTime":%d,"avgPrice":"3000","avgClosePrice":"3100","realisedProfit":"50","netProfit":"45","positionAmt":"1","closePositionAmt":"1","leverage":10,"closeAllPositions":true,"positionCommission":"-1","totalFunding":"-1"}`,
			posID, from+1000+int64(i), from+2000+int64(i),
		)
	}
	page0Data := json.RawMessage(fmt.Sprintf(`{"positionHistory":[%s]}`, strings.Join(page0Positions, ",")))

	page1Data := json.RawMessage(`{"positionHistory":[]}`)

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("ETH-USDT", from, to, 0, 100): page0Data,
			buildSourceKey("ETH-USDT", from, to, 1, 100): page1Data,
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	positions, err := source.FetchClosedPositions(ctx, "ETH-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(positions) != 100 {
		t.Errorf("len(positions) = %d, erwartet 100 (page1 leer → stop)", len(positions))
	}
}

func TestBingXClosedPositionSource_MappingError_ReturnsError(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	// Response mit ungültigem JSON in der Position.
	badData := json.RawMessage(`{"positionHistory": [{"positionId": "invalid"}]}`)

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("BAD-USDT", from, to, 0, 100): badData,
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(ctx, "BAD-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
}

func TestBingXClosedPositionSource_Exactly89DaysAllowed(t *testing.T) {
	// 89 Tage exakt: OK. 89 Tage + 1 ms: Error.
	ctx := context.Background()

	from := int64(1780000000000)

	// Genau 89 Tage.
	to89Days := from + 89*24*60*60*1000
	posData := json.RawMessage(`{"positionHistory":[]}`)

	fakeClient := &fakeGetClient{
		responses: map[string]json.RawMessage{
			buildSourceKey("SUI-USDT", from, to89Days, 0, 100): posData,
		},
	}
	source := NewBingXClosedPositionSource(fakeClient)

	_, err := source.FetchClosedPositions(ctx, "SUI-USDT", from, to89Days)
	if err != nil {
		t.Fatalf("89 Tage sollte OK sein, aber bekam Fehler: %v", err)
	}

	// 89 Tage + 1 ms.
	to89DaysPlus1 := from + 89*24*60*60*1000 + 1
	_, err = source.FetchClosedPositions(ctx, "SUI-USDT", from, to89DaysPlus1)
	if err == nil {
		t.Fatalf("89 Tage + 1 ms sollte Fehler sein, aber keiner geliefert")
	}
	if !strings.Contains(err.Error(), "89") {
		t.Errorf("Fehler sollte '89' erwähnen: %v", err)
	}
}

func TestBingXClosedPositionSource_ContextCancelled_NoClientCall(t *testing.T) {
	// Abgebrochener Context → errors.Is(err, context.Canceled), kein API-Aufruf.
	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	callCount := 0
	fakeClient := &fakeGetClientWithCapture{
		fn: func(_ context.Context, _ string, _ []bingx.QueryParam) (json.RawMessage, error) {
			callCount++
			return nil, nil
		},
	}

	// Context canceln BEVOR wir die Source aufrufen.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	source := NewBingXClosedPositionSource(fakeClient)
	_, err := source.FetchClosedPositions(ctx, "XRP-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet für cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Fehler sollte context.Canceled sein, aber: %T: %v", err, err)
	}
	if callCount != 0 {
		t.Errorf("Client.Get sollte nicht aufgerufen werden bei cancelled context, aber callCount = %d", callCount)
	}
}

func TestBingXClosedPositionSource_PagingUpperLimit(t *testing.T) {
	// Wenn die API endlos volle Seiten liefert (Fehler oder Schleife), muss die Source
	// nach einer festen Obergrenze (z.B. 1000 Seiten) mit Fehler abbrechen, nicht infinit loopen.
	// Dieser Test verifiziert, dass der Call nach ~1000 Aufrufen abbricht.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	pageCallCount := 0
	fakeClient := &fakeGetClientWithCapture{
		fn: func(_ context.Context, _ string, _ []bingx.QueryParam) (json.RawMessage, error) {
			pageCallCount++

			// Der Fake bricht NIE selbst mit Fehler ab. Nur Schutz des Testlaufs:
			// nach 5000 Aufrufen Context abbrechen, danach t.Fatal.
			if pageCallCount == 5000 {
				cancel()
			}
			if pageCallCount > 5100 {
				t.Fatalf("Source hat nach %d Aufrufen kein Seitenlimit erzwungen", pageCallCount)
			}

			// Sonst: immer 100 Einträge (full page) → paging fortsetzt sich.
			positions := make([]string, 100)
			for i := 0; i < 100; i++ {
				posID := fmt.Sprintf("9000000000030%05d", i)
				positions[i] = fmt.Sprintf(
					`{"positionId":"%s","symbol":"ADA-USDT","positionSide":"LONG","isolated":true,"openTime":%d,"updateTime":%d,"avgPrice":"1","avgClosePrice":"1","realisedProfit":"1","netProfit":"1","positionAmt":"1","closePositionAmt":"1","leverage":1,"closeAllPositions":true,"positionCommission":"-1","totalFunding":"-1"}`,
					posID, from+1000, from+2000,
				)
			}
			return json.RawMessage(fmt.Sprintf(`{"positionHistory":[%s]}`, strings.Join(positions, ","))), nil
		},
	}

	source := NewBingXClosedPositionSource(fakeClient)
	_, err := source.FetchClosedPositions(ctx, "ADA-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet für endless paging (sollte durch Limit abgebrochen werden)")
	}

	var pageLimit *domainposition.PageLimitExceededError
	if !errors.As(err, &pageLimit) {
		t.Fatalf("Fehler = %v, erwartet *PageLimitExceededError", err)
	}
	if pageLimit.Pages != maxClosedPositionPages || pageLimit.Symbol == "" {
		t.Errorf("PageLimitExceededError = %+v, erwartet Pages=%d und Symbol", pageLimit, maxClosedPositionPages)
	}
	// Der Fehler muss von der Source stammen, nicht vom abgebrochenen Context.
	if errors.Is(err, context.Canceled) {
		t.Errorf("Fehler stammt aus Context-Abbruch statt aus Seitenlimit: %v", err)
	}
	if pageCallCount > 1001 {
		t.Errorf("pageCallCount = %d, erwartet <= 1001 (Limit 1000 + 1 Toleranz)", pageCallCount)
	}
	t.Logf("Paging abgebrochen nach %d Aufrufen, Fehler: %v", pageCallCount, err)
}

func TestBingXClosedPositionSource_PageIndexStartsFromZero(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	positionHistoryData := fixtureDataClosed(t, "positionHistory.page0.json")

	// Capture the call to verify pageIndex=0 is used, not pageIndex=1.
	callCount := 0
	fakeClient := &fakeGetClientWithCapture{
		fn: func(_ context.Context, _ string, params []bingx.QueryParam) (json.RawMessage, error) {
			callCount++
			for _, p := range params {
				if p.Name == "pageIndex" && p.Value != "0" {
					t.Errorf("pageIndex = %s, erwartet 0", p.Value)
				}
			}
			return positionHistoryData, nil
		},
	}

	source := NewBingXClosedPositionSource(fakeClient)
	_, err := source.FetchClosedPositions(ctx, "SUI-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if callCount == 0 {
		t.Errorf("Erwarteter mindestens ein Aufruf zu fakeClient.Get")
	}
}

// fakeGetClientWithCapture erlaubt benutzerdefinierte Logik pro Aufruf.
type fakeGetClientWithCapture struct {
	fn func(ctx context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error)
}

func (f *fakeGetClientWithCapture) Get(ctx context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	return f.fn(ctx, path, params)
}
