package position_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/position"
)

// Fälle portiert aus BingXOpenPositionSource.test.ts und UnconfiguredOpenPositionSource.test.ts.

const (
	openPositionsPath   = "/openApi/swap/v2/user/positions"
	requestStartMillis  = 1_791_800_000_000
	responseReceivedAdd = 250
)

// fakeClock ist eine Uhr, die nur der Test verstellt. Sicher für -race.
type fakeClock struct {
	mutex   sync.Mutex
	current time.Time
}

func newFakeClock(startMillis int64) *fakeClock {
	return &fakeClock{current: time.UnixMilli(startMillis)}
}

func (clock *fakeClock) Now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.current
}

func (clock *fakeClock) Set(next time.Time) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.current = next
}

func (clock *fakeClock) Advance(delta time.Duration) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.current = clock.current.Add(delta)
}

type getCall struct {
	ctx    context.Context
	path   string
	params []bingx.QueryParam
}

// fakeGetClient liefert eine vorbereitete data-Antwort oder einen Fehler.
type fakeGetClient struct {
	data      json.RawMessage
	err       error
	onRequest func()
	calls     []getCall
}

func (client *fakeGetClient) Get(ctx context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	client.calls = append(client.calls, getCall{ctx: ctx, path: path, params: slices.Clone(params)})
	if client.onRequest != nil {
		client.onRequest()
	}
	if client.err != nil {
		return nil, client.err
	}
	return client.data, nil
}

func fixtureData(t *testing.T, name string) json.RawMessage {
	t.Helper()
	// Pfad relativ zum Paketordner internal/infrastructure/position.
	content, err := os.ReadFile("../../../../fixtures/bingx/" + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return envelope["data"]
}

func TestBingXOpenPositionSource_ErfuelltPort(_ *testing.T) {
	var _ interfaces.OpenPositionSource = (*position.BingXOpenPositionSource)(nil)
	var _ interfaces.OpenPositionSource = position.UnconfiguredOpenPositionSource{}
	var _ interfaces.OpenPositionSource = (*position.CachingSource)(nil)
}

func TestBingXOpenPositionSource_RuftEndpunktOhneParameterAuf(t *testing.T) {
	type contextKey struct{}
	client := &fakeGetClient{data: fixtureData(t, "openPositions.json")}
	source := position.NewBingXOpenPositionSource(client, newFakeClock(requestStartMillis))
	ctx := context.WithValue(context.Background(), contextKey{}, "anfrage-7")

	if _, err := source.FetchOpenPositions(ctx); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("%d Aufrufe, erwartet 1", len(client.calls))
	}
	call := client.calls[0]
	if call.path != openPositionsPath {
		t.Errorf("Pfad = %q, erwartet %q", call.path, openPositionsPath)
	}
	if len(call.params) != 0 {
		t.Errorf("Parameter = %v, erwartet keine", call.params)
	}
	if call.ctx == nil || call.ctx.Value(contextKey{}) != "anfrage-7" {
		t.Errorf("Client hat den Context des Aufrufers nicht bekommen")
	}
}

func TestBingXOpenPositionSource_LiefertGemapptePositionen(t *testing.T) {
	client := &fakeGetClient{data: fixtureData(t, "openPositions.json")}
	source := position.NewBingXOpenPositionSource(client, newFakeClock(requestStartMillis))

	snapshot, err := source.FetchOpenPositions(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(snapshot.Positions) != 2 {
		t.Fatalf("%d Positionen, erwartet 2", len(snapshot.Positions))
	}
	if snapshot.Positions[0].PositionID != "9100000000000000001" || snapshot.Positions[1].PositionID != "9100000000000000002" {
		t.Errorf("IDs = %q, %q", snapshot.Positions[0].PositionID, snapshot.Positions[1].PositionID)
	}
	if snapshot.Positions[0].LiquidationPrice != "0.865" || snapshot.Positions[1].RealisedProfit != "-0.5187" {
		t.Errorf("Felder passen nicht: %+v", snapshot.Positions)
	}
}

func TestBingXOpenPositionSource_LeereListe(t *testing.T) {
	for _, testCase := range []struct {
		name string
		data json.RawMessage
	}{
		{"empty-Fixture", fixtureData(t, "openPositions.empty.json")},
		{"data null", json.RawMessage("null")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := position.NewBingXOpenPositionSource(&fakeGetClient{data: testCase.data}, newFakeClock(requestStartMillis))

			snapshot, err := source.FetchOpenPositions(context.Background())
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(snapshot.Positions) != 0 || snapshot.FetchedAt != requestStartMillis {
				t.Errorf("Snapshot = %+v, erwartet leer mit FetchedAt %d", snapshot, requestStartMillis)
			}
		})
	}
}

