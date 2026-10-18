package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// OpenPositionRecorder legt aktuell offene Positionen im Journal an, damit sie vor dem Schließen
// verwaltet werden können. Bereits bekannte Positionen bleiben unverändert.
type OpenPositionRecorder interface {
	RecordOpenPositions(ctx context.Context, positions []position.OpenPosition) error
}
