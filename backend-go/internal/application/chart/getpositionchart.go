// Package chart enthält den Use Case für den Positions-Chart.
package chart

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	positioninterfaces "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

// Query sind die bereits geprüften Parameter der Abfrage.
type Query struct {
	PositionID string
	Interval   chart.CandleInterval
	Limit      int
}

// EntryLevel ist der Einstieg der Position (Preis und Zeit in Epoch-ms).
type EntryLevel struct {
	Price string
	Time  int64
}

// ProtectiveOrderWithDistance ist eine Schutzorder plus relativer Abstand zum Entry.
type ProtectiveOrderWithDistance struct {
	chart.ProtectiveOrder
	DistanceRatio string
}

// MarkerKind benennt die Art eines Markers im Chart.
type MarkerKind string

// MarkerEntry ist in Stufe 1 der einzige Marker.
const MarkerEntry MarkerKind = "ENTRY"

// Marker ist ein Punkt im Chart.
type Marker struct {
	Time  int64
	Kind  MarkerKind
	Price string
}

// PositionChart ist das Ergebnis des Use Cases. Alle Listen sind nie nil.
type PositionChart struct {
	PositionID string
	Symbol     string
	Side       position.PositionSide
	Interval   chart.CandleInterval
	Candles    []chart.Candle
	Entry      EntryLevel
	StopLoss   []ProtectiveOrderWithDistance
	TakeProfit []ProtectiveOrderWithDistance
	Markers    []Marker
	FetchedAt  int64
}

// GetPositionChartUseCase baut den Chart einer offenen Position.
type GetPositionChartUseCase struct {
	positions positioninterfaces.OpenPositionSource
	orders    chartinterfaces.ProtectiveOrderSource
	source    chartinterfaces.CandleSource
	candles   chartinterfaces.CandleRepository
	logger    *slog.Logger
}

// NewGetPositionChartUseCase baut den Use Case mit den injizierten Ports.
func NewGetPositionChartUseCase(
	positions positioninterfaces.OpenPositionSource,
	orders chartinterfaces.ProtectiveOrderSource,
	source chartinterfaces.CandleSource,
	candles chartinterfaces.CandleRepository,
	logger *slog.Logger,
) *GetPositionChartUseCase {
	return &GetPositionChartUseCase{positions: positions, orders: orders, source: source, candles: candles, logger: logger}
}

// Execute sucht die Position, holt Orders und Kerzen, schreibt die Kerzen in den Cache und liest sie von dort.
func (useCase *GetPositionChartUseCase) Execute(ctx context.Context, query Query) (PositionChart, error) {
	openPosition, err := useCase.findPosition(ctx, query.PositionID)
	if err != nil {
		return PositionChart{}, err
	}
	ordersSnapshot, err := useCase.orders.FetchProtectiveOrders(ctx)
	if err != nil {
		return PositionChart{}, err
	}
	stopLoss, takeProfit, err := splitOrders(ordersSnapshot.Orders, openPosition)
	if err != nil {
		return PositionChart{}, err
	}

	fetched, err := useCase.source.FetchCandles(ctx, openPosition.Symbol, query.Interval, query.Limit)
	if err != nil {
		return PositionChart{}, err
	}
	if len(fetched) == 0 {
		// Leere Antwort bei offener Position ist verdächtig: loggen, nicht still hinnehmen.
		useCase.logger.WarnContext(ctx, "BingX lieferte keine Kerzen für eine offene Position",
			slog.String("symbol", openPosition.Symbol), slog.String("interval", string(query.Interval)))
	}
	if err := useCase.candles.UpsertCandles(ctx, fetched); err != nil {
		return PositionChart{}, fmt.Errorf("kerzen speichern: %w", err)
	}
	cached, err := useCase.candles.ListCandles(ctx, openPosition.Symbol, query.Interval, query.Limit)
	if err != nil {
		return PositionChart{}, fmt.Errorf("kerzen lesen: %w", err)
	}
	if cached == nil {
		cached = []chart.Candle{}
	}

	return PositionChart{
		PositionID: openPosition.PositionID,
		Symbol:     openPosition.Symbol,
		Side:       openPosition.Side,
		Interval:   query.Interval,
		Candles:    cached,
		Entry:      EntryLevel{Price: openPosition.AvgOpenPrice, Time: openPosition.OpenTime},
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		Markers:    []Marker{{Time: openPosition.OpenTime, Kind: MarkerEntry, Price: openPosition.AvgOpenPrice}},
		FetchedAt:  ordersSnapshot.FetchedAt,
	}, nil
}

func (useCase *GetPositionChartUseCase) findPosition(ctx context.Context, positionID string) (position.OpenPosition, error) {
	snapshot, err := useCase.positions.FetchOpenPositions(ctx)
	if err != nil {
		return position.OpenPosition{}, err
	}
	for _, openPosition := range snapshot.Positions {
		if openPosition.PositionID == positionID {
			return openPosition, nil
		}
	}
	return position.OpenPosition{}, &chart.PositionNotFoundError{PositionID: positionID}
}

// splitOrders wählt die Orders der Position (exakter String-Vergleich der ID) und trennt SL von TP.
func splitOrders(orders []chart.ProtectiveOrder, openPosition position.OpenPosition) (stopLoss, takeProfit []ProtectiveOrderWithDistance, err error) {
	stopLoss = []ProtectiveOrderWithDistance{}
	takeProfit = []ProtectiveOrderWithDistance{}
	for _, order := range orders {
		if order.PositionID != openPosition.PositionID {
			continue
		}
		triggerPrice, err := chart.ParseTriggerPrice(order.TriggerPrice)
		if err != nil {
			return nil, nil, err
		}
		ratio, err := chart.DistanceRatio(openPosition.AvgOpenPrice, triggerPrice)
		if err != nil {
			return nil, nil, fmt.Errorf("abstand der order %s: %w", order.OrderID, err)
		}
		withDistance := ProtectiveOrderWithDistance{ProtectiveOrder: order, DistanceRatio: ratio}
		if order.Kind == chart.StopLoss {
			stopLoss = append(stopLoss, withDistance)
		} else {
			takeProfit = append(takeProfit, withDistance)
		}
	}
	return stopLoss, takeProfit, nil
}