func TestBingXOpenPositionSource_FetchedAtNachDerAntwort(t *testing.T) {
	clock := newFakeClock(requestStartMillis)
	client := &fakeGetClient{
		data:      fixtureData(t, "openPositions.json"),
		onRequest: func() { clock.Set(time.UnixMilli(requestStartMillis + responseReceivedAdd)) },
	}
	source := position.NewBingXOpenPositionSource(client, clock)

	snapshot, err := source.FetchOpenPositions(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if snapshot.FetchedAt != requestStartMillis+responseReceivedAdd {
		t.Errorf("FetchedAt = %d, erwartet %d", snapshot.FetchedAt, requestStartMillis+responseReceivedAdd)
	}
}

func TestBingXOpenPositionSource_NeueZeitJeAufruf(t *testing.T) {
	clock := newFakeClock(requestStartMillis)
	source := position.NewBingXOpenPositionSource(&fakeGetClient{data: json.RawMessage("[]")}, clock)

	first, _ := source.FetchOpenPositions(context.Background())
	clock.Advance(15 * time.Second)
	second, _ := source.FetchOpenPositions(context.Background())

	if first.FetchedAt != requestStartMillis || second.FetchedAt != requestStartMillis+15_000 {
		t.Errorf("FetchedAt = %d, %d; erwartet %d, %d", first.FetchedAt, second.FetchedAt,
			int64(requestStartMillis), int64(requestStartMillis+15_000))
	}
}

func TestBingXOpenPositionSource_Fehlerabbildung(t *testing.T) {
	testCases := []struct {
		name           string
		clientError    error
		expectedReason domainposition.UnavailableReason
	}{
		{"RateLimitError", &bingx.RateLimitError{Path: openPositionsPath, AttemptCount: 2}, domainposition.ReasonRateLimited},
		{"NetworkError", &bingx.NetworkError{Path: openPositionsPath, AttemptCount: 2, CauseName: "*url.Error"}, domainposition.ReasonUnreachable},
		{"APIError", &bingx.APIError{Code: 100001, Msg: "signature verification failed", Path: openPositionsPath}, domainposition.ReasonUpstreamError},
		{"HTTPError", &bingx.HTTPError{Status: 500, Path: openPositionsPath}, domainposition.ReasonUpstreamError},
		{"UnexpectedResponseError", &bingx.UnexpectedResponseError{Message: "Antwort ohne erwartete Form"}, domainposition.ReasonInvalidResponse},
		{"PositionDomainError", &domainposition.InvalidPositionIDError{Value: "abc"}, domainposition.ReasonInvalidResponse},
		{"gewrappter RateLimitError", fmt.Errorf("Hülle: %w", &bingx.RateLimitError{Path: openPositionsPath}), domainposition.ReasonRateLimited},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := position.NewBingXOpenPositionSource(&fakeGetClient{err: testCase.clientError}, newFakeClock(requestStartMillis))

			snapshot, err := source.FetchOpenPositions(context.Background())

			var unavailable *domainposition.OpenPositionsUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("Fehler = %v (%T), erwartet *OpenPositionsUnavailableError", err, err)
			}
			if unavailable.Reason != testCase.expectedReason {
				t.Errorf("Reason = %q, erwartet %q", unavailable.Reason, testCase.expectedReason)
			}
			if unavailable.Cause != testCase.clientError {
				t.Errorf("Cause = %v, erwartet den Client-Fehler", unavailable.Cause)
			}
			if !errors.Is(err, testCase.clientError) {
				t.Errorf("errors.Is findet den Client-Fehler nicht")
			}
			if len(snapshot.Positions) != 0 || snapshot.FetchedAt != 0 {
				t.Errorf("Snapshot = %+v, erwartet leer bei Fehler", snapshot)
			}
		})
	}
}

