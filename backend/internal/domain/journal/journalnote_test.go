package journal_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
)

func textPtr(value string) *string { return &value }
func intPtr(value int) *int        { return &value }

func TestNewJournalNote_UebernimmtAlleFelder(t *testing.T) {
	note, err := journal.NewJournalNote(journal.JournalNoteInput{
		PositionID: "9100000000000000001", Setup: textPtr("Breakout"), Rating: intPtr(4),
		Mistakes: textPtr("zu früh raus"), Notes: textPtr("sauberer Trade"), UpdatedAt: 1791600000000,
	})
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if note.PositionID != "9100000000000000001" || *note.Setup != "Breakout" || *note.Rating != 4 ||
		*note.Mistakes != "zu früh raus" || *note.Notes != "sauberer Trade" || note.UpdatedAt != 1791600000000 {
		t.Errorf("note = %+v", note)
	}
}

func TestNewJournalNote_NichtGesetzteFelderBleibenNil(t *testing.T) {
	note, err := journal.NewJournalNote(journal.JournalNoteInput{PositionID: "1", UpdatedAt: 1})
	if err != nil || note.Setup != nil || note.Rating != nil || note.Mistakes != nil || note.Notes != nil {
		t.Fatalf("note=%+v err=%v", note, err)
	}
}

func TestNewJournalNote_UngueltigePositionsIdWirdAbgelehnt(t *testing.T) {
	for _, id := range []string{"", "abc", "12 3", "-1", "1.5"} {
		_, err := journal.NewJournalNote(journal.JournalNoteInput{PositionID: id, UpdatedAt: 1})
		var invalid *journal.InvalidPositionIDError
		if !errors.As(err, &invalid) {
			t.Errorf("id %q: err = %v, erwartet *InvalidPositionIDError", id, err)
		}
	}
}

func TestNewJournalNote_NegativerZeitstempelWirdAbgelehnt(t *testing.T) {
	_, err := journal.NewJournalNote(journal.JournalNoteInput{PositionID: "1", UpdatedAt: -1})
	var invalid *journal.InvalidUpdatedAtError
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, erwartet *InvalidUpdatedAtError", err)
	}
}

func TestNoteNotFoundError_TraegtPositionsIdUndIstPerErrorsAsErkennbar(t *testing.T) {
	var err error = &journal.NoteNotFoundError{PositionID: "42"}
	var notFound *journal.NoteNotFoundError
	if !errors.As(err, &notFound) || notFound.PositionID != "42" || err.Error() == "" {
		t.Fatalf("err = %v", err)
	}
}
