package chart_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
	chartinfra "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/chart"
)

// Erwartet: chartinfra.NewBingXStopOrderHistorySource(client bingx.GetClient) *BingXStopOrderHistorySource
// mit FetchStopOrders(ctx, symbol, fromMs, toMs) ([]position.StopOrderRecord, error).
// GET /openApi/swap/v2/trade/allOrders, Parameter symbol, startTime, endTime (je Fenster <= 7 Tage), limit.
// Fixture: fixtures/bingx/allOrders.page.json (MARKET, STOP_MARKET FILLED 79000.0, TAKE_PROFIT_MARKET, STOP_MARKET CANCELLED 78000.0).

const (
	allOrdersPath = "/openApi/swap/v2/trade/allOrders"
	sevenDaysMs   = int64(7 * 24 * 60 * 60 * 1000)
	historyFrom   = int64(1_790_000_000_000)
)

var _ interfaces.StopOrderHistorySource = (*chartinfra.BingXStopOrderHistorySource)(nil)

// pagedClient liefert beim ersten Aufruf die Fixture, danach leere Fenster.
type pagedClient struct {
	fakeGetClient
	first json.RawMessage
}

func (client *pagedClient) Get(ctx context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	data := client.first
	if len(client.calls) > 0 {
		data = json.RawMessage(`{"orders":[]}`)
	}
	client.data = data
	return client.fakeGetClient.Get(ctx, path, params)
}

func TestStopOrderHistory_MapptNurStopOrdersMitAllenStatus(t *testing.T) {
	client := &fakeGetClient{data: loadEnvelope(t, "allOrders.page.json").Data}
	records, err := chartinfra.NewBingXStopOrderHistorySource(client).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1000)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := []position.StopOrderRecord{
		{PositionID: "9100000000000000001", TriggerPrice: "79000.0", Time: 1790817200000},
		{PositionID: "9100000000000000001", TriggerPrice: "78000.0", Time: 1790824400000},
	}
	if len(records) != len(want) {
		t.Fatalf("records = %+v", records)
	}
	for index := range want {
		if records[index] != want[index] {
			t.Errorf("record %d = %+v, erwartet %+v", index, records[index], want[index])
		}
	}
}

func TestStopOrderHistory_AnfrageParameter(t *testing.T) {
	client := &fakeGetClient{data: json.RawMessage(`{"orders":[]}`)}
	_, err := chartinfra.NewBingXStopOrderHistorySource(client).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1000)
	if err != nil || len(client.calls) != 1 {
		t.Fatalf("err=%v calls=%d", err, len(client.calls))
	}
	call := client.calls[0]
	symbol, _ := paramValue(call, "symbol")
	start, _ := paramValue(call, "startTime")
	end, _ := paramValue(call, "endTime")
	if call.path != allOrdersPath || symbol != "BTC-USDT" || start != "1790000000000" || end != "1790000001000" {
		t.Errorf("call = %+v", call)
	}
}

func TestStopOrderHistory_LeereAntwortenErgebenLeereListe(t *testing.T) {
	for name, data := range map[string]json.RawMessage{
		"null": json.RawMessage(`null`), "orders leer": json.RawMessage(`{"orders":[]}`), "orders null": json.RawMessage(`{"orders":null}`),
	} {
		t.Run(name, func(t *testing.T) {
			records, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{data: data}).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
			if err != nil || len(records) != 0 {
				t.Errorf("records=%+v err=%v", records, err)
			}
		})
	}
}

