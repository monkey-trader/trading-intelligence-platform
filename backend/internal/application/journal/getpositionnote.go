// Package journal enthält die Use Cases des Trading-Journals.
package journal

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
)

// GetPositionNoteUseCase liefert die Notiz zu einer Position.
type GetPositionNoteUseCase struct{ notes journal.NoteRepository }

// NewGetPositionNoteUseCase verdrahtet den Use Case mit dem Notiz-Port.
func NewGetPositionNoteUseCase(notes journal.NoteRepository) *GetPositionNoteUseCase {
	return &GetPositionNoteUseCase{notes: notes}
}

// Execute prüft die ID und liest die Notiz; Fehler des Ports gehen unverändert durch.
func (useCase *GetPositionNoteUseCase) Execute(ctx context.Context, positionID string) (journal.JournalNote, error) {
	if err := journal.ValidatePositionID(positionID); err != nil {
		return journal.JournalNote{}, err
	}
	return useCase.notes.FindByPositionID(ctx, positionID)
}
