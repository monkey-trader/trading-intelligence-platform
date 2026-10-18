package chart_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
	chartinfra "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/chart"
)

// Fixtures: fixtures/bingx/klines.*.json und openOrders.json (README dort beschreibt die Eigenheiten).

const (
	klinesPath = "/openApi/swap/v3/quote/klines"
	ordersPath = "/openApi/swap/v2/trade/openOrders"
	startMs    = int64(1_791_800_000_000)
)

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func loadEnvelope(t *testing.T, name string) envelope {
	t.Helper()
	content, err := os.ReadFile("../../../../fixtures/bingx/" + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	var parsed envelope
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return parsed
}

type fakeClock struct{ current time.Time }

func (clock *fakeClock) Now() time.Time { return clock.current }

type getCall struct {
	path   string
	params []bingx.QueryParam
}

type fakeGetClient struct {
	data      json.RawMessage
	err       error
	onRequest func()
	calls     []getCall
}

func (client *fakeGetClient) Get(_ context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	client.calls = append(client.calls, getCall{path: path, params: slices.Clone(params)})
	if client.onRequest != nil {
		client.onRequest()
	}
	return client.data, client.err
}

func paramValue(call getCall, name string) (string, bool) {
	for _, param := range call.params {
		if param.Name == name {
			return param.Value, true
		}
	}
	return "", false
}

func reasonOf(t *testing.T, err error) position.UnavailableReason {
	t.Helper()
	var unavailable *chart.DataUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("erwartet *DataUnavailableError, erhalten %T: %v", err, err)
	}
	return unavailable.Reason
}

var (
	_ chartinterfaces.CandleSource          = (*chartinfra.BingXCandleSource)(nil)
	_ chartinterfaces.ProtectiveOrderSource = (*chartinfra.BingXProtectiveOrderSource)(nil)
)

// ---- Klines ----

func TestMapKlines_FixtureWirdAufsteigendSortiert(t *testing.T) {
	candles, err := chartinfra.MapKlines(loadEnvelope(t, "klines.sample.json").Data, "BTC-USDT", chart.Interval15m)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(candles) != 8 {
		t.Fatalf("erwartet 8 Kerzen, erhalten %d", len(candles))
	}
	for index := 1; index < len(candles); index++ {
		if candles[index].OpenTime <= candles[index-1].OpenTime {
			t.Fatalf("nicht streng aufsteigend an Index %d", index)
		}
	}
	// Die neueste Kerze (erster Eintrag der absteigenden Antwort) steht zuletzt.
	last := candles[len(candles)-1]
	expected := chart.Candle{Symbol: "BTC-USDT", Interval: chart.Interval15m, OpenTime: 1790819100000,
		Open: "83557.1", High: "83570.7", Low: "83464.8", Close: "83478.2", Volume: "7.5886"}
	if last != expected {
		t.Errorf("letzte Kerze = %+v, erwartet %+v", last, expected)
	}
}

func TestMapKlines_Randfaelle(t *testing.T) {
	candle := func(openTime int64, closePrice string) string {
		encoded, _ := json.Marshal(map[string]any{"open": "1", "high": "2", "low": "0.5", "close": closePrice, "volume": "3", "time": openTime})
		return string(encoded)
	}
	t.Run("Duplikate gleicher Zeit ergeben eine Kerze", func(t *testing.T) {
		data := json.RawMessage("[" + candle(2000, "1.5") + "," + candle(1000, "1.1") + "," + candle(2000, "1.5") + "]")
		candles, err := chartinfra.MapKlines(data, "X-USDT", chart.Interval5m)
		if err != nil || len(candles) != 2 || candles[0].OpenTime != 1000 || candles[1].OpenTime != 2000 {
			t.Errorf("candles = %+v, err = %v", candles, err)
		}
	})
	for _, empty := range []string{"[]", "null", ""} {
		t.Run("leere Antwort "+empty, func(t *testing.T) {
			candles, err := chartinfra.MapKlines(json.RawMessage(empty), "X-USDT", chart.Interval5m)
			if err != nil || candles == nil || len(candles) != 0 {
				t.Errorf("erwartet leere nicht-nil Liste ohne Fehler: %v, %v", candles, err)
			}
		})
	}
	invalid := map[string]string{
		"kein Array":              `{"a":1}`,
		"Müll":                    `"abc"`,
		"time als String":         `[{"open":"1","high":"2","low":"0.5","close":"1","volume":"3","time":"1000"}]`,
		"time fehlt":              `[{"open":"1","high":"2","low":"0.5","close":"1","volume":"3"}]`,
		"time null":               `[{"open":"1","high":"2","low":"0.5","close":"1","volume":"3","time":null}]`,
		"time null oder negativ":  `[{"open":"1","high":"2","low":"0.5","close":"1","volume":"3","time":-5}]`,
		"open fehlt":              `[{"high":"2","low":"0.5","close":"1","volume":"3","time":1000}]`,
		"close fehlt":             `[{"open":"1","high":"2","low":"0.5","volume":"3","time":1000}]`,
		"high keine Zahl":         `[{"open":"1","high":"abc","low":"0.5","close":"1","volume":"3","time":1000}]`,
		"volume fehlt":            `[{"open":"1","high":"2","low":"0.5","close":"1","time":1000}]`,
		"Preis als number":        `[{"open":1,"high":"2","low":"0.5","close":"1","volume":"3","time":1000}]`,
		"Eintrag ist kein Objekt": `[1]`,
	}
	for name, raw := range invalid {
		t.Run("ungültig: "+name, func(t *testing.T) {
			_, err := chartinfra.MapKlines(json.RawMessage(raw), "X-USDT", chart.Interval5m)
			if reason := reasonOf(t, err); reason != position.ReasonInvalidResponse {
				t.Errorf("Reason = %q, erwartet invalid-response", reason)
			}
		})
	}
}

