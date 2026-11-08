package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// SwingLongRepository speichert das manuelle Swing-Long-Flag je Position (positionId als TEXT).
type SwingLongRepository interface {
	// Set markiert die Position; idempotent (setAt in Epoch-ms).
	Set(ctx context.Context, positionID string, setAt int64) error
	// Unset entfernt das Flag; idempotent.
	Unset(ctx context.Context, positionID string) error
	// IsSet sagt, ob das Flag gesetzt ist.
	IsSet(ctx context.Context, positionID string) (bool, error)
	// ListIDs liefert die IDs aller markierten Positionen (offen und geschlossen).
	ListIDs(ctx context.Context) ([]string, error)
}

// SwingLongSpanSource liefert Symbol und Zeitraum aller markierten Positionen (offene mit CloseTime 0).
type SwingLongSpanSource interface {
	ListSwingLongSpans(ctx context.Context) ([]position.SwingLongSpan, error)
}
