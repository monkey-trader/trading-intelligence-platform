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
	want := []position.RLevel{
		{Multiple: 1, Price: "1.02", Percent: "0.0625"},
		{Multiple: 2, Price: "1.08", Percent: "0.125"},
		{Multiple: 3, Price: "1.14", Percent: "0.1875"},
	}
	if len(result.Levels) != 3 {
		t.Fatalf("Levels = %+v", result.Levels)
	}
	for index := range want {
		if result.Levels[index] != want[index] {
			t.Errorf("Level %d = %+v", index+1, result.Levels[index])
		}
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
	if len(result.Levels) != 3 || result.Levels[0].Price != "1.0325" {
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
