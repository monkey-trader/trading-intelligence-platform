package position_test

import (
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Erwartet: OpenPositionEntry.Levels []position.RLevel (aus position.RLevels(side, risk.EntryPrice, risk.InitialStopLoss));
// nil bei unbekanntem Risiko.

func TestLevels_AusGespeichertemRisiko(t *testing.T) {
	// Entry 0.9600, SL 0.9000 -> 1R 1.02, 2R 1.08, 3R 1.14.
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9600")}})

	want, err := position.RLevels(position.Long, "0.9600", "0.9000")
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Levels) != 3 {
		t.Fatalf("Levels: %+v", entry.Levels)
	}
	for index, level := range entry.Levels {
		if level != want[index] {
			t.Errorf("Level %d = %+v, erwartet %+v", index, level, want[index])
		}
	}
	if entry.Levels[1].Multiple != 2 || entry.Levels[1].Price != "1.08" || entry.Levels[2].Price != "1.14" {
		t.Errorf("Preise: %+v", entry.Levels)
	}
}

func TestLevels_LeerBeiUnbekanntemRisiko(t *testing.T) {
	entry := executeRisk(t, &fakeRiskRepo{}, &fakeCandles{}, &fakeOrders{})
	if len(entry.Levels) != 0 {
		t.Errorf("Levels: %+v", entry.Levels)
	}
}
