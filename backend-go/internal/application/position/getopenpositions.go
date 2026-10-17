// Package position enthält die Use Cases rund um Positionen.
package position

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

// OpenPositionWithDistance ist eine offene Position plus Abstand zur Liquidation (nil ohne).
type OpenPositionWithDistance struct {
	domainposition.OpenPosition
	LiquidationDistanceRatio *string
}

// OpenPositionsOverview ist das Ergebnis: sortierte Positionen (nie nil), exakte PnL-Summe, Abfragezeit.
type OpenPositionsOverview struct {
	Positions             []OpenPositionWithDistance
	TotalUnrealizedProfit string
	FetchedAt             int64
}

// GetOpenPositionsUseCase liefert die offenen Positionen, neueste zuerst.
type GetOpenPositionsUseCase struct {
	source interfaces.OpenPositionSource
}

// NewGetOpenPositionsUseCase baut den Use Case mit der injizierten Source.
func NewGetOpenPositionsUseCase(source interfaces.OpenPositionSource) *GetOpenPositionsUseCase {
	return &GetOpenPositionsUseCase{source: source}
}

// Execute fragt die Source ab, sortiert, ergänzt den Liquidationsabstand und summiert den PnL.
func (useCase *GetOpenPositionsUseCase) Execute(ctx context.Context) (OpenPositionsOverview, error) {
	snapshot, err := useCase.source.FetchOpenPositions(ctx)
	if err != nil {
		return OpenPositionsOverview{}, err
	}

	// Kopie sortieren: Die Source (z.B. ein Cache) darf ihr Slice behalten.
	sortedPositions := slices.Clone(snapshot.Positions)
	slices.SortFunc(sortedPositions, compareNewestFirst)

	positionsWithDistance := make([]OpenPositionWithDistance, 0, len(sortedPositions))
	for _, openPosition := range sortedPositions {
		ratio, err := domainposition.LiquidationDistanceRatio(
			openPosition.Side, openPosition.MarkPrice, openPosition.LiquidationPrice)
		if err != nil {
			return OpenPositionsOverview{}, fmt.Errorf("berechne Liquidationsabstand für Position %s: %w", openPosition.PositionID, err)
		}
		positionsWithDistance = append(positionsWithDistance, OpenPositionWithDistance{
			OpenPosition:             openPosition,
			LiquidationDistanceRatio: ratio,
		})
	}

	totalUnrealizedProfit, err := domainposition.SummarizeOpenPositions(sortedPositions)
	if err != nil {
		return OpenPositionsOverview{}, fmt.Errorf("summiere unrealisierten PnL: %w", err)
	}

	return OpenPositionsOverview{
		Positions:             positionsWithDistance,
		TotalUnrealizedProfit: totalUnrealizedProfit,
		FetchedAt:             snapshot.FetchedAt,
	}, nil
}

// compareNewestFirst sortiert nach openTime absteigend, dann Symbol, Seite und positionId (Byte-Reihenfolge).
func compareNewestFirst(left, right domainposition.OpenPosition) int {
	return cmp.Or(
		cmp.Compare(right.OpenTime, left.OpenTime),
		strings.Compare(left.Symbol, right.Symbol),
		strings.Compare(string(left.Side), string(right.Side)),
		strings.Compare(left.PositionID, right.PositionID),
	)
}
