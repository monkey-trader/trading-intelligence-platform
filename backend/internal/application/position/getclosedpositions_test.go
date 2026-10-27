package position_test

import (
	"context"
	"errors"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

type fakeClosedLister struct {
	positions []position.ClosedPosition
	err       error
	got       position.ClosedPositionQuery
}

func (lister *fakeClosedLister) ListClosedPositions(_ context.Context, query position.ClosedPositionQuery) ([]position.ClosedPosition, error) {
	lister.got = query
	return lister.positions, lister.err
}

// readOnlyRisks erlaubt nur Get; SaveIfAbsent meldet einen Fehler (Schreibschutz).
type readOnlyRisks struct {
	t      *testing.T
	stored map[string]position.PositionRisk
	getErr error
}

func (repo *readOnlyRisks) SaveIfAbsent(context.Context, position.PositionRisk) (bool, error) {
	repo.t.Error("GetClosedPositions darf nie SaveIfAbsent aufrufen")
	return false, nil
}

func (repo *readOnlyRisks) Get(_ context.Context, id string) (position.PositionRisk, bool, error) {
	risk, found := repo.stored[id]
	return risk, found, repo.getErr
}

func closedOf(id string, side position.PositionSide, netProfit string) position.ClosedPosition {
	return position.ClosedPosition{PositionID: id, Symbol: "SUI-USDT", Side: side, AvgOpenPrice: "100",
		PositionAmount: "2", NetProfit: netProfit, FullyClosed: true, CloseTime: 2000, OpenTime: 1000}
}

func runClosed(t *testing.T, positions []position.ClosedPosition, stored map[string]position.PositionRisk) []appposition.ClosedPositionView {
	t.Helper()
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	views, err := appposition.NewGetClosedPositionsUseCase(&fakeClosedLister{positions: positions},
		&readOnlyRisks{t: t, stored: stored}).Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return views
}

func strOrNil(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

func TestGetClosedPositions_RLevelsUndRResult(t *testing.T) {
	testCases := []struct {
		name                 string
		side                 position.PositionSide
		stopLoss, netProfit  string
		wantR, wantBE, wantT string
	}{
		{"LONG +3R", position.Long, "90", "60", "3", "120", "130"},
		{"LONG -1R", position.Long, "90", "-20", "-1", "120", "130"},
		{"LONG Nachkommastellen", position.Long, "90", "30", "1.5", "120", "130"},
		{"SHORT +3R", position.Short, "110", "60", "3", "80", "70"},
		{"SHORT -1R", position.Short, "110", "-20", "-1", "80", "70"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			views := runClosed(t, []position.ClosedPosition{closedOf("1", testCase.side, testCase.netProfit)},
				map[string]position.PositionRisk{"1": {PositionID: "1", EntryPrice: "100", InitialStopLoss: testCase.stopLoss}})
			if len(views) != 1 {
				t.Fatalf("%d Views", len(views))
			}
			view := views[0]
			if strOrNil(view.RResult) != testCase.wantR || strOrNil(view.BreakEvenPrice) != testCase.wantBE || strOrNil(view.TP1Price) != testCase.wantT {
				t.Errorf("R=%s BE=%s TP1=%s, erwartet %s/%s/%s", strOrNil(view.RResult), strOrNil(view.BreakEvenPrice),
					strOrNil(view.TP1Price), testCase.wantR, testCase.wantBE, testCase.wantT)
			}
		})
	}
}

func TestGetClosedPositions_OhneBekanntesRisikoSindAlleKennzahlenNil(t *testing.T) {
	testCases := []struct {
		name   string
		side   position.PositionSide
		stored map[string]position.PositionRisk
	}{
		{"Risiko fehlt", position.Long, nil},
		{"SL = Entry", position.Long, map[string]position.PositionRisk{"1": {EntryPrice: "100", InitialStopLoss: "100"}}},
		{"LONG mit SL über Entry", position.Long, map[string]position.PositionRisk{"1": {EntryPrice: "100", InitialStopLoss: "110"}}},
		{"SHORT mit SL unter Entry", position.Short, map[string]position.PositionRisk{"1": {EntryPrice: "100", InitialStopLoss: "90"}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			views := runClosed(t, []position.ClosedPosition{closedOf("1", testCase.side, "10")}, testCase.stored)
			if len(views) != 1 {
				t.Fatalf("%d Views", len(views))
			}
			if views[0].RResult != nil || views[0].BreakEvenPrice != nil || views[0].TP1Price != nil {
				t.Errorf("alle Kennzahlen müssen nil sein: %+v", views[0])
			}
		})
	}
}

func TestGetClosedPositions_BehaeltReihenfolgeUndReichtAnfrageDurch(t *testing.T) {
	lister := &fakeClosedLister{positions: []position.ClosedPosition{closedOf("2", position.Long, "1"), closedOf("1", position.Long, "1")}}
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{Symbol: "SUI-USDT"})
	views, err := appposition.NewGetClosedPositionsUseCase(lister, &readOnlyRisks{t: t}).Execute(context.Background(), query)
	if err != nil || len(views) != 2 || views[0].PositionID != "2" || views[1].PositionID != "1" {
		t.Fatalf("views=%v err=%v", views, err)
	}
	if lister.got.Symbol() != "SUI-USDT" {
		t.Errorf("Anfrage nicht durchgereicht: %q", lister.got.Symbol())
	}
}

func TestGetClosedPositions_LeereListeIstNichtNilUndKeinFehler(t *testing.T) {
	views := runClosed(t, nil, nil)
	if views == nil || len(views) != 0 {
		t.Errorf("erwartet leere, nicht-nil Liste, war %v", views)
	}
}

func TestGetClosedPositions_ReichtFehlerDurch(t *testing.T) {
	sentinel := errors.New("db kaputt")
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	_, err := appposition.NewGetClosedPositionsUseCase(&fakeClosedLister{err: sentinel}, &readOnlyRisks{t: t}).Execute(context.Background(), query)
	if !errors.Is(err, sentinel) {
		t.Errorf("Lister-Fehler: %v", err)
	}
	_, err = appposition.NewGetClosedPositionsUseCase(
		&fakeClosedLister{positions: []position.ClosedPosition{closedOf("1", position.Long, "1")}},
		&readOnlyRisks{t: t, getErr: sentinel}).Execute(context.Background(), query)
	if !errors.Is(err, sentinel) {
		t.Errorf("Risiko-Fehler: %v", err)
	}
}
