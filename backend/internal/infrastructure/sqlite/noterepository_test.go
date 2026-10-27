package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func TestFindByPositionID_LiefertGespeicherteNotizMitAllenFeldern(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	input := sampleOpenPosition()
	if err := NewPositionRepository(db, fixedClock(&now)).RecordOpenPositions(context.Background(), []position.OpenPosition{input}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes (position_id, setup, rating, mistakes, notes, updated_at) VALUES (?, 'Breakout', 4, 'zu früh', 'ok', 1791600000000)`, input.PositionID); err != nil {
		t.Fatal(err)
	}

	note, err := NewNoteRepository(db).FindByPositionID(context.Background(), input.PositionID)

	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if note.PositionID != input.PositionID || *note.Setup != "Breakout" || *note.Rating != 4 ||
		*note.Mistakes != "zu früh" || *note.Notes != "ok" || note.UpdatedAt != 1791600000000 {
		t.Errorf("note = %+v", note)
	}
}

func TestFindByPositionID_NullSpaltenWerdenZuNilZeigern(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	input := sampleOpenPosition()
	if err := NewPositionRepository(db, fixedClock(&now)).RecordOpenPositions(context.Background(), []position.OpenPosition{input}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes (position_id, updated_at) VALUES (?, 1791600000000)`, input.PositionID); err != nil {
		t.Fatal(err)
	}

	note, err := NewNoteRepository(db).FindByPositionID(context.Background(), input.PositionID)

	if err != nil || note.Setup != nil || note.Rating != nil || note.Mistakes != nil || note.Notes != nil {
		t.Fatalf("note=%+v err=%v", note, err)
	}
}

func TestFindByPositionID_OhneNotizLiefertNoteNotFoundError(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	input := sampleOpenPosition()
	if err := NewPositionRepository(db, fixedClock(&now)).RecordOpenPositions(context.Background(), []position.OpenPosition{input}); err != nil {
		t.Fatal(err)
	}

	_, err := NewNoteRepository(db).FindByPositionID(context.Background(), input.PositionID)

	var notFound *journal.NoteNotFoundError
	if !errors.As(err, &notFound) || notFound.PositionID != input.PositionID {
		t.Fatalf("err = %v", err)
	}
}

func TestFindByPositionID_UnbekanntePositionLiefertNoteNotFoundError(t *testing.T) {
	_, err := NewNoteRepository(openMemory(t)).FindByPositionID(context.Background(), "123")
	var notFound *journal.NoteNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestFindByPositionID_SchreibtNichts(t *testing.T) {
	db := openMemory(t)
	_, _ = NewNoteRepository(db).FindByPositionID(context.Background(), "123")
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
