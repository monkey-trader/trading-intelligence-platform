package position_test

import (
	"context"
	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Erwartet: OpenPositionEntry.Levels []position.RLevel (aus position.RLevels(side, risk.EntryPrice, risk.InitialStopLoss));
// nil bei unbekanntem Risiko.

func TestLevels_AusGespeichertemRisiko(t *testing.T) {
	// Entry 0.9600, SL 0.9000 -> 2R 1.08, 3R 1.14 (kein 1R).
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9600")}})

	want, err := position.RLevels(position.Long, "0.9600", "0.9000")
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Levels) != 2 {
		t.Fatalf("Levels: %+v", entry.Levels)
	}
	for index, level := range entry.Levels {
		if level.CoincidesWithTakeProfit != nil {
			t.Errorf("Level %d: coincidesWithTakeProfit muss in OpenPosition nil sein", index)
		}
		if level != want[index] {
			t.Errorf("Level %d = %+v, erwartet %+v", index, level, want[index])
		}
	}
	if entry.Levels[0].Multiple != 2 || entry.Levels[0].Price != "1.08" || entry.Levels[1].Multiple != 3 || entry.Levels[1].Price != "1.14" {
		t.Errorf("Preise: %+v", entry.Levels)
	}
}

func TestLevels_LeerBeiUnbekanntemRisiko(t *testing.T) {
	entry := executeRisk(t, &fakeRiskRepo{}, &fakeCandles{}, &fakeOrders{})
	if len(entry.Levels) != 0 {
		t.Errorf("Levels: %+v", entry.Levels)
	}
}

// Regression TASK-14: levels[1] war früher 2R, ist jetzt 3R. Die BE-Erkennung muss das 2R-Level
// per position.LevelAt(levels, 2) suchen (Reached2RAt/BreakEvenTriggerReached).
func shortCandle(openTime int64, low string) chart.Candle {
	return chart.Candle{Symbol: "SUI-USDT", Interval: chart.Interval15m, OpenTime: openTime, Open: "1", High: "1", Low: low, Close: "1", Volume: "1"}
}

func TestBreakEven_NutztZweiRLevelNichtDreiR(t *testing.T) {
	// LONG: Entry 0.96, SL 0.90 -> 2R 1.08, 3R 1.14. High 1.08 berührt 2R, nicht 3R.
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	entry := executeRisk(t, risks, &fakeCandles{candles: []chart.Candle{candle(1_791_600_000_000, "1.08")}}, &fakeOrders{})
	if entry.BreakEvenTriggerReached == nil || !*entry.BreakEvenTriggerReached ||
		entry.Reached2RAt == nil || *entry.Reached2RAt != 1_791_600_000_000 {
		t.Errorf("LONG 2R bei High 1.08 muss erreicht sein: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
	}
}

func TestBreakEven_ShortErkennung(t *testing.T) {
	// SHORT: Entry 0.96, SL 1.02 -> 1R=0.06, 2R 0.84, 3R 0.78.
	asShort := func(p *position.OpenPosition) { p.Side = position.Short }
	run := func(low string) appposition.OpenPositionEntry {
		risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "1.0200"}}}
		candles := &fakeCandles{candles: []chart.Candle{shortCandle(1_791_500_000_000, "0.95"), shortCandle(1_791_600_000_000, low)}}
		useCase := appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(openPosition(asShort)), &fakeOrders{}).
			WithRisk(risks, candles, func() int64 { return riskNow })
		overview, err := useCase.Execute(context.Background())
		if err != nil || len(overview.Positions) != 1 {
			t.Fatalf("err=%v", err)
		}
		return overview.Positions[0]
	}
	if entry := run("0.84"); entry.BreakEvenTriggerReached == nil || !*entry.BreakEvenTriggerReached ||
		entry.Reached2RAt == nil || *entry.Reached2RAt != 1_791_600_000_000 {
		t.Errorf("SHORT Low 0.84 = 2R genau: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
	}
	if entry := run("0.8401"); entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached || entry.Reached2RAt != nil {
		t.Errorf("SHORT Low 0.8401 = 2R nicht erreicht: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
	}
}
