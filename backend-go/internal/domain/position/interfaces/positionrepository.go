package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// PositionRepository speichert geschlossene Positionen dauerhaft.
type PositionRepository interface {
	// UpsertClosedPosition speichert oder aktualisiert eine geschlossene Position per Upsert (positionId ist Schlüssel).
	UpsertClosedPosition(ctx context.Context, position position.ClosedPosition) error
}
