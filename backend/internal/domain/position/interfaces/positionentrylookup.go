package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionEntryLookup liefert Seite und Einstiegspreis einer bereits bekannten Position (lokal, ohne BingX).
// found=false, wenn die Position nicht in der Datenbank steht.
type PositionEntryLookup interface {
	FindEntry(ctx context.Context, positionID string) (position.PositionEntry, bool, error)
}
