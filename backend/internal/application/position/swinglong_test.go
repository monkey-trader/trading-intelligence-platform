package position_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// fakeSwingLong ist ein InMemory-Adapter des Ports SwingLongRepository.
type fakeSwingLong struct {
	flags map[string]int64
	err   error
}

func newFakeSwingLong(ids ...string) *fakeSwingLong {
	fake := &fakeSwingLong{flags: map[string]int64{}}
	for _, id := range ids {
		fake.flags[id] = 1
	}
	return fake
}

func (fake *fakeSwingLong) Set(_ context.Context, id string, at int64) error {
	if fake.err != nil {
		return fake.err
	}
	fake.flags[id] = at
	return nil
}

func (fake *fakeSwingLong) Unset(_ context.Context, id string) error {
	if fake.err != nil {
		return fake.err
	}
	delete(fake.flags, id)
	return nil
}

func (fake *fakeSwingLong) IsSet(_ context.Context, id string) (bool, error) {
	_, found := fake.flags[id]
	return found, fake.err
}

func (fake *fakeSwingLong) ListIDs(context.Context) ([]string, error) {
	ids := make([]string, 0, len(fake.flags))
	for id := range fake.flags {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, fake.err
}

var _ interfaces.SwingLongRepository = (*fakeSwingLong)(nil)

func TestSetSwingLong_SetztFlagMitZeitstempelUndIstIdempotent(t *testing.T) {
	flags := newFakeSwingLong()
	useCase := appposition.NewSetSwingLongUseCase(flags, func() int64 { return 1_791_800_000_000 })
	for range 2 {
		if err := useCase.Execute(context.Background(), "9100000000000000001"); err != nil {
			t.Fatal(err)
		}
	}
	if flags.flags["9100000000000000001"] != 1_791_800_000_000 || len(flags.flags) != 1 {
		t.Errorf("flags = %v", flags.flags)
	}
}

func TestSetSwingLong_ReichtRepositoryFehlerDurch(t *testing.T) {
	boom := errors.New("db kaputt")
	flags := newFakeSwingLong()
	flags.err = boom
	if err := appposition.NewSetSwingLongUseCase(flags, func() int64 { return 1 }).Execute(context.Background(), "1"); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestUnsetSwingLong_EntferntFlagIdempotentAuchOhneFlag(t *testing.T) {
	flags := newFakeSwingLong("1")
	useCase := appposition.NewUnsetSwingLongUseCase(flags)
	for range 2 {
		if err := useCase.Execute(context.Background(), "1"); err != nil {
			t.Fatal(err)
		}
	}
	if len(flags.flags) != 0 {
		t.Errorf("flags = %v", flags.flags)
	}
}

func TestUnsetSwingLong_ReichtRepositoryFehlerDurch(t *testing.T) {
	boom := errors.New("db kaputt")
	flags := newFakeSwingLong()
	flags.err = boom
	if err := appposition.NewUnsetSwingLongUseCase(flags).Execute(context.Background(), "1"); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestGetOpenPositions_SwingLongFlagUndGetrennteSumme(t *testing.T) {
	source := &fakeSource{snapshot: position.OpenPositionsSnapshot{FetchedAt: fetchedAt, Positions: []position.OpenPosition{
		openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.UnrealizedProfit = "25.5" }),
		openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.UnrealizedProfit = "-100.25" }),
		openPosition(func(p *position.OpenPosition) { p.PositionID = "3"; p.UnrealizedProfit = "4" }),
	}}}
	overview, err := appposition.NewGetOpenPositionsUseCase(source).WithSwingLong(newFakeSwingLong("2")).Execute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	flagged := map[string]bool{}
	for _, entry := range overview.Positions {
		flagged[entry.PositionID] = entry.SwingLong
	}
	if flagged["1"] || !flagged["2"] || flagged["3"] {
		t.Errorf("SwingLong-Flags = %v", flagged)
	}
	if overview.TotalUnrealizedProfit != "29.5" || overview.SwingLongUnrealizedProfit != "-100.25" {
		t.Errorf("total=%s swing=%s, erwartet 29.5 / -100.25", overview.TotalUnrealizedProfit, overview.SwingLongUnrealizedProfit)
	}
}

func TestGetOpenPositions_SwingLongSummenNullWennLeerOderNurSwing(t *testing.T) {
	testCases := []struct {
		name                 string
		positions            []position.OpenPosition
		flags                []string
		wantTotal, wantSwing string
	}{
		{"keine Positionen", nil, nil, "0", "0"},
		{"kein Flag", []position.OpenPosition{openPosition(nil)}, nil, "25", "0"},
		{"nur Swing Long", []position.OpenPosition{openPosition(nil)}, []string{"9100000000000000001"}, "0", "25"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := &fakeSource{snapshot: position.OpenPositionsSnapshot{FetchedAt: fetchedAt, Positions: testCase.positions}}
			overview, err := appposition.NewGetOpenPositionsUseCase(source).WithSwingLong(newFakeSwingLong(testCase.flags...)).Execute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if overview.TotalUnrealizedProfit != testCase.wantTotal || overview.SwingLongUnrealizedProfit != testCase.wantSwing {
				t.Errorf("total=%s swing=%s", overview.TotalUnrealizedProfit, overview.SwingLongUnrealizedProfit)
			}
		})
	}
}

func TestGetOpenPositions_SwingLongFehlerDesRepositoryWirdGemeldet(t *testing.T) {
	flags := newFakeSwingLong()
	flags.err = errors.New("db kaputt")
	source := &fakeSource{snapshot: position.OpenPositionsSnapshot{Positions: []position.OpenPosition{openPosition(nil)}}}
	if _, err := appposition.NewGetOpenPositionsUseCase(source).WithSwingLong(flags).Execute(context.Background()); err == nil {
		t.Error("Fehler erwartet")
	}
}

func TestGetClosedPositions_SwingLongFlagBleibtNachSchliessen(t *testing.T) {
	positions := []position.ClosedPosition{closedOf("1", position.Long, "10"), closedOf("2", position.Long, "-50")}
	query, _ := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	views, err := appposition.NewGetClosedPositionsUseCase(&fakeClosedLister{positions: positions}, &readOnlyRisks{t: t}).
		WithSwingLong(newFakeSwingLong("2")).Execute(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].SwingLong || !views[1].SwingLong {
		t.Errorf("views = %+v", views)
	}
}

func TestGetClosedPositions_OhneSwingLongRepositoryIstFlagFalse(t *testing.T) {
	views := runClosed(t, []position.ClosedPosition{closedOf("1", position.Long, "10")}, nil)
	if views[0].SwingLong {
		t.Error("SwingLong muss false sein")
	}
}
