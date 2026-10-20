// Package position enthält die Use Cases rund um Positionen.
package position

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart/interfaces"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// OpenPositionWithDistance ist eine offene Position plus Abstand zur Liquidation (nil ohne).
type OpenPositionWithDistance struct {
	domainposition.OpenPosition
	LiquidationDistanceRatio *string
	StopLoss                 *string // engster SL, nil ohne
	TakeProfit               *string // nächster TP, nil ohne
	BreakEven                *bool   // nil = Orders unbekannt
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
	orders chartinterfaces.ProtectiveOrderSource // optional
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

	groupedOrders := useCase.ordersByPosition(ctx)
	positionsWithDistance := make([]OpenPositionWithDistance, 0, len(sortedPositions))
	for _, openPosition := range sortedPositions {
		ratio, err := domainposition.LiquidationDistanceRatio(
			openPosition.Side, openPosition.MarkPrice, openPosition.LiquidationPrice)
		if err != nil {
			return OpenPositionsOverview{}, fmt.Errorf("berechne Liquidationsabstand für Position %s: %w", openPosition.PositionID, err)
		}
		entry := OpenPositionWithDistance{OpenPosition: openPosition, LiquidationDistanceRatio: ratio}
		if groupedOrders != nil {
			applyProtection(ctx, &entry, groupedOrders[openPosition.PositionID])
		}
		positionsWithDistance = append(positionsWithDistance, entry)
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

// NewGetOpenPositionsUseCaseWithProtection ergänzt SL/TP/Break-even aus der Order-Source.
func NewGetOpenPositionsUseCaseWithProtection(source interfaces.OpenPositionSource, orders chartinterfaces.ProtectiveOrderSource) *GetOpenPositionsUseCase {
	return &GetOpenPositionsUseCase{source: source, orders: orders}
}

// ordersByPosition gruppiert die Schutzorders je positionId; nil, wenn sie nicht ladbar sind (Warn-Log).
func (useCase *GetOpenPositionsUseCase) ordersByPosition(ctx context.Context) map[string][]chart.ProtectiveOrder {
	if useCase.orders == nil {
		return nil
	}
	snapshot, err := useCase.orders.FetchProtectiveOrders(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Schutzorders nicht ladbar, SL/TP/Break-even bleiben leer", slog.String("error", err.Error()))
		return nil
	}
	grouped := make(map[string][]chart.ProtectiveOrder)
	for _, order := range snapshot.Orders {
		grouped[order.PositionID] = append(grouped[order.PositionID], order)
	}
	return grouped
}

// applyProtection setzt SL, TP und Break-even; Fehler bei der Berechnung lassen die Felder leer.
func applyProtection(ctx context.Context, entry *OpenPositionWithDistance, orders []chart.ProtectiveOrder) {
	var stopTriggers, profitTriggers []string
	for _, order := range orders {
		switch order.Kind {
		case chart.StopLoss:
			stopTriggers = append(stopTriggers, order.TriggerPrice)
		case chart.TakeProfit:
			profitTriggers = append(profitTriggers, order.TriggerPrice)
		}
	}
	stopLoss, err := domainposition.TightestStopLoss(entry.Side, stopTriggers)
	if err == nil {
		var takeProfit *string
		takeProfit, err = domainposition.NearestTakeProfit(entry.Side, entry.AvgOpenPrice, profitTriggers)
		if err == nil {
			var breakEven bool
			breakEven, err = domainposition.IsBreakEven(entry.Side, entry.AvgOpenPrice, stopLoss)
			if err == nil {
				entry.StopLoss, entry.TakeProfit, entry.BreakEven = stopLoss, takeProfit, &breakEven
			}
		}
	}
	if err != nil {
		slog.WarnContext(ctx, "SL/TP/Break-even nicht berechenbar",
			slog.String("positionId", entry.PositionID), slog.String("error", err.Error()))
	}
}
