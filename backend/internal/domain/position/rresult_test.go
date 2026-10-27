package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func TestRResult(t *testing.T) {
	// Risiko = |100 - 90| * 2 = 20 Einheiten.
	risk := position.PositionRisk{PositionID: "1", EntryPrice: "100", InitialStopLoss: "90"}
	testCases := []struct {
		name, netProfit, amount, expected string
		risk                              position.PositionRisk
	}{
		{"+3R", "60", "2", "3", risk},
		{"-1R", "-20", "2", "-1", risk},
		{"0R", "0", "2", "0", risk},
		{"Nachkommastellen", "30", "2", "1.5", risk},
		{"gerundet auf 4 Stellen", "20", "6", "0.3333", position.PositionRisk{EntryPrice: "100", InitialStopLoss: "90"}},
		{"SHORT (SL über Entry)", "60", "2", "3", position.PositionRisk{EntryPrice: "100", InitialStopLoss: "110"}},
		{"exakt bei feinen Dezimalstellen", "0.0003", "1", "3", position.PositionRisk{EntryPrice: "1.2345", InitialStopLoss: "1.2344"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := position.RResult(testCase.netProfit, testCase.amount, testCase.risk)
			if err != nil || got != testCase.expected {
				t.Errorf("got %q err %v, erwartet %q", got, err, testCase.expected)
			}
		})
	}
}

func TestRResult_SLAufEntryIstFehler(t *testing.T) {
	_, err := position.RResult("10", "1", position.PositionRisk{EntryPrice: "100", InitialStopLoss: "100"})
	var atEntry *position.StopLossAtEntryError
	if !errors.As(err, &atEntry) {
		t.Fatalf("erwartet *StopLossAtEntryError, war %v", err)
	}
}

func TestRResult_UngueltigeZahlIstFehler(t *testing.T) {
	risk := position.PositionRisk{EntryPrice: "100", InitialStopLoss: "90"}
	if _, err := position.RResult("abc", "1", risk); err == nil {
		t.Error("ungültiger netProfit muss Fehler liefern")
	}
	if _, err := position.RResult("10", "abc", risk); err == nil {
		t.Error("ungültige positionAmount muss Fehler liefern")
	}
}
