package position

import (
	"context"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// UnconfiguredOpenPositionSource ersetzt die BingX-Source, wenn BX_KEY/BX_SECRET fehlen.
type UnconfiguredOpenPositionSource struct{}

// FetchOpenPositions liefert immer not-configured.
func (UnconfiguredOpenPositionSource) FetchOpenPositions(context.Context) (domainposition.OpenPositionsSnapshot, error) {
	return domainposition.OpenPositionsSnapshot{}, &domainposition.OpenPositionsUnavailableError{
		Reason: domainposition.ReasonNotConfigured,
	}
}
