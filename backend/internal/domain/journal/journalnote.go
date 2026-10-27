// Package journal enthält die Fachobjekte des Trading-Journals (Notizen zu Positionen).
package journal

import (
	"context"
	"regexp"
)

var positionIDPattern = regexp.MustCompile(`^\d{1,20}$`)

// JournalNote ist die Notiz des Nutzers zu einer Position; nicht gesetzte Felder sind nil.
type JournalNote struct { //nolint:revive // Name laut Vertrag (Tests, OpenAPI-Schema)
	PositionID string
	Setup      *string
	Rating     *int
	Mistakes   *string
	Notes      *string
	UpdatedAt  int64
}

// JournalNoteInput sind die Rohwerte für NewJournalNote.
type JournalNoteInput struct { //nolint:revive // Name laut Vertrag (Tests)
	PositionID string
	Setup      *string
	Rating     *int
	Mistakes   *string
	Notes      *string
	UpdatedAt  int64
}

// NewJournalNote prüft die Invarianten (Positions-ID nur Ziffern, Zeitstempel nicht negativ).
func NewJournalNote(input JournalNoteInput) (JournalNote, error) {
	if err := ValidatePositionID(input.PositionID); err != nil {
		return JournalNote{}, err
	}
	if input.UpdatedAt < 0 {
		return JournalNote{}, &InvalidUpdatedAtError{UpdatedAt: input.UpdatedAt}
	}
	return JournalNote(input), nil
}

// ValidatePositionID prüft, dass die ID aus 1 bis 20 Ziffern besteht.
func ValidatePositionID(positionID string) error {
	if !positionIDPattern.MatchString(positionID) {
		return &InvalidPositionIDError{PositionID: positionID}
	}
	return nil
}

// NoteRepository ist der Port zum Lesen von Notizen.
type NoteRepository interface {
	// FindByPositionID liefert die Notiz oder *NoteNotFoundError.
	FindByPositionID(ctx context.Context, positionID string) (JournalNote, error)
}
