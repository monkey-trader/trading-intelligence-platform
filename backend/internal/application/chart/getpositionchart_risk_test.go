package chart_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	appchart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	positioninterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet: (*GetPositionChartUseCase).WithRisk(risks positioninterfaces.PositionRiskRepository,
// now func() int64) *GetPositionChartUseCase; PositionChart.Levels []position.RLevel (nie nil, leer bei unbekanntem Risiko).

type fakeRisks struct {
	stored map[string]position.PositionRisk
	saved  []position.PositionRisk
}

func (repo *fakeRisks) SaveIfAbsent(_ context.Context, risk position.PositionRisk) (bool, error) {
	repo.saved = append(repo.saved, risk)
	if _, exists := repo.stored[risk.PositionID]; exists {
		return false, nil
	}
	if repo.stored == nil {
		repo.stored = map[string]position.PositionRisk{}
	}
	repo.stored[risk.PositionID] = risk
	return true, nil
}

func (repo *fakeRisks) Get(_ context.Context, positionID string) (position.PositionRisk, bool, error) {
	risk, found := repo.stored[positionID]
	return risk, found, nil
}

var _ positioninterfaces.PositionRiskRepository = (*fakeRisks)(nil)

func (f *fixture) executeWithRisk(risks *fakeRisks) (appchart.PositionChart, error) {
	logger := slog.New(slog.NewTextHandler(f.logs, nil))
	useCase := appchart.NewGetPositionChartUseCase(f.positions, f.orders, f.source, f.repo, logger).
		WithRisk(risks, func() int64 { return 42 })
	return useCase.Execute(context.Background(), defaultQuery())
}

func TestLevels_AusGespeichertemRisiko(t *testing.T) {
	f := newFixture()
	risks := &fakeRisks{stored: map[string]position.PositionRisk{suiPositionID: {PositionID: suiPositionID, EntryPrice: suiAvgOpenPrice, InitialStopLoss: "0.9000"}}}
	result, err := f.executeWithRisk(risks)
	if err != nil {
		t.Fatal(err)
	}
	// Nur 2R und 3R; TPs der Fixture (1.2, 1.35) liegen weit von 3R (1.14) -> false (gesetzt, nicht nil).
	if len(result.Levels) != 2 {
		t.Fatalf("Levels = %+v", result.Levels)
	}
	two, three := result.Levels[0], result.Levels[1]
	if two != (position.RLevel{Multiple: 2, Price: "1.08", Percent: "0.125"}) {
		t.Errorf("2R = %+v (coincidesWithTakeProfit muss nil sein)", two)
	}
	if three.Multiple != 3 || three.Price != "1.14" || three.Percent != "0.1875" ||
		three.CoincidesWithTakeProfit == nil || *three.CoincidesWithTakeProfit {
		t.Errorf("3R = %+v", three)
	}
}

func TestLevels_ErstesSehenSpeichertSL(t *testing.T) {
	f := newFixture() // Orders: SL 0.8875
	risks := &fakeRisks{}
	result, err := f.executeWithRisk(risks)
	if err != nil {
		t.Fatal(err)
	}
	saved := risks.stored[suiPositionID]
	if saved.InitialStopLoss != "0.8875" || saved.EntryPrice != suiAvgOpenPrice || saved.FirstSeenAt != 42 {
		t.Errorf("gespeichert: %+v", saved)
	}
	if len(result.Levels) != 2 || result.Levels[0].Price != "1.105" {
		t.Errorf("Levels = %+v", result.Levels)
	}
}

func TestLevels_RisikoUnbekanntLiefertLeereListe(t *testing.T) {
	f := newFixture()
	f.orders.orders = []chart.ProtectiveOrder{protective("1", chart.StopLoss, "0.9600", "500", suiPositionID)} // SL == Entry
	risks := &fakeRisks{}
	result, err := f.executeWithRisk(risks)
	if err != nil {
		t.Fatal(err)
	}
	if result.Levels == nil || len(result.Levels) != 0 || len(risks.saved) != 0 {
		t.Errorf("Levels = %#v, saved = %+v", result.Levels, risks.saved)
	}
}

func TestLevels_OrdersNichtLadbarStuerztNichtAb(t *testing.T) {
	// Bestehendes Verhalten (Fehler der Order-Source wird gemeldet) bleibt: kein Panic, Fehler durchgereicht.
	f := newFixture()
	f.orders.err = errors.New("429")
	risks := &fakeRisks{stored: map[string]position.PositionRisk{suiPositionID: {PositionID: suiPositionID, EntryPrice: suiAvgOpenPrice, InitialStopLoss: "0.9000"}}}
	if _, err := f.executeWithRisk(risks); err == nil {
		t.Error("Fehler der Order-Source muss durchgereicht werden")
	}
}

// Erwartet: (*GetPositionChartUseCase).WithInitialStopLossOverrides(overrides positioninterfaces.PositionInitialStopLossRepository)
// (nach WithRisk); der Override hat Vorrang, Level beruhen auf Override-Entry und -SL.

type fakeOverrideRepo struct {
	stored map[string]position.InitialStopLossOverride
}

func (repo *fakeOverrideRepo) Set(context.Context, position.InitialStopLossOverride) error {
	return nil
}
func (repo *fakeOverrideRepo) Delete(context.Context, string) error { return nil }
func (repo *fakeOverrideRepo) Get(_ context.Context, positionID string) (position.InitialStopLossOverride, bool, error) {
	override, found := repo.stored[positionID]
	return override, found, nil
}

var _ positioninterfaces.PositionInitialStopLossRepository = (*fakeOverrideRepo)(nil)

func TestLevels_OverrideHatVorrangUndNutztOverrideEntry(t *testing.T) {
	f := newFixture()
	risks := &fakeRisks{stored: map[string]position.PositionRisk{suiPositionID: {PositionID: suiPositionID, EntryPrice: suiAvgOpenPrice, InitialStopLoss: "0.9000"}}}
	overrides := &fakeOverrideRepo{stored: map[string]position.InitialStopLossOverride{
		suiPositionID: {PositionID: suiPositionID, EntryPrice: "1.0000", InitialStopLoss: "0.9000", SetAt: 1},
	}}
	logger := slog.New(slog.NewTextHandler(f.logs, nil))
	useCase := appchart.NewGetPositionChartUseCase(f.positions, f.orders, f.source, f.repo, logger).
		WithRisk(risks, func() int64 { return 42 }).WithInitialStopLossOverrides(overrides)

	result, err := useCase.Execute(context.Background(), defaultQuery())
	if err != nil || len(result.Levels) != 2 {
		t.Fatalf("err=%v levels=%+v", err, result.Levels)
	}
	if result.Levels[0].Price != "1.2" || result.Levels[1].Price != "1.3" {
		t.Errorf("Level müssen auf Entry 1.0000 / SL 0.9000 beruhen: %+v", result.Levels)
	}
}

// coincidesWithTakeProfit: nur beim 3R-Level (immer gesetzt, true/false), beim 2R-Level nil.
// Schwelle: |TP-Trigger - 3R-Preis| <= 0.1 % des Entry-Preises (inklusive Grenze), jede offene TP-Order zählt.
func TestLevels_CoincidesWithTakeProfit(t *testing.T) {
	// Entry 0.9600 -> Schwelle 0.00096. LONG: SL 0.9000 -> 3R 1.14. SHORT: SL 1.0200 -> 3R 0.78.
	tests := []struct {
		name string
		side position.PositionSide
		stop string
		take []string
		want bool
	}{
		{"LONG TP genau auf 3R", position.Long, "0.9000", []string{"1.1400"}, true},
		{"LONG TP genau an der Grenze oben", position.Long, "0.9000", []string{"1.14096"}, true},
		{"LONG TP genau an der Grenze unten", position.Long, "0.9000", []string{"1.13904"}, true},
		{"LONG TP knapp über der Grenze", position.Long, "0.9000", []string{"1.14097"}, false},
		{"LONG TP knapp unter der Grenze", position.Long, "0.9000", []string{"1.13903"}, false},
		{"LONG mehrere TPs, einer nahe", position.Long, "0.9000", []string{"1.2000", "1.1405"}, true},
		{"LONG mehrere TPs, keiner nahe", position.Long, "0.9000", []string{"1.2000", "1.3500"}, false},
		{"LONG kein TP", position.Long, "0.9000", nil, false},
		{"SHORT TP nahe 3R", position.Short, "1.0200", []string{"0.78078"}, true},
		{"SHORT TP knapp über der Grenze", position.Short, "1.0200", []string{"0.78097"}, false},
		{"SHORT kein TP", position.Short, "1.0200", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture()
			f.positions.positions[0].Side = test.side
			f.orders.orders = []chart.ProtectiveOrder{protective("1", chart.StopLoss, test.stop, "500", suiPositionID)}
			for index, trigger := range test.take {
				f.orders.orders = append(f.orders.orders, protective("tp"+string(rune('a'+index)), chart.TakeProfit, trigger, "250", suiPositionID))
			}
			risks := &fakeRisks{stored: map[string]position.PositionRisk{suiPositionID: {PositionID: suiPositionID, EntryPrice: suiAvgOpenPrice, InitialStopLoss: test.stop}}}
			result, err := f.executeWithRisk(risks)
			if err != nil || len(result.Levels) != 2 {
				t.Fatalf("err=%v levels=%+v", err, result.Levels)
			}
			if result.Levels[0].Multiple != 2 || result.Levels[0].CoincidesWithTakeProfit != nil {
				t.Errorf("2R muss nil sein: %+v", result.Levels[0])
			}
			three := result.Levels[1]
			if three.Multiple != 3 || three.CoincidesWithTakeProfit == nil || *three.CoincidesWithTakeProfit != test.want {
				t.Errorf("3R = %+v, erwartet coincides=%v", three, test.want)
			}
		})
	}
}
