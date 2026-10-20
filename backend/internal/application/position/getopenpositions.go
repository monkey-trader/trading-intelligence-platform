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

// OpenPositionEntry ist eine offene Position plus SL, TP und Break-even (nil, wenn unbekannt).
type OpenPositionEntry struct {
	domainposition.OpenPosition
	StopLoss   *string // engster SL, nil ohne
	TakeProfit *string // nächster TP, nil ohne
	BreakEven  *bool   // nil = Orders unbekannt

	// Risiko-Felder (alle nil, wenn das Risiko unbekannt ist).
	InitialStopLoss         *string
	RiskDistance            *string
	CurrentRMultiple        *string
	BreakEvenTriggerReached *bool                   // nil auch, wenn die Kerzen nicht lesbar waren
	Reached2RAt             *int64                  // Kerzenbeginn, nil wenn nicht erreicht oder unbekannt
	Levels                  []domainposition.RLevel // 1R/2R/3R, nil bei unbekanntem Risiko
}

// riskCandleLimit ist die Zahl der 5-Minuten-Kerzen, in denen nach 2R gesucht wird.
const riskCandleLimit = 1000

// OpenPositionsOverview ist das Ergebnis: sortierte Positionen (nie nil), exakte PnL-Summe, Abfragezeit.
type OpenPositionsOverview struct {
	Positions             []OpenPositionEntry
	TotalUnrealizedProfit string
	FetchedAt             int64
}

// GetOpenPositionsUseCase liefert die offenen Positionen, neueste zuerst.
type GetOpenPositionsUseCase struct {
	source   interfaces.OpenPositionSource
	orders   chartinterfaces.ProtectiveOrderSource // optional
	risks    interfaces.PositionRiskRepository     // optional
	resolver *RiskResolver
	candles  chartinterfaces.CandleRepository // optional
	now      func() int64
}

// WithRisk ergänzt R-Multiple und 2R-Status; now liefert die Zeit in Epoch-ms für das erste Sehen.
func (useCase *GetOpenPositionsUseCase) WithRisk(risks interfaces.PositionRiskRepository, candles chartinterfaces.CandleRepository, now func() int64) *GetOpenPositionsUseCase {
	useCase.risks, useCase.candles, useCase.now = risks, candles, now
	useCase.resolver = NewRiskResolver(risks, now)
	return useCase
}

// WithRiskResolver nutzt einen gemeinsam verwendeten Resolver (z.B. mit dem Chart-Use-Case) samt seinem Repository.
func (useCase *GetOpenPositionsUseCase) WithRiskResolver(resolver *RiskResolver, risks interfaces.PositionRiskRepository, candles chartinterfaces.CandleRepository, now func() int64) *GetOpenPositionsUseCase {
	useCase.risks, useCase.candles, useCase.now = risks, candles, now
	useCase.resolver = resolver
	return useCase
}

// WithStopHistory ergänzt die Suche des ursprünglichen SL in der Order-Historie; nach WithRisk aufrufen.
func (useCase *GetOpenPositionsUseCase) WithStopHistory(history interfaces.StopOrderHistorySource) *GetOpenPositionsUseCase {
	if useCase.resolver != nil {
		useCase.resolver.WithHistory(history)
	}
	return useCase
}

// NewGetOpenPositionsUseCase baut den Use Case mit der injizierten Source.
func NewGetOpenPositionsUseCase(source interfaces.OpenPositionSource) *GetOpenPositionsUseCase {
	return &GetOpenPositionsUseCase{source: source}
}

