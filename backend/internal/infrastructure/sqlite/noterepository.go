package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
)

// NoteRepository liest Journal-Notizen (nur SELECT; der Import schreibt nie in notes).
type NoteRepository struct{ db *sql.DB }

// NewNoteRepository erzeugt das Repository auf der geöffneten Datenbank.
func NewNoteRepository(db *sql.DB) *NoteRepository { return &NoteRepository{db: db} }

// FindByPositionID liefert die Notiz oder *journal.NoteNotFoundError.
func (repo *NoteRepository) FindByPositionID(ctx context.Context, positionID string) (journal.JournalNote, error) {
	var (
		setup, mistakes, notes sql.NullString
		rating, updatedAt      sql.NullInt64
	)
	err := repo.db.QueryRowContext(ctx,
		`SELECT setup, rating, mistakes, notes, updated_at FROM notes WHERE position_id = ?`, positionID,
	).Scan(&setup, &rating, &mistakes, &notes, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return journal.JournalNote{}, &journal.NoteNotFoundError{PositionID: positionID}
	}
	if err != nil {
		return journal.JournalNote{}, fmt.Errorf("notiz zu Position %s lesen: %w", positionID, err)
	}
	input := journal.JournalNoteInput{
		PositionID: positionID, Setup: textOrNil(setup), Mistakes: textOrNil(mistakes), Notes: textOrNil(notes),
		UpdatedAt: updatedAt.Int64,
	}
	if rating.Valid {
		value := int(rating.Int64)
		input.Rating = &value
	}
	note, err := journal.NewJournalNote(input)
	if err != nil {
		return journal.JournalNote{}, fmt.Errorf("notiz zu Position %s ungültig: %w", positionID, err)
	}
	return note, nil
}

func textOrNil(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
