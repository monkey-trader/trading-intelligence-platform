// Package interfaces enthält die Ports des Positions-Kerns.
package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// OpenPositionSource liefert die aktuell offenen Positionen samt Abfragezeitpunkt.
type OpenPositionSource interface {
	FetchOpenPositions(ctx context.Context) (position.OpenPositionsSnapshot, error)
}
