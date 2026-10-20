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

// Erwartet: (*GetOpenPositionsUseCase).WithStopHistory(history interfaces.StopOrderHistorySource) *GetOpenPositionsUseCase
// (nach WithRisk aufgerufen; nutzt dessen now-Funktion). Port:
//   interfaces.StopOrderHistorySource.FetchStopOrders(ctx, symbol string, fromMs, toMs int64) ([]position.StopOrderRecord, error)
// Fabrik-Position: SUI-USDT LONG, Entry 0.9600, openTime 1791500000000.

const (
	historyNow       = int64(1_791_700_000_000)
	historyOpenTime  = int64(1_791_500_000_000)
	negativeCacheTTL = int64(10 * 60 * 1000)
	errorBackoff     = int64(60 * 1000)
)

type fakeHistory struct {
	records []position.StopOrderRecord
	err     error
	calls   []historyCall
}

type historyCall struct {
	symbol       string
	fromMs, toMs int64
}

func (source *fakeHistory) FetchStopOrders(_ context.Context, symbol string, fromMs, toMs int64) ([]position.StopOrderRecord, error) {
	source.calls = append(source.calls, historyCall{symbol, fromMs, toMs})
	return source.records, source.err
}

var _ interfaces.StopOrderHistorySource = (*fakeHistory)(nil)

type historyRig struct {
	risks   *fakeRiskRepo
	history *fakeHistory
	orders  *fakeOrders
	clock   *int64
	useCase *appposition.GetOpenPositionsUseCase
}

func newHistoryRig(orders []chart.ProtectiveOrder, records []position.StopOrderRecord) *historyRig {
	current := historyNow
	rig := &historyRig{risks: &fakeRiskRepo{}, history: &fakeHistory{records: records}, orders: &fakeOrders{orders: orders}, clock: &current}
	rig.useCase = appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(openPosition(nil)), rig.orders).
		WithRisk(rig.risks, &fakeCandles{}, func() int64 { return current }).
		WithStopHistory(rig.history)
	return rig
}

func (rig *historyRig) run(t *testing.T) appposition.OpenPositionEntry {
	t.Helper()
	overview, err := rig.useCase.Execute(context.Background())
	if err != nil || len(overview.Positions) != 1 {
		t.Fatalf("err=%v positions=%d", err, len(overview.Positions))
	}
	return overview.Positions[0]
}

func record(positionID, trigger string, time int64) position.StopOrderRecord {
	return position.StopOrderRecord{PositionID: positionID, TriggerPrice: trigger, Time: time}
}

func TestStopHistory_BreakEvenSLHolsUrspruenglichenSLAusHistorie(t *testing.T) {
	// aktueller SL 0.9600 = Entry (Break-even) -> Historie: frühester Verlust-SL 0.9000.
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9600")}, []position.StopOrderRecord{
		record(riskPositionID, "0.9600", 3_000),
		record(riskPositionID, "0.9000", 1_000),
		record(riskPositionID, "0.9300", 2_000),
		record("9100000000000000999", "0.5000", 500), // fremde Position, früher, wird ignoriert
	})
	entry := rig.run(t)

	saved, ok := rig.risks.stored[riskPositionID]
	if !ok || saved.InitialStopLoss != "0.9000" || saved.EntryPrice != "0.9600" || saved.FirstSeenAt != historyNow {
		t.Fatalf("gespeichert: %+v", saved)
	}
	if entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" || entry.RiskDistance == nil || *entry.RiskDistance != "0.06" {
		t.Errorf("Felder: %+v", entry)
	}
	if len(rig.history.calls) != 1 {
		t.Fatalf("Aufrufe: %+v", rig.history.calls)
	}
	if call := rig.history.calls[0]; call.symbol != "SUI-USDT" || call.fromMs != historyOpenTime || call.toMs != historyNow {
		t.Errorf("Aufruf: %+v", call)
	}
}

func TestStopHistory_OhneAktuellenSLWirdHistorieGefragt(t *testing.T) {
	rig := newHistoryRig(nil, []position.StopOrderRecord{record(riskPositionID, "0.9000", 1_000)})
	entry := rig.run(t)
	if entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" || len(rig.history.calls) != 1 {
		t.Errorf("entry=%+v calls=%d", entry, len(rig.history.calls))
	}
}

