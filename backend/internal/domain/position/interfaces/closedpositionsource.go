package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// ClosedPositionSource liefert geschlossene Positionen aus der BingX-API pro Symbol und Zeitbereich.
type ClosedPositionSource interface {
	// FetchClosedPositions ruft positionHistory für ein Symbol im Zeitraum [from, to] ab.
	// Die Quelle muss sicherstellen, dass der Bereich nicht größer als 89 Tage ist.
	FetchClosedPositions(ctx context.Context, symbol string, from, to int64) ([]position.ClosedPosition, error)
}