func TestStopOrderHistory_Fenstergrenzen(t *testing.T) {
	tests := []struct {
		name      string
		span      int64
		wantCalls int
	}{
		{"genau 7 Tage: ein Fenster", sevenDaysMs, 1},
		{"7 Tage + 1 ms: zwei Fenster", sevenDaysMs + 1, 2},
		{"15 Tage: drei Fenster", 15 * 24 * 60 * 60 * 1000, 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeGetClient{data: json.RawMessage(`{"orders":[]}`)}
			to := historyFrom + test.span
			if _, err := chartinfra.NewBingXStopOrderHistorySource(client).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, to); err != nil {
				t.Fatal(err)
			}
			if len(client.calls) != test.wantCalls {
				t.Fatalf("Aufrufe = %d, erwartet %d", len(client.calls), test.wantCalls)
			}
			var previousEnd int64
			for index, call := range client.calls {
				startText, _ := paramValue(call, "startTime")
				endText, _ := paramValue(call, "endTime")
				var start, end int64
				if _, err := fmt.Sscan(startText, &start); err != nil {
					t.Fatal(err)
				}
				if _, err := fmt.Sscan(endText, &end); err != nil {
					t.Fatal(err)
				}
				if end-start > sevenDaysMs {
					t.Errorf("Fenster %d zu groß: %d ms", index, end-start)
				}
				if index == 0 && start != historyFrom {
					t.Errorf("erstes Fenster beginnt bei %d", start)
				}
				if index > 0 && (start > previousEnd+1 || start <= previousEnd-sevenDaysMs) {
					t.Errorf("Fenster %d nicht lückenlos: start %d, vorheriges Ende %d", index, start, previousEnd)
				}
				if index == len(client.calls)-1 && end != to {
					t.Errorf("letztes Fenster endet bei %d, erwartet %d", end, to)
				}
				previousEnd = end
			}
		})
	}
}

func TestStopOrderHistory_SammeltAlleFensterUndVerliertPositionIDNicht(t *testing.T) {
	client := &pagedClient{first: loadEnvelope(t, "allOrders.page.json").Data}
	records, err := chartinfra.NewBingXStopOrderHistorySource(client).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+10*24*60*60*1000)
	if err != nil || len(records) != 2 || len(client.calls) != 2 {
		t.Fatalf("records=%+v err=%v calls=%d", records, err, len(client.calls))
	}
	if records[0].PositionID != "9100000000000000001" {
		t.Errorf("positionID verfälscht: %q", records[0].PositionID)
	}
}

func TestStopOrderHistory_PositionIDAlsStringUndOhneID(t *testing.T) {
	data := json.RawMessage(`{"orders":[
		{"symbol":"BTC-USDT","orderId":1,"type":"STOP_MARKET","status":"NEW","stopPrice":"79000","time":5,"positionID":"9100000000000000001"},
		{"symbol":"BTC-USDT","orderId":2,"type":"STOP_MARKET","status":"NEW","stopPrice":"78000","time":6,"positionID":0}]}`)
	records, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{data: data}).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
	if err != nil || len(records) < 1 || records[0] != (position.StopOrderRecord{PositionID: "9100000000000000001", TriggerPrice: "79000", Time: 5}) {
		t.Errorf("records=%+v err=%v", records, err)
	}
	for _, record := range records {
		if record.PositionID == "0" || record.PositionID == "" {
			t.Errorf("Order ohne Position darf nicht erscheinen: %+v", record)
		}
	}
}

func TestStopOrderHistory_Fehler(t *testing.T) {
	t.Run("code != 0", func(t *testing.T) {
		apiErr := &bingx.APIError{Code: 109400, Msg: "bad range"}
		_, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{err: apiErr}).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
		var target *bingx.APIError
		if err == nil || !errors.As(err, &target) || target.Code != 109400 {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("HTTP 429", func(t *testing.T) {
		_, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{err: &bingx.RateLimitError{}}).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
		var target *bingx.RateLimitError
		if !errors.As(err, &target) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("Antwort ist kein Objekt", func(t *testing.T) {
		_, err := chartinfra.NewBingXStopOrderHistorySource(&fakeGetClient{data: json.RawMessage(`[1]`)}).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+1)
		if err == nil {
			t.Error("Fehler erwartet")
		}
	})
	t.Run("Fehler im zweiten Fenster bricht ab", func(t *testing.T) {
		client := &fakeGetClient{data: json.RawMessage(`{"orders":[]}`)}
		client.onRequest = func() {
			if len(client.calls) == 2 {
				client.err = errors.New("boom")
			}
		}
		_, err := chartinfra.NewBingXStopOrderHistorySource(client).FetchStopOrders(context.Background(), "BTC-USDT", historyFrom, historyFrom+10*24*60*60*1000)
		if err == nil {
			t.Error("Fehler erwartet")
		}
	})
}
