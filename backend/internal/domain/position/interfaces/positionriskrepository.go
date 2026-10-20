package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionRiskRepository speichert das ursprüngliche Risiko je Position; vorhandene Einträge bleiben unverändert.
type PositionRiskRepository interface {
	// SaveIfAbsent legt das Risiko an; created=false, wenn die positionId schon existiert (nichts wird überschrieben).
	SaveIfAbsent(ctx context.Context, risk position.PositionRisk) (created bool, err error)
	// Get liefert das Risiko; found=false, wenn die positionId fehlt.
	Get(ctx context.Context, positionID string) (risk position.PositionRisk, found bool, err error)
}