// Execute fragt die Source ab, sortiert, ergänzt SL/TP/Break-even und summiert den PnL.
func (useCase *GetOpenPositionsUseCase) Execute(ctx context.Context) (OpenPositionsOverview, error) {
	snapshot, err := useCase.source.FetchOpenPositions(ctx)
	if err != nil {
		return OpenPositionsOverview{}, err
	}

	// Kopie sortieren: Die Source (z.B. ein Cache) darf ihr Slice behalten.
	sortedPositions := slices.Clone(snapshot.Positions)
	slices.SortFunc(sortedPositions, compareNewestFirst)

	if useCase.resolver != nil {
		openIDs := make([]string, 0, len(sortedPositions))
		for _, openPosition := range sortedPositions {
			openIDs = append(openIDs, openPosition.PositionID)
		}
		useCase.resolver.BeginRun(openIDs)
	}
	groupedOrders := useCase.ordersByPosition(ctx)
	entries := make([]OpenPositionEntry, 0, len(sortedPositions))
	for _, openPosition := range sortedPositions {
		entry := OpenPositionEntry{OpenPosition: openPosition}
		if groupedOrders != nil {
			applyProtection(ctx, &entry, groupedOrders[openPosition.PositionID])
		}
		if useCase.risks != nil {
			useCase.applyRisk(ctx, &entry, groupedOrders[openPosition.PositionID])
		}
		entries = append(entries, entry)
	}

	totalUnrealizedProfit, err := domainposition.SummarizeOpenPositions(sortedPositions)
	if err != nil {
		return OpenPositionsOverview{}, fmt.Errorf("summiere unrealisierten PnL: %w", err)
	}

	return OpenPositionsOverview{
		Positions:             entries,
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
func applyProtection(ctx context.Context, entry *OpenPositionEntry, orders []chart.ProtectiveOrder) {
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

// applyRisk setzt die Risiko-Felder; Fehler lassen sie leer (Warn-Log), die Liste bleibt lieferbar.
func (useCase *GetOpenPositionsUseCase) applyRisk(ctx context.Context, entry *OpenPositionEntry, orders []chart.ProtectiveOrder) {
	var stopTriggers []string
	for _, order := range orders {
		if order.Kind == chart.StopLoss {
			stopTriggers = append(stopTriggers, order.TriggerPrice)
		}
	}
	risk, known := useCase.resolver.Resolve(ctx, entry.OpenPosition, stopTriggers)
	if !known {
		return
	}
	riskDistance, err := domainposition.RiskDistance(entry.Side, risk.EntryPrice, risk.InitialStopLoss)
	if err != nil {
		slog.WarnContext(ctx, "Risiko nicht berechenbar", slog.String("positionId", entry.PositionID), slog.String("error", err.Error()))
		return
	}
	currentR, err := domainposition.CurrentRMultiple(entry.Side, risk.EntryPrice, riskDistance, entry.MarkPrice)
	if err != nil {
		slog.WarnContext(ctx, "R-Multiple nicht berechenbar", slog.String("positionId", entry.PositionID), slog.String("error", err.Error()))
		return
	}
	entry.InitialStopLoss, entry.RiskDistance, entry.CurrentRMultiple = &risk.InitialStopLoss, &riskDistance, &currentR
	if levels, levelsErr := domainposition.RLevels(entry.Side, risk.EntryPrice, risk.InitialStopLoss); levelsErr == nil {
		entry.Levels = levels
	}
	entry.BreakEvenTriggerReached, entry.Reached2RAt = useCase.twoRStatus(ctx, entry, risk)
}

// twoRStatus prüft in den 5-Minuten-Kerzen seit Eröffnung, ob 2R berührt wurde; (nil, nil), wenn unbekannt.
func (useCase *GetOpenPositionsUseCase) twoRStatus(ctx context.Context, entry *OpenPositionEntry, risk domainposition.PositionRisk) (*bool, *int64) {
	if useCase.candles == nil {
		return nil, nil
	}
	levels, err := domainposition.RLevels(entry.Side, risk.EntryPrice, risk.InitialStopLoss)
	if err != nil {
		return nil, nil
	}
	candles, err := useCase.candles.ListCandles(ctx, entry.Symbol, chart.Interval5m, riskCandleLimit)
	if err != nil {
		slog.WarnContext(ctx, "Kerzen nicht lesbar, 2R bleibt unbekannt", slog.String("symbol", entry.Symbol), slog.String("error", err.Error()))
		return nil, nil
	}
	ranges := make([]domainposition.CandleRange, 0, len(candles))
	for _, candle := range candles {
		ranges = append(ranges, domainposition.CandleRange{OpenTime: candle.OpenTime, High: candle.High, Low: candle.Low})
	}
	reachedAt, err := domainposition.ReachedAt(entry.Side, levels[1].Price, entry.OpenTime, ranges)
	if err != nil {
		slog.WarnContext(ctx, "2R nicht berechenbar", slog.String("positionId", entry.PositionID), slog.String("error", err.Error()))
		return nil, nil
	}
	reached := reachedAt != nil
	return &reached, reachedAt
}
