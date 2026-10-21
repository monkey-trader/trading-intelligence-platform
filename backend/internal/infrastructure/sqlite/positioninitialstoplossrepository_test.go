package sqlite

import (
	"context"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet: NewPositionInitialStopLossRepository(db) implementiert interfaces.PositionInitialStopLossRepository
// (Set ersetzt, Get, Delete idempotent). Tabelle position_initial_stop_override entsteht über Open.
var _ interfaces.PositionInitialStopLossRepository = (*PositionInitialStopLossRepository)(nil)

func overrideOf(positionID, stop string, setAt int64) position.InitialStopLossOverride {
	return position.InitialStopLossOverride{PositionID: positionID, EntryPrice: "0.9600", InitialStopLoss: stop, SetAt: setAt}
}

func TestInitialStopLoss_FehlendeLiefertNichtGefunden(t *testing.T) {
	repo := NewPositionInitialStopLossRepository(openMemory(t))
	_, found, err := repo.Get(context.Background(), "1")
	if err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestInitialStopLoss_SetUndGetMit19stelligerId(t *testing.T) {
	repo := NewPositionInitialStopLossRepository(openMemory(t))
	ctx := context.Background()
	want := overrideOf("9100000000000000001", "0.9000", 1_791_800_000_000)
	if err := repo.Set(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, found, err := repo.Get(ctx, want.PositionID)
	if err != nil || !found || got != want {
		t.Fatalf("got=%+v found=%v err=%v", got, found, err)
	}
}

func TestInitialStopLoss_SetErsetzt(t *testing.T) {
	repo := NewPositionInitialStopLossRepository(openMemory(t))
	ctx := context.Background()
	_ = repo.Set(ctx, overrideOf("1", "0.9000", 100))
	second := overrideOf("1", "0.9500", 200)
	if err := repo.Set(ctx, second); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := repo.Get(ctx, "1"); got != second {
		t.Errorf("nicht ersetzt: %+v", got)
	}
}

func TestInitialStopLoss_DeleteIdempotentUndGetrennt(t *testing.T) {
	repo := NewPositionInitialStopLossRepository(openMemory(t))
	ctx := context.Background()
	_ = repo.Set(ctx, overrideOf("1", "0.9", 1))
	_ = repo.Set(ctx, overrideOf("2", "0.8", 2))
	for range 2 { // zweites Delete darf nicht scheitern
		if err := repo.Delete(ctx, "1"); err != nil {
			t.Fatal(err)
		}
	}
	if _, found, _ := repo.Get(ctx, "1"); found {
		t.Error("1 noch vorhanden")
	}
	if _, found, _ := repo.Get(ctx, "2"); !found {
		t.Error("2 fehlt")
	}
	if err := repo.Delete(ctx, "unbekannt"); err != nil {
		t.Fatal(err)
	}
}
