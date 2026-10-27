package httpapi

import "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"

// journalNoteDto ist die JSON-Form von GET /positions/{positionId}/note.
type journalNoteDto struct {
	PositionID string  `json:"positionId"`
	Setup      *string `json:"setup"`
	Rating     *int    `json:"rating"`
	Mistakes   *string `json:"mistakes"`
	Notes      *string `json:"notes"`
	UpdatedAt  int64   `json:"updatedAt"`
}

func toJournalNoteDto(note journal.JournalNote) journalNoteDto {
	return journalNoteDto(note)
}
