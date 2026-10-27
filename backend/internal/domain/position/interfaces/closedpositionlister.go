package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// ClosedPositionLister liest vollständig geschlossene Positionen, neueste Schließung zuerst.
type ClosedPositionLister interface {
	ListClosedPositions(ctx context.Context, query position.ClosedPositionQuery) ([]position.ClosedPosition, error)
}
