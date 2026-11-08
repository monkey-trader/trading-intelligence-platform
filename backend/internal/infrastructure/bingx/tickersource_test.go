package bingx_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/price"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

type tickerClient struct {
	data   json.RawMessage
	err    error
	paths  []string
	params [][]bingx.QueryParam
}

func (client *tickerClient) Get(_ context.Context, path string, params []bingx.QueryParam) (json.RawMessage, error) {
	client.paths = append(client.paths, path)
	client.params = append(client.params, params)
	return client.data, client.err
}

type tickerClock struct{}

func (tickerClock) Now() time.Time { return time.UnixMilli(1_791_800_000_000) }

func tickerFixtureData(t *testing.T) json.RawMessage {
	t.Helper()
	content, err := os.ReadFile("../../../../fixtures/bingx/ticker.json")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed.Data
}

func TestTickerSourceMapsFixtureAndSkipsInvalidSymbols(t *testing.T) {
	client := &tickerClient{data: tickerFixtureData(t)}
	quotes, err := bingx.NewTickerSource(client, tickerClock{}).Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	expected := []price.Quote{
		{Symbol: "BTC-USDT", Price: "113250.5", Change24hPercent: "-0.45", At: 1_791_800_000_000},
		{Symbol: "SUI-USDT", Price: "3.2105", Change24hPercent: "1.30", At: 1_791_800_000_000},
	}
	if len(quotes) != len(expected) {
		t.Fatalf("Quotes = %+v, erwartet %+v (leerer Preis und ungültige Prozentzahl werden übersprungen)", quotes, expected)
	}
	for index := range expected {
		if quotes[index] != expected[index] {
			t.Errorf("Quote %d = %+v, erwartet %+v", index, quotes[index], expected[index])
		}
	}
}

func TestTickerSourceMakesOneRequestWithoutSymbol(t *testing.T) {
	client := &tickerClient{data: tickerFixtureData(t)}
	if _, err := bingx.NewTickerSource(client, tickerClock{}).Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(client.paths) != 1 || client.paths[0] != "/openApi/swap/v2/quote/ticker" {
		t.Fatalf("Requests = %v, erwartet genau einer auf /openApi/swap/v2/quote/ticker", client.paths)
	}
	for _, param := range client.params[0] {
		if param.Name == "symbol" {
			t.Errorf("symbol darf nicht gesetzt sein (alle Symbole in einem Request)")
		}
	}
}

func TestTickerSourceReturnsUnavailableOnClientError(t *testing.T) {
	client := &tickerClient{err: errors.New("code 100001: signature")}
	_, err := bingx.NewTickerSource(client, tickerClock{}).Fetch(context.Background())
	if !errors.Is(err, price.ErrPriceUnavailable) {
		t.Fatalf("Fehler = %v, erwartet ErrPriceUnavailable", err)
	}
}

func TestTickerSourceDataShapes(t *testing.T) {
	cases := []struct {
		name      string
		data      string
		wantCount int
		wantErr   bool
	}{
		{"leeres Array", `[]`, 0, false},
		{"null", `null`, 0, false},
		{"kein Array", `{"symbol":"BTC-USDT"}`, 0, true},
		{"Preis als Zahl statt String wird übersprungen", `[{"symbol":"A-USDT","lastPrice":1.5,"priceChangePercent":"1"}]`, 0, false},
		{"Preis null oder 0 wird übersprungen", `[{"symbol":"A-USDT","lastPrice":"0","priceChangePercent":"1"},{"symbol":"B-USDT","priceChangePercent":"1"}]`, 0, false},
		{"Symbol fehlt wird übersprungen", `[{"lastPrice":"1","priceChangePercent":"1"}]`, 0, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &tickerClient{data: json.RawMessage(testCase.data)}
			quotes, err := bingx.NewTickerSource(client, tickerClock{}).Fetch(context.Background())
			if testCase.wantErr {
				if !errors.Is(err, price.ErrPriceUnavailable) {
					t.Fatalf("Fehler = %v, erwartet ErrPriceUnavailable", err)
				}
				return
			}
			if err != nil || len(quotes) != testCase.wantCount {
				t.Fatalf("quotes=%+v err=%v, erwartet %d Quotes ohne Fehler", quotes, err, testCase.wantCount)
			}
		})
	}
}
