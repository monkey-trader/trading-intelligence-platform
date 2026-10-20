package position_test

import (
	"context"
	"errors"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet: (*GetOpenPositionsUseCase).WithRisk(risks interfaces.PositionRiskRepository,
// candles chartinterfaces.CandleRepository, now func() int64) *GetOpenPositionsUseCase
// und neue Felder an OpenPositionEntry: InitialStopLoss, RiskDistance, CurrentRMultiple *string,
// BreakEvenTriggerReached *bool, Reached2RAt *int64. Kerzen werden mit chart.Interval5m gelesen.

type fakeRiskRepo struct {
	stored   map[string]position.PositionRisk
	saveErr  error
	getErr   error
	saveCall []position.PositionRisk
}

func (repo *fakeRiskRepo) SaveIfAbsent(_ context.Context, risk position.PositionRisk) (bool, error) {
	repo.saveCall = append(repo.saveCall, risk)
	if repo.saveErr != nil {
		return false, repo.saveErr
	}
	if _, exists := repo.stored[risk.PositionID]; exists {
		return false, nil
	}
	if repo.stored == nil {
		repo.stored = map[string]position.PositionRisk{}
	}
	repo.stored[risk.PositionID] = risk
	return true, nil
}

func (repo *fakeRiskRepo) Get(_ context.Context, positionID string) (position.PositionRisk, bool, error) {
	if repo.getErr != nil {
		return position.PositionRisk{}, false, repo.getErr
	}
	risk, found := repo.stored[positionID]
	return risk, found, nil
}

var _ interfaces.PositionRiskRepository = (*fakeRiskRepo)(nil)

type fakeCandles struct {
	candles []chart.Candle
	err     error
}

func (repo *fakeCandles) UpsertCandles(context.Context, []chart.Candle) error { return nil }
func (repo *fakeCandles) ListCandles(context.Context, string, chart.CandleInterval, int) ([]chart.Candle, error) {
	return repo.candles, repo.err
}

const riskPositionID = "9100000000000000001"

func stopOrder(trigger string) chart.ProtectiveOrder {
	return chart.ProtectiveOrder{OrderID: "1", PositionID: riskPositionID, Symbol: "SUI-USDT", Kind: chart.StopLoss, TriggerPrice: trigger, Quantity: "500", Status: "NEW"}
}

func candle(openTime int64, high string) chart.Candle {
	return chart.Candle{Symbol: "SUI-USDT", Interval: chart.Interval5m, OpenTime: openTime, Open: "1", High: high, Low: "0.9", Close: "1", Volume: "1"}
}

func executeRisk(t *testing.T, risks *fakeRiskRepo, candles *fakeCandles, orders *fakeOrders) appposition.OpenPositionEntry {
	t.Helper()
	useCase := appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(openPosition(nil)), orders).
		WithRisk(risks, candles, func() int64 { return 1_791_700_000_000 })
	overview, err := useCase.Execute(context.Background())
	if err != nil || len(overview.Positions) != 1 {
		t.Fatalf("err=%v positions=%d", err, len(overview.Positions))
	}
	return overview.Positions[0]
}

func TestRisk_GespeichertesRisikoBefuelltFelder(t *testing.T) {
	// Entry 0.9600, initialer SL 0.9000 -> 1R = 0.06, 2R-Preis 1.08, Mark 1.0100 -> 0.8333R.
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000", FirstSeenAt: 1}}}
	candles := &fakeCandles{candles: []chart.Candle{candle(1_791_500_000_000, "1.07"), candle(1_791_600_000_000, "1.08")}}
	entry := executeRisk(t, risks, candles, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9600")}})

	if entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" ||
		entry.RiskDistance == nil || *entry.RiskDistance != "0.06" ||
		entry.CurrentRMultiple == nil || *entry.CurrentRMultiple != "0.8333" {
		t.Errorf("Felder: %+v", entry)
	}
	if entry.BreakEvenTriggerReached == nil || !*entry.BreakEvenTriggerReached ||
		entry.Reached2RAt == nil || *entry.Reached2RAt != 1_791_600_000_000 {
		t.Errorf("2R: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
	}
}

func TestRisk_2RNichtErreicht(t *testing.T) {
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	entry := executeRisk(t, risks, &fakeCandles{candles: []chart.Candle{candle(1_791_500_000_000, "1.0799")}}, &fakeOrders{})
	if entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached || entry.Reached2RAt != nil {
		t.Errorf("2R: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
	}
}

func TestRisk_ErstesSehenSpeichertAktuellenSL(t *testing.T) {
	risks := &fakeRiskRepo{}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9000")}})
	saved := risks.stored[riskPositionID]
	if saved.InitialStopLoss != "0.9000" || saved.EntryPrice != "0.9600" || saved.FirstSeenAt != 1_791_700_000_000 {
		t.Errorf("gespeichert: %+v", saved)
	}
	if entry.RiskDistance == nil || *entry.RiskDistance != "0.06" {
		t.Errorf("RiskDistance = %v", entry.RiskDistance)
	}
}

func TestRisk_SLGleichEntryWirdNichtGespeichert(t *testing.T) {
	risks := &fakeRiskRepo{}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9600")}})
	if len(risks.saveCall) != 0 {
		t.Errorf("gespeichert: %+v", risks.saveCall)
	}
	if entry.InitialStopLoss != nil || entry.RiskDistance != nil || entry.CurrentRMultiple != nil ||
		entry.BreakEvenTriggerReached != nil || entry.Reached2RAt != nil {
		t.Errorf("Risiko muss unbekannt (nil) sein: %+v", entry)
	}
}

func TestRisk_SLAufFalscherSeiteUndOhneSLNichtGespeichert(t *testing.T) {
	for name, orders := range map[string][]chart.ProtectiveOrder{
		"LONG SL über Entry": {stopOrder("1.0000")},
		"ohne SL":            nil,
	} {
		t.Run(name, func(t *testing.T) {
			risks := &fakeRiskRepo{}
			entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: orders})
			if len(risks.saveCall) != 0 || entry.RiskDistance != nil || entry.CurrentRMultiple != nil {
				t.Errorf("saved=%v entry=%+v", risks.saveCall, entry)
			}
		})
	}
}

func TestRisk_OrdersNichtLadbarStuerztNichtAb(t *testing.T) {
	risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{err: errors.New("429")})
	if entry.StopLoss != nil || entry.InitialStopLoss == nil || entry.CurrentRMultiple == nil {
		t.Errorf("gespeichertes Risiko soll trotzdem gelten: %+v", entry)
	}
}

func TestRisk_RepositoryOderKerzenfehlerLassenFelderLeerOhneFehler(t *testing.T) {
	t.Run("Get-Fehler", func(t *testing.T) {
		entry := executeRisk(t, &fakeRiskRepo{getErr: errors.New("db")}, &fakeCandles{}, &fakeOrders{})
		if entry.RiskDistance != nil {
			t.Errorf("%+v", entry)
		}
	})
	t.Run("Kerzenfehler: R-Werte bleiben, 2R unbekannt", func(t *testing.T) {
		risks := &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
		entry := executeRisk(t, risks, &fakeCandles{err: errors.New("db")}, &fakeOrders{})
		if entry.CurrentRMultiple == nil || entry.BreakEvenTriggerReached != nil || entry.Reached2RAt != nil {
			t.Errorf("%+v", entry)
		}
	})
}