func TestBingXCandleSource_Request(t *testing.T) {
	client := &fakeGetClient{data: loadEnvelope(t, "klines.sample.json").Data}
	source := chartinfra.NewBingXCandleSource(client)

	candles, err := source.FetchCandles(context.Background(), "BTC-USDT", chart.Interval1h, 1000)
	if err != nil || len(candles) != 8 {
		t.Fatalf("candles = %d, err = %v", len(candles), err)
	}
	if len(client.calls) != 1 || client.calls[0].path != klinesPath {
		t.Fatalf("calls = %+v", client.calls)
	}
	for name, expected := range map[string]string{"symbol": "BTC-USDT", "interval": "1h", "limit": "1000"} {
		if value, found := paramValue(client.calls[0], name); !found || value != expected {
			t.Errorf("Parameter %s = %q (gefunden: %v), erwartet %q", name, value, found, expected)
		}
	}
	if _, hasEndTime := paramValue(client.calls[0], "endTime"); hasEndTime {
		t.Error("endTime darf nicht gesetzt sein (neueste Kerzen gewünscht)")
	}
	if candles[0].Symbol != "BTC-USDT" || candles[0].Interval != chart.Interval1h {
		t.Errorf("Symbol/Intervall nicht übernommen: %+v", candles[0])
	}
}

func TestBingXCandleSource_Fehler(t *testing.T) {
	intervalError := loadEnvelope(t, "klines.error.interval.json")
	symbolError := loadEnvelope(t, "klines.error.symbolNotFound.json")
	testCases := []struct {
		name           string
		clientErr      error
		expectedReason position.UnavailableReason
		check          func(error) bool
	}{
		{"109400 Parameter (HTTP 200)", &bingx.APIError{Code: intervalError.Code, Msg: intervalError.Msg, Path: klinesPath},
			position.ReasonUpstreamError, func(err error) bool {
				var target *chart.KlineParameterError
				return errors.As(err, &target) && target.Code == 109400
			}},
		{"109425 Symbol unbekannt", &bingx.APIError{Code: symbolError.Code, Msg: symbolError.Msg, Path: klinesPath},
			position.ReasonUpstreamError, func(err error) bool {
				var target *chart.SymbolNotFoundError
				return errors.As(err, &target) && target.Symbol == "FOO-USDT"
			}},
		{"anderer code != 0", &bingx.APIError{Code: 100001, Msg: "x", Path: klinesPath},
			position.ReasonUpstreamError, func(err error) bool { var target *bingx.APIError; return errors.As(err, &target) }},
		{"HTTP 429", &bingx.RateLimitError{Path: klinesPath, AttemptCount: 3},
			position.ReasonRateLimited, func(err error) bool { var target *bingx.RateLimitError; return errors.As(err, &target) }},
		{"Netzwerk", &bingx.NetworkError{Path: klinesPath, AttemptCount: 3, CauseName: "timeout"},
			position.ReasonUnreachable, func(_ error) bool { return true }},
		{"HTTP 500", &bingx.HTTPError{Status: 500, Path: klinesPath},
			position.ReasonUpstreamError, func(_ error) bool { return true }},
		{"unerwartete Form", &bingx.UnexpectedResponseError{Message: "x"},
			position.ReasonInvalidResponse, func(_ error) bool { return true }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := chartinfra.NewBingXCandleSource(&fakeGetClient{err: testCase.clientErr})

			_, err := source.FetchCandles(context.Background(), "FOO-USDT", chart.Interval15m, 300)

			if reason := reasonOf(t, err); reason != testCase.expectedReason {
				t.Errorf("Reason = %q, erwartet %q", reason, testCase.expectedReason)
			}
			if !testCase.check(err) {
				t.Errorf("Ursache nicht über errors.As erreichbar: %v", err)
			}
		})
	}
}