func TestBingXOpenPositionSource_MappingFehlerWerdenInvalidResponse(t *testing.T) {
	testCases := []struct {
		name        string
		data        json.RawMessage
		assertCause func(*testing.T, error)
	}{
		{
			name: "unerwartete Antwortform",
			data: json.RawMessage(`{"positions":[]}`),
			assertCause: func(t *testing.T, cause error) {
				var unexpected *bingx.UnexpectedResponseError
				if !errors.As(cause, &unexpected) {
					t.Errorf("Cause = %v (%T), erwartet *bingx.UnexpectedResponseError", cause, cause)
				}
			},
		},
		{
			name: "fehlendes data",
			data: nil,
			assertCause: func(t *testing.T, cause error) {
				var unexpected *bingx.UnexpectedResponseError
				if !errors.As(cause, &unexpected) {
					t.Errorf("Cause = %v (%T), erwartet *bingx.UnexpectedResponseError", cause, cause)
				}
			},
		},
		{
			name: "Domain-Fehler aus dem Mapping",
			data: json.RawMessage(`[{"positionId":"abc","symbol":"SUI-USDT","positionSide":"LONG","isolated":true,` +
				`"positionAmt":"500","avgPrice":"0.96","markPrice":"1.01","leverage":10,"initialMargin":"48",` +
				`"margin":"60.5","positionValue":"505","unrealizedProfit":"25","realisedProfit":"-0.24",` +
				`"pnlRatio":"0.52","liquidationPrice":0.865,"createTime":1791500000000,"updateTime":1791600000000}]`),
			assertCause: func(t *testing.T, cause error) {
				var domainError domainposition.PositionDomainError
				if !errors.As(cause, &domainError) {
					t.Errorf("Cause = %v (%T), erwartet PositionDomainError", cause, cause)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := position.NewBingXOpenPositionSource(&fakeGetClient{data: testCase.data}, newFakeClock(requestStartMillis))

			_, err := source.FetchOpenPositions(context.Background())

			var unavailable *domainposition.OpenPositionsUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("Fehler = %v (%T), erwartet *OpenPositionsUnavailableError", err, err)
			}
			if unavailable.Reason != domainposition.ReasonInvalidResponse {
				t.Errorf("Reason = %q, erwartet invalid-response", unavailable.Reason)
			}
			testCase.assertCause(t, unavailable.Cause)
		})
	}
}

func TestBingXOpenPositionSource_UnbekannteFehlerGehenDurch(t *testing.T) {
	unknown := errors.New("Programmierfehler")
	source := position.NewBingXOpenPositionSource(&fakeGetClient{err: unknown}, newFakeClock(requestStartMillis))

	_, err := source.FetchOpenPositions(context.Background())

	if !errors.Is(err, unknown) {
		t.Fatalf("Fehler = %v, erwartet den unbekannten Fehler", err)
	}
	var unavailable *domainposition.OpenPositionsUnavailableError
	if errors.As(err, &unavailable) {
		t.Errorf("unbekannter Fehler wurde zu %q übersetzt; erwartet unverändert (später 500 internal)", unavailable.Reason)
	}
}

func TestUnconfiguredOpenPositionSource(t *testing.T) {
	for range 2 {
		snapshot, err := position.UnconfiguredOpenPositionSource{}.FetchOpenPositions(context.Background())

		var unavailable *domainposition.OpenPositionsUnavailableError
		if !errors.As(err, &unavailable) {
			t.Fatalf("Fehler = %v (%T), erwartet *OpenPositionsUnavailableError", err, err)
		}
		if unavailable.Reason != domainposition.ReasonNotConfigured {
			t.Errorf("Reason = %q, erwartet not-configured", unavailable.Reason)
		}
		if unavailable.Cause != nil {
			t.Errorf("Cause = %v, erwartet nil", unavailable.Cause)
		}
		if len(snapshot.Positions) != 0 {
			t.Errorf("Snapshot = %+v, erwartet leer", snapshot)
		}
	}
}
