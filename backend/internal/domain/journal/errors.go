package journal

import "fmt"

// InvalidPositionIDError meldet eine Positions-ID, die nicht aus 1 bis 20 Ziffern besteht.
type InvalidPositionIDError struct{ PositionID string }

func (err *InvalidPositionIDError) Error() string {
	return fmt.Sprintf("ungültige Positions-ID %q", err.PositionID)
}

// InvalidUpdatedAtError meldet einen negativen Zeitstempel.
type InvalidUpdatedAtError struct{ UpdatedAt int64 }

func (err *InvalidUpdatedAtError) Error() string {
	return fmt.Sprintf("ungültiger Änderungszeitpunkt %d", err.UpdatedAt)
}

// NoteNotFoundError meldet: Zu dieser Position gibt es keine Notiz.
type NoteNotFoundError struct{ PositionID string }

func (err *NoteNotFoundError) Error() string {
	return fmt.Sprintf("keine Notiz zur Position %s", err.PositionID)
}