// ---- openOrders ----

func ordersData(t *testing.T) json.RawMessage { return loadEnvelope(t, "openOrders.json").Data }

func TestMapProtectiveOrders_Fixture(t *testing.T) {
	orders, err := chartinfra.MapProtectiveOrders(ordersData(t))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	// Die LIMIT-Order (positionID 0) wird ignoriert.
	expected := []chart.ProtectiveOrder{
		{OrderID: "9200000000000000001", PositionID: "9100000000000000001", Symbol: "SUI-USDT", Kind: chart.StopLoss,
			TriggerPrice: "0.8875", Quantity: "500", Status: "NEW"},
		{OrderID: "9200000000000000002", PositionID: "9100000000000000001", Symbol: "SUI-USDT", Kind: chart.TakeProfit,
			TriggerPrice: "1.2000", Quantity: "250", Status: "NEW"},
		{OrderID: "9200000000000000003", PositionID: "9100000000000000001", Symbol: "SUI-USDT", Kind: chart.TakeProfit,
			TriggerPrice: "1.3500", Quantity: "250", Status: "NEW"},
	}
	if !slices.Equal(orders, expected) {
		t.Errorf("orders = %+v\nerwartet %+v", orders, expected)
	}
}

func TestMapProtectiveOrders_IdsVerlustfreiAlsNumberUndString(t *testing.T) {
	// 9100000000000000001 liegt über 2^53: ein float64 würde auf ...000 runden.
	order := func(orderID, positionID, orderType, stopPrice string) string {
		return `{"symbol":"SUI-USDT","orderId":` + orderID + `,"positionID":` + positionID +
			`,"type":"` + orderType + `","origQty":"1","stopPrice":"` + stopPrice + `","status":"NEW"}`
	}
	data := json.RawMessage(`{"orders":[` + order("9200000000000000007", "9100000000000000001", "STOP_MARKET", "0.5") +
		`,` + order(`"9200000000000000008"`, `"9100000000000000001"`, "TAKE_PROFIT_MARKET", "2") + `]}`)

	orders, err := chartinfra.MapProtectiveOrders(data)

	if err != nil || len(orders) != 2 {
		t.Fatalf("orders = %+v, err = %v", orders, err)
	}
	if orders[0].OrderID != "9200000000000000007" || orders[0].PositionID != "9100000000000000001" {
		t.Errorf("number-IDs verfälscht: %+v", orders[0])
	}
	if orders[1].OrderID != "9200000000000000008" || orders[1].PositionID != "9100000000000000001" {
		t.Errorf("String-IDs verfälscht: %+v", orders[1])
	}
}

func TestMapProtectiveOrders_NichtRelevanteOrdersWerdenIgnoriert(t *testing.T) {
	base := `"symbol":"SUI-USDT","orderId":1,"origQty":"1","stopPrice":"1","status":"NEW"`
	data := json.RawMessage(`{"orders":[` +
		`{` + base + `,"type":"LIMIT","positionID":0},` +
		`{` + base + `,"type":"LIMIT","positionID":9100000000000000001},` +
		`{` + base + `,"type":"MARKET","positionID":9100000000000000001},` +
		`{` + base + `,"type":"STOP_MARKET","positionID":0},` +
		`{` + base + `,"type":"TRAILING_STOP_MARKET","positionID":9100000000000000001}` +
		`]}`)

	orders, err := chartinfra.MapProtectiveOrders(data)

	if err != nil || orders == nil || len(orders) != 0 {
		t.Errorf("erwartet leere nicht-nil Liste ohne Fehler: %+v, %v", orders, err)
	}
}

func TestMapProtectiveOrders_LeereAntworten(t *testing.T) {
	for _, raw := range []string{`{"orders":[]}`, `{"orders":null}`, `{}`, `null`, ``} {
		t.Run(raw, func(t *testing.T) {
			orders, err := chartinfra.MapProtectiveOrders(json.RawMessage(raw))
			if err != nil || orders == nil || len(orders) != 0 {
				t.Errorf("orders = %+v, err = %v", orders, err)
			}
		})
	}
}