func TestStopHistory_GueltigerAktuellerSLFragtHistorieNicht(t *testing.T) {
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9000")}, []position.StopOrderRecord{record(riskPositionID, "0.8000", 1)})
	entry := rig.run(t)
	if len(rig.history.calls) != 0 {
		t.Errorf("Historie darf nicht gefragt werden: %+v", rig.history.calls)
	}
	if rig.risks.stored[riskPositionID].InitialStopLoss != "0.9000" || entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" {
		t.Errorf("gespeichert: %+v", rig.risks.stored)
	}
}

func TestStopHistory_GespeichertesRisikoFragtHistorieNicht(t *testing.T) {
	rig := newHistoryRig(nil, nil)
	rig.risks.stored = map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}
	rig.run(t)
	rig.run(t)
	if len(rig.history.calls) != 0 {
		t.Errorf("Aufrufe: %+v", rig.history.calls)
	}
}

func TestStopHistory_WiederholterAufrufNachErfolgFragtNichtErneut(t *testing.T) {
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9600")}, []position.StopOrderRecord{record(riskPositionID, "0.9000", 1)})
	rig.run(t)
	rig.run(t)
	if len(rig.history.calls) != 1 {
		t.Errorf("Aufrufe: %d", len(rig.history.calls))
	}
}

func TestStopHistory_NegativCache(t *testing.T) {
	// Historie hat nichts Brauchbares (nur Gewinnseite / fremde Position).
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9600")}, []position.StopOrderRecord{
		record(riskPositionID, "1.0500", 1), record("9100000000000000999", "0.9", 1),
	})
	first := rig.run(t)
	if first.InitialStopLoss != nil || len(rig.risks.saveCall) != 0 || len(rig.history.calls) != 1 {
		t.Fatalf("erster Lauf: %+v saved=%v calls=%d", first, rig.risks.saveCall, len(rig.history.calls))
	}
	*rig.clock = historyNow + negativeCacheTTL - 60_000 // knapp innerhalb von 10 min
	rig.run(t)
	if len(rig.history.calls) != 1 {
		t.Errorf("innerhalb des Negativ-Caches nicht erneut fragen: %d Aufrufe", len(rig.history.calls))
	}
	*rig.clock = historyNow + negativeCacheTTL + 1 // nach 10 min wieder
	rig.run(t)
	if len(rig.history.calls) != 2 {
		t.Errorf("nach Ablauf erneut fragen: %d Aufrufe", len(rig.history.calls))
	}
}

func TestStopHistory_QuellenfehlerBackoffStattDauerfeuer(t *testing.T) {
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9600")}, nil)
	rig.history.err = errors.New("BingX nicht erreichbar")
	entry := rig.run(t)
	if entry.InitialStopLoss != nil || entry.RiskDistance != nil || len(rig.risks.saveCall) != 0 {
		t.Fatalf("Risiko muss unbekannt bleiben: %+v", entry)
	}
	// Direkt danach (Poll im Sekundentakt) wird BingX nicht erneut belastet.
	rig.history.err = nil
	rig.history.records = []position.StopOrderRecord{record(riskPositionID, "0.9000", 1)}
	*rig.clock = historyNow + errorBackoff - 1
	rig.run(t)
	if len(rig.history.calls) != 1 {
		t.Fatalf("innerhalb des Fehler-Backoffs nicht erneut fragen: %d Aufrufe", len(rig.history.calls))
	}
	// Nach dem Backoff wird neu gefragt und kann gelingen.
	*rig.clock = historyNow + errorBackoff + 1
	entry = rig.run(t)
	if len(rig.history.calls) != 2 || entry.InitialStopLoss == nil || *entry.InitialStopLoss != "0.9000" {
		t.Errorf("calls=%d entry=%+v", len(rig.history.calls), entry)
	}
}

func TestStopHistory_SaveFehlerStuerztNichtAb(t *testing.T) {
	rig := newHistoryRig([]chart.ProtectiveOrder{stopOrder("0.9600")}, []position.StopOrderRecord{record(riskPositionID, "0.9000", 1)})
	rig.risks.saveErr = errors.New("db weg")
	entry := rig.run(t)
	if entry.InitialStopLoss != nil {
		t.Errorf("Risiko muss unbekannt sein: %+v", entry)
	}
}

func TestStopHistory_OhneQuelleVerhaeltSichWieBisher(t *testing.T) {
	risks := &fakeRiskRepo{}
	entry := executeRisk(t, risks, &fakeCandles{}, &fakeOrders{orders: []chart.ProtectiveOrder{stopOrder("0.9600")}})
	if entry.InitialStopLoss != nil || len(risks.saveCall) != 0 {
		t.Errorf("entry=%+v", entry)
	}
}
