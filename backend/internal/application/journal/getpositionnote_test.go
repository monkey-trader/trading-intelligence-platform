package journal_test

import (
	"context"
	"errors"
	"testing"

	appjournal "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/journal"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
)

type fakeNotes struct {
	note   journal.JournalNote
	err    error
	gotID  string
	called int
}

func (repo *fakeNotes) FindByPositionID(_ context.Context, positionID string) (journal.JournalNote, error) {
	repo.called++
	repo.gotID = positionID
	return repo.note, repo.err
}

func TestGetPositionNote_LiefertDieNotizDesRepositories(t *testing.T) {
	setup := "Breakout"
	repo := &fakeNotes{note: journal.JournalNote{PositionID: "9100000000000000001", Setup: &setup, UpdatedAt: 5}}

	note, err := appjournal.NewGetPositionNoteUseCase(repo).Execute(context.Background(), "9100000000000000001")

	if err != nil || note.PositionID != "9100000000000000001" || *note.Setup != "Breakout" || repo.gotID != "9100000000000000001" {
		t.Fatalf("note=%+v err=%v gotID=%q", note, err, repo.gotID)
	}
}

func TestGetPositionNote_ReichtNoteNotFoundErrorDurch(t *testing.T) {
	repo := &fakeNotes{err: &journal.NoteNotFoundError{PositionID: "7"}}

	_, err := appjournal.NewGetPositionNoteUseCase(repo).Execute(context.Background(), "7")

	var notFound *journal.NoteNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetPositionNote_UngueltigePositionsIdFragtDasRepositoryNichtAn(t *testing.T) {
	repo := &fakeNotes{}

	_, err := appjournal.NewGetPositionNoteUseCase(repo).Execute(context.Background(), "abc")

	var invalid *journal.InvalidPositionIDError
	if !errors.As(err, &invalid) || repo.called != 0 {
		t.Fatalf("err=%v called=%d", err, repo.called)
	}
}

func TestGetPositionNote_ReichtInfrastrukturfehlerDurch(t *testing.T) {
	boom := errors.New("db kaputt")
	_, err := appjournal.NewGetPositionNoteUseCase(&fakeNotes{err: boom}).Execute(context.Background(), "7")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}