func TestMapProtectiveOrders_UngueltigeAntworten(t *testing.T) {
	const prefix = `{"orders":[{"symbol":"SUI-USDT","orderId":1,"positionID":9100000000000000001,"status":"NEW",`
	invalid := map[string]string{
		"kein Objekt":           `[1]`,
		"Müll":                  `"abc"`,
		"orders kein Array":     `{"orders":5}`,
		"stopPrice fehlt":       prefix + `"type":"STOP_MARKET","origQty":"1"}]}`,
		"stopPrice leer":        prefix + `"type":"TAKE_PROFIT_MARKET","origQty":"1","stopPrice":""}]}`,
		"stopPrice keine Zahl":  prefix + `"type":"STOP_MARKET","origQty":"1","stopPrice":"abc"}]}`,
		"origQty fehlt":         prefix + `"type":"STOP_MARKET","stopPrice":"1"}]}`,
		"orderId fehlt":         `{"orders":[{"symbol":"SUI-USDT","positionID":9100000000000000001,"type":"STOP_MARKET","origQty":"1","stopPrice":"1"}]}`,
		"symbol fehlt":          `{"orders":[{"orderId":1,"positionID":9100000000000000001,"type":"STOP_MARKET","origQty":"1","stopPrice":"1"}]}`,
		"Eintrag kein Objekt":   `{"orders":[1]}`,
		"positionID Fließkomma": `{"orders":[{"symbol":"S-USDT","orderId":1,"positionID":9.1e18,"type":"STOP_MARKET","origQty":"1","stopPrice":"1"}]}`,
	}
	for name, raw := range invalid {
		t.Run(name, func(t *testing.T) {
			_, err := chartinfra.MapProtectiveOrders(json.RawMessage(raw))
			if err == nil {
				t.Fatal("erwartet Fehler")
			}
			if reason := reasonOf(t, err); reason != position.ReasonInvalidResponse {
				t.Errorf("Reason = %q, erwartet invalid-response", reason)
			}
		})
	}
}

func TestMapProtectiveOrders_TriggerPreisNullIstDomainFehler(t *testing.T) {
	data := json.RawMessage(`{"orders":[{"symbol":"S-USDT","orderId":1,"positionID":9100000000000000001,` +
		`"type":"STOP_MARKET","origQty":"1","stopPrice":"0.0000","status":"NEW"}]}`)

	_, err := chartinfra.MapProtectiveOrders(data)

	var notPositive *chart.TriggerPriceMustBePositiveError
	if !errors.As(err, &notPositive) {
		t.Errorf("erwartet *TriggerPriceMustBePositiveError, erhalten %v", err)
	}
}

func TestBingXProtectiveOrderSource(t *testing.T) {
	clock := &fakeClock{current: time.UnixMilli(startMs)}
	client := &fakeGetClient{data: ordersData(t), onRequest: func() { clock.current = time.UnixMilli(startMs + 250) }}
	source := chartinfra.NewBingXProtectiveOrderSource(client, clock)

	snapshot, err := source.FetchProtectiveOrders(context.Background())

	if err != nil || len(snapshot.Orders) != 3 {
		t.Fatalf("snapshot = %+v, err = %v", snapshot, err)
	}
	if len(client.calls) != 1 || client.calls[0].path != ordersPath {
		t.Errorf("calls = %+v", client.calls)
	}
	// Wie bei den offenen Positionen: Zeitpunkt nach der Antwort.
	if snapshot.FetchedAt != startMs+250 {
		t.Errorf("FetchedAt = %d, erwartet %d", snapshot.FetchedAt, startMs+250)
	}
}

func TestBingXProtectiveOrderSource_Fehler(t *testing.T) {
	testCases := []struct {
		name      string
		clientErr error
		expected  position.UnavailableReason
	}{
		{"HTTP 429", &bingx.RateLimitError{Path: ordersPath, AttemptCount: 3}, position.ReasonRateLimited},
		{"code != 0", &bingx.APIError{Code: 100001, Msg: "x", Path: ordersPath}, position.ReasonUpstreamError},
		{"Netzwerk", &bingx.NetworkError{Path: ordersPath, AttemptCount: 3, CauseName: "timeout"}, position.ReasonUnreachable},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := chartinfra.NewBingXProtectiveOrderSource(&fakeGetClient{err: testCase.clientErr}, &fakeClock{})
			_, err := source.FetchProtectiveOrders(context.Background())
			if reason := reasonOf(t, err); reason != testCase.expected {
				t.Errorf("Reason = %q, erwartet %q", reason, testCase.expected)
			}
		})
	}
	t.Run("Müll von BingX", func(t *testing.T) {
		source := chartinfra.NewBingXProtectiveOrderSource(&fakeGetClient{data: json.RawMessage(`"müll"`)}, &fakeClock{})
		_, err := source.FetchProtectiveOrders(context.Background())
		if reason := reasonOf(t, err); reason != position.ReasonInvalidResponse {
			t.Errorf("Reason = %q, erwartet invalid-response", reason)
		}
	})
}
