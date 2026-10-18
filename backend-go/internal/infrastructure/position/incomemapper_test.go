package position

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// incomeFixture liest das data-Array (die Fixture enthält nur das Array).
func incomeFixture(t *testing.T) json.RawMessage {
	t.Helper()
	content, err := os.ReadFile(fixtureDir + "income.page.json")
	if err != nil {
		t.Fatalf("Fixture: %v", err)
	}
	return content
}

func TestMapIncomeResponse_Fixture(t *testing.T) {
	entries, err := mapIncomeResponse(incomeFixture(t))
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("Anzahl %d, erwartet 5", len(entries))
	}
	first := entries[0]
	if first.TranID != "8000000000000000001" || first.Symbol != "ONDO-USDT" || first.IncomeType != "TRADING_FEE" ||
		first.Amount != "-0.0256" || first.Time != 1780000000500 || first.Asset != "USDT" || first.TradeID != "7000000000000000001" {
		t.Errorf("erster Eintrag falsch: %+v", first)
	}
	if !strings.Contains(first.Raw, `"tranId": "8000000000000000001"`) && !strings.Contains(first.Raw, `"tranId":"8000000000000000001"`) {
		t.Errorf("Raw enthält den Rohtext nicht: %q", first.Raw)
	}
	if entries[2].TradeID != "" {
		t.Errorf("leere tradeId muss leer bleiben, ist %q", entries[2].TradeID)
	}
}

func TestMapIncomeResponse_LeerFaelle(t *testing.T) {
	for _, data := range []string{`null`, `[]`} {
		entries, err := mapIncomeResponse(json.RawMessage(data))
		if err != nil || len(entries) != 0 {
			t.Errorf("%s: entries=%v err=%v, erwartet leer ohne Fehler", data, entries, err)
		}
	}
}

func TestMapIncomeResponse_AbweichendeAntworten(t *testing.T) {
	const entryOK = `{"symbol":"ONDO-USDT","incomeType":"REALIZED_PNL","income":"1.7","asset":"USDT","time":1780018017000,"tranId":"1","tradeId":"2"}`
	tests := []struct{ name, data string }{
		{"kein Array", `{"a":1}`},
		{"Eintrag kein Objekt", `[1]`},
		{"symbol fehlt", `[{"incomeType":"X","income":"1","asset":"USDT","time":1,"tranId":"1","tradeId":""}]`},
		{"incomeType fehlt", `[{"symbol":"A-B","income":"1","asset":"USDT","time":1,"tranId":"1","tradeId":""}]`},
		{"income fehlt", `[{"symbol":"A-B","incomeType":"X","asset":"USDT","time":1,"tranId":"1","tradeId":""}]`},
		{"income als Zahl statt String", `[{"symbol":"A-B","incomeType":"X","income":1.5,"asset":"USDT","time":1,"tranId":"1","tradeId":""}]`},
		{"time als String", `[{"symbol":"A-B","incomeType":"X","income":"1","asset":"USDT","time":"1","tranId":"1","tradeId":""}]`},
		{"tranId fehlt", `[{"symbol":"A-B","incomeType":"X","income":"1","asset":"USDT","time":1,"tradeId":""}]`},
		{"ein schlechter Eintrag nach einem guten", `[` + entryOK + `,{"x":1}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mapIncomeResponse(json.RawMessage(tt.data))
			var unexpectedErr *bingx.UnexpectedResponseError
			if !errors.As(err, &unexpectedErr) {
				t.Fatalf("erwartet *bingx.UnexpectedResponseError, bekam %v", err)
			}
		})
	}
}
