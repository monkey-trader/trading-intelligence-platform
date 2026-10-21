package position

import (
	"context"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// SetInitialStopLossResult ist der gespeicherte Override samt Risiko-Distanz (1R).
type SetInitialStopLossResult struct {
	Override     domainposition.InitialStopLossOverride
	RiskDistance string
}

// SetInitialStopLossUseCase setzt den ursprünglichen SL einer offenen Position manuell.
type SetInitialStopLossUseCase struct {
	source    interfaces.OpenPositionSource
	overrides interfaces.PositionInitialStopLossRepository
	resolver  *RiskResolver
	now       func() int64
}

// NewSetInitialStopLossUseCase baut den Use Case.
func NewSetInitialStopLossUseCase(source interfaces.OpenPositionSource, overrides interfaces.PositionInitialStopLossRepository, resolver *RiskResolver, now func() int64) *SetInitialStopLossUseCase {
	return &SetInitialStopLossUseCase{source: source, overrides: overrides, resolver: resolver, now: now}
}

// Execute validiert und speichert den Override; die Position muss offen sein.
func (useCase *SetInitialStopLossUseCase) Execute(ctx context.Context, positionID, rawStopLoss string) (SetInitialStopLossResult, error) {
	snapshot, err := useCase.source.FetchOpenPositions(ctx)
	if err != nil {
		return SetInitialStopLossResult{}, err
	}
	for _, openPosition := range snapshot.Positions {
		if openPosition.PositionID != positionID {
			continue
		}
		override, err := domainposition.NewInitialStopLossOverride(positionID, openPosition.Side, openPosition.AvgOpenPrice, rawStopLoss, useCase.now())
		if err != nil {
			return SetInitialStopLossResult{}, err
		}
		riskDistance, err := domainposition.RiskDistance(openPosition.Side, override.EntryPrice, override.InitialStopLoss)
		if err != nil {
			return SetInitialStopLossResult{}, err
		}
		if err := useCase.overrides.Set(ctx, override); err != nil {
			return SetInitialStopLossResult{}, err
		}
		useCase.resolver.ForgetPosition(positionID)
		return SetInitialStopLossResult{Override: override, RiskDistance: riskDistance}, nil
	}
	return SetInitialStopLossResult{}, &chart.PositionNotFoundError{PositionID: positionID}
}

// ClearInitialStopLossUseCase entfernt den manuellen SL; danach gilt wieder die automatische Ermittlung.
type ClearInitialStopLossUseCase struct {
	overrides interfaces.PositionInitialStopLossRepository
	resolver  *RiskResolver
}

// NewClearInitialStopLossUseCase baut den Use Case.
func NewClearInitialStopLossUseCase(overrides interfaces.PositionInitialStopLossRepository, resolver *RiskResolver) *ClearInitialStopLossUseCase {
	return &ClearInitialStopLossUseCase{overrides: overrides, resolver: resolver}
}

// Execute löscht idempotent und leert den Negativ-Cache.
func (useCase *ClearInitialStopLossUseCase) Execute(ctx context.Context, positionID string) error {
	if err := useCase.overrides.Delete(ctx, positionID); err != nil {
		return fmt.Errorf("ursprungs-sl entfernen: %w", err)
	}
	useCase.resolver.ForgetPosition(positionID)
	return nil
}
