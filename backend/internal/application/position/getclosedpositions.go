package position

import (
	"context"
	"fmt"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// ClosedPositionView ist eine geschlossene Position plus R-Kennzahlen (nil ohne bekanntes Risiko).
type ClosedPositionView struct {
	domainposition.ClosedPosition
	RResult        *string
	BreakEvenPrice *string
	TP1Price       *string
}

// GetClosedPositionsUseCase liest nur; es speichert nie ein Risiko.
type GetClosedPositionsUseCase struct {
	lister interfaces.ClosedPositionLister
	risks  interfaces.PositionRiskRepository
}

// NewGetClosedPositionsUseCase erzeugt den Use Case.
func NewGetClosedPositionsUseCase(lister interfaces.ClosedPositionLister, risks interfaces.PositionRiskRepository) *GetClosedPositionsUseCase {
	return &GetClosedPositionsUseCase{lister: lister, risks: risks}
}

// Execute liefert die Positionen in Listen-Reihenfolge. Ohne verwertbares Risiko bleiben die Kennzahlen nil.
func (useCase *GetClosedPositionsUseCase) Execute(ctx context.Context, query domainposition.ClosedPositionQuery) ([]ClosedPositionView, error) {
	closed, err := useCase.lister.ListClosedPositions(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("geschlossene Positionen lesen: %w", err)
	}
	views := make([]ClosedPositionView, 0, len(closed))
	for _, item := range closed {
		view := ClosedPositionView{ClosedPosition: item}
		risk, found, err := useCase.risks.Get(ctx, item.PositionID)
		if err != nil {
			return nil, fmt.Errorf("risiko von position %s lesen: %w", item.PositionID, err)
		}
		if found {
			view.addRMetrics(risk)
		}
		views = append(views, view)
	}
	return views, nil
}

// addRMetrics setzt die Kennzahlen nur, wenn alle berechenbar sind (SL auf dem Entry oder falsche Seite: nichts).
func (view *ClosedPositionView) addRMetrics(risk domainposition.PositionRisk) {
	levels, err := domainposition.RLevels(view.Side, risk.EntryPrice, risk.InitialStopLoss)
	if err != nil {
		return
	}
	breakEven, foundBreakEven := domainposition.LevelAt(levels, 2)
	takeProfit, foundTakeProfit := domainposition.LevelAt(levels, 3)
	if !foundBreakEven || !foundTakeProfit {
		return
	}
	rResult, err := domainposition.RResult(view.NetProfit, view.PositionAmount, risk)
	if err != nil {
		return
	}
	view.RResult, view.BreakEvenPrice, view.TP1Price = &rResult, &breakEven.Price, &takeProfit.Price
}
