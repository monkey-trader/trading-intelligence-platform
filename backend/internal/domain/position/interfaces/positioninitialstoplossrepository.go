package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionInitialStopLossRepository speichert manuell gesetzte ursprüngliche SLs je Position.
type PositionInitialStopLossRepository interface {
	// Set legt den Override an oder ersetzt den vorhandenen.
	Set(ctx context.Context, override position.InitialStopLossOverride) error
	// Get liefert den Override; found=false, wenn keiner existiert.
	Get(ctx context.Context, positionID string) (override position.InitialStopLossOverride, found bool, err error)
	// Delete entfernt den Override; unbekannte IDs sind kein Fehler.
	Delete(ctx context.Context, positionID string) error
}
