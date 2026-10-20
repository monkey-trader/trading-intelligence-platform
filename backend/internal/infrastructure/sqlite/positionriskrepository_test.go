package sqlite

import (
	"context"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet: NewPositionRiskRepository(db) mit
//
//	SaveIfAbsent(ctx, position.PositionRisk) (created bool, err error)
//	Get(ctx, positionID string) (position.PositionRisk, bool, error)
//
// PositionRisk{PositionID, EntryPrice, InitialStopLoss string; FirstSeenAt int64}.
var _ interfaces.PositionRiskRepository = (*PositionRiskRepository)(nil)

func riskOf(positionID, stop string, seen int64) position.PositionRisk {
	return position.PositionRisk{PositionID: positionID, EntryPrice: "0.9600", InitialStopLoss: stop, FirstSeenAt: seen}
}

func TestPositionRisk_FehlendeLiefertNichtGefunden(t *testing.T) {
	repo := NewPositionRiskRepository(openMemory(t))
	_, found, err := repo.Get(context.Background(), "9100000000000000001")
	if err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestPositionRisk_AnlegenUndLesen(t *testing.T) {
	repo := NewPositionRiskRepository(openMemory(t))
	ctx := context.Background()
	want := riskOf("9100000000000000001", "0.9000", 1_791_800_000_000) // 19-stellige ID bleibt String
	created, err := repo.SaveIfAbsent(ctx, want)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	got, found, err := repo.Get(ctx, want.PositionID)
	if err != nil || !found || got != want {
		t.Fatalf("got=%+v found=%v err=%v", got, found, err)
	}
}

func TestPositionRisk_ZweitesSpeichernUeberschreibtNie(t *testing.T) {
	repo := NewPositionRiskRepository(openMemory(t))
	ctx := context.Background()
	first := riskOf("1", "0.9000", 100)
	_, _ = repo.SaveIfAbsent(ctx, first)
	created, err := repo.SaveIfAbsent(ctx, riskOf("1", "0.9500", 200))
	if err != nil || created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if got, _, _ := repo.Get(ctx, "1"); got != first {
		t.Errorf("überschrieben: %+v", got)
	}
}

func TestPositionRisk_GetrenntePositionen(t *testing.T) {
	repo := NewPositionRiskRepository(openMemory(t))
	ctx := context.Background()
	_, _ = repo.SaveIfAbsent(ctx, riskOf("1", "0.9", 1))
	if created, _ := repo.SaveIfAbsent(ctx, riskOf("2", "0.8", 2)); !created {
		t.Error("andere positionId muss angelegt werden")
	}
}
