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
	entries   interfaces.PositionEntryLookup
	overrides interfaces.PositionInitialStopLossRepository
	resolver  *RiskResolver
	now       func() int64
}

// NewSetInitialStopLossUseCase baut den Use Case.
func NewSetInitialStopLossUseCase(entries interfaces.PositionEntryLookup, overrides interfaces.PositionInitialStopLossRepository, resolver *RiskResolver, now func() int64) *SetInitialStopLossUseCase {
	return &SetInitialStopLossUseCase{entries: entries, overrides: overrides, resolver: resolver, now: now}
}

// Execute validiert und speichert den Override; Seite und Einstieg stammen aus der lokalen Datenbank.
func (useCase *SetInitialStopLossUseCase) Execute(ctx context.Context, positionID, rawStopLoss string) (SetInitialStopLossResult, error) {
	entry, found, err := useCase.entries.FindEntry(ctx, positionID)
	if err != nil {
		return SetInitialStopLossResult{}, err
	}
	if !found {
		return SetInitialStopLossResult{}, &chart.PositionNotFoundError{PositionID: positionID}
	}
	override, err := domainposition.NewInitialStopLossOverride(positionID, entry.Side, entry.AvgOpenPrice, rawStopLoss, useCase.now())
	if err != nil {
		return SetInitialStopLossResult{}, err
	}
	riskDistance, err := domainposition.RiskDistance(entry.Side, override.EntryPrice, override.InitialStopLoss)
	if err != nil {
		return SetInitialStopLossResult{}, err
	}
	if err := useCase.overrides.Set(ctx, override); err != nil {
		return SetInitialStopLossResult{}, err
	}
	useCase.resolver.ForgetPosition(positionID)
	return SetInitialStopLossResult{Override: override, RiskDistance: riskDistance}, nil
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
