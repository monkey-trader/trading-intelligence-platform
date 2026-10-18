package httpapi

import appchart "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/chart"

// candleDto ist das Schema Candle aus docs/openapi.yaml.
type candleDto struct {
	Time   int64  `json:"time"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

type entryLevelDto struct {
	Price string `json:"price"`
	Time  int64  `json:"time"`
}

type protectiveOrderDto struct {
	OrderID       string `json:"orderId"`
	TriggerPrice  string `json:"triggerPrice"`
	Quantity      string `json:"quantity"`
	DistanceRatio string `json:"distanceRatio"`
}

type markerDto struct {
	Time  int64  `json:"time"`
	Kind  string `json:"kind"`
	Price string `json:"price"`
}

// positionChartDto ist das Schema PositionChart; Listen sind nie null.
type positionChartDto struct {
	PositionID string               `json:"positionId"`
	Symbol     string               `json:"symbol"`
	Side       string               `json:"side"`
	Interval   string               `json:"interval"`
	Candles    []candleDto          `json:"candles"`
	Entry      entryLevelDto        `json:"entry"`
	StopLoss   []protectiveOrderDto `json:"stopLoss"`
	TakeProfit []protectiveOrderDto `json:"takeProfit"`
	Markers    []markerDto          `json:"markers"`
	FetchedAt  int64                `json:"fetchedAt"`
}

func toPositionChartDto(result appchart.PositionChart) positionChartDto {
	candles := make([]candleDto, 0, len(result.Candles))
	for _, candle := range result.Candles {
		candles = append(candles, candleDto{Time: candle.OpenTime, Open: candle.Open, High: candle.High,
			Low: candle.Low, Close: candle.Close, Volume: candle.Volume})
	}
	markers := make([]markerDto, 0, len(result.Markers))
	for _, marker := range result.Markers {
		markers = append(markers, markerDto{Time: marker.Time, Kind: string(marker.Kind), Price: marker.Price})
	}
	return positionChartDto{
		PositionID: result.PositionID, Symbol: result.Symbol, Side: string(result.Side),
		Interval: string(result.Interval), Candles: candles,
		Entry:    entryLevelDto{Price: result.Entry.Price, Time: result.Entry.Time},
		StopLoss: toProtectiveOrderDtos(result.StopLoss), TakeProfit: toProtectiveOrderDtos(result.TakeProfit),
		Markers: markers, FetchedAt: result.FetchedAt,
	}
}

func toProtectiveOrderDtos(orders []appchart.ProtectiveOrderWithDistance) []protectiveOrderDto {
	dtos := make([]protectiveOrderDto, 0, len(orders))
	for _, order := range orders {
		dtos = append(dtos, protectiveOrderDto{OrderID: order.OrderID, TriggerPrice: order.TriggerPrice,
			Quantity: order.Quantity, DistanceRatio: order.DistanceRatio})
	}
	return dtos
}
