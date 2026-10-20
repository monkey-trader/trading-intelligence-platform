package position

import (
	"context"
	"log/slog"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// ResolveRisk liefert das gespeicherte Risiko einer Position. Fehlt es, wird es aus dem aktuell engsten SL
// angelegt, aber nur, wenn dieser auf der Verlustseite des Entry liegt. known=false heißt: Risiko unbekannt
// (Fehler werden geloggt, nie weitergereicht).
func ResolveRisk(
	ctx context.Context,
	risks interfaces.PositionRiskRepository,
	now func() int64,
	openPosition domainposition.OpenPosition,
	stopTriggers []string,
) (risk domainposition.PositionRisk, known bool) {
	stored, found, err := risks.Get(ctx, openPosition.PositionID)
	if err != nil {
		slog.WarnContext(ctx, "Risiko nicht lesbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		return domainposition.PositionRisk{}, false
	}
	if found {
		return stored, true
	}
	stopLoss, err := domainposition.TightestStopLoss(openPosition.Side, stopTriggers)
	if err != nil || stopLoss == nil {
		return domainposition.PositionRisk{}, false
	}
	if _, err := domainposition.RiskDistance(openPosition.Side, openPosition.AvgOpenPrice, *stopLoss); err != nil {
		return domainposition.PositionRisk{}, false
	}
	created := domainposition.PositionRisk{
		PositionID:      openPosition.PositionID,
		EntryPrice:      openPosition.AvgOpenPrice,
		InitialStopLoss: *stopLoss,
		FirstSeenAt:     now(),
	}
	if _, err := risks.SaveIfAbsent(ctx, created); err != nil {
		slog.WarnContext(ctx, "Risiko nicht speicherbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		return domainposition.PositionRisk{}, false
	}
	return created, true
}
