package sqlite

import (
	"context"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"testing"
	"time"
)

func TestFindEntry_LiefertSeiteUndEinstiegEinerAufgezeichnetenPosition(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	input := sampleOpenPosition()
	if err := NewPositionRepository(db, fixedClock(&now)).RecordOpenPositions(context.Background(), []position.OpenPosition{input}); err != nil {
		t.Fatal(err)
	}

	entry, found, err := NewPositionEntryLookup(db).FindEntry(context.Background(), input.PositionID)
	if err != nil || !found || string(entry.Side) != "SHORT" || entry.AvgOpenPrice != input.AvgOpenPrice {
		t.Fatalf("entry=%+v found=%v err=%v", entry, found, err)
	}
}

func TestFindEntry_UnbekanntePositionIstNichtGefunden(t *testing.T) {
	_, found, err := NewPositionEntryLookup(openMemory(t)).FindEntry(context.Background(), "123")
	if err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
}
