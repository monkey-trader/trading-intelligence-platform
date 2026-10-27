package httpapi

import appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"

// closedPositionDto ist die JSON-Form eines Eintrags von GET /positions/closed.
type closedPositionDto struct {
	PositionID     string  `json:"positionId"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	OpenTime       int64   `json:"openTime"`
	CloseTime      int64   `json:"closeTime"`
	AvgOpenPrice   string  `json:"avgOpenPrice"`
	AvgClosePrice  string  `json:"avgClosePrice"`
	PositionAmount string  `json:"positionAmount"`
	Leverage       float64 `json:"leverage"`
	RealisedProfit string  `json:"realisedProfit"`
	NetProfit      string  `json:"netProfit"`
	Commission     string  `json:"commission"`
	TotalFunding   string  `json:"totalFunding"`
	RResult        *string `json:"rResult"`
	BreakEvenPrice *string `json:"breakEvenPrice"`
	TP1Price       *string `json:"tp1Price"`
}

type closedPositionsDto struct {
	Positions []closedPositionDto `json:"positions"`
}

func toClosedPositionsDto(views []appposition.ClosedPositionView) closedPositionsDto {
	positions := make([]closedPositionDto, 0, len(views))
	for _, view := range views {
		positions = append(positions, closedPositionDto{
			PositionID: view.PositionID, Symbol: view.Symbol, Side: string(view.Side),
			OpenTime: view.OpenTime, CloseTime: view.CloseTime,
			AvgOpenPrice: view.AvgOpenPrice, AvgClosePrice: view.AvgClosePrice,
			PositionAmount: view.PositionAmount, Leverage: view.Leverage,
			RealisedProfit: view.RealisedProfit, NetProfit: view.NetProfit,
			Commission: view.Commission, TotalFunding: view.TotalFunding,
			RResult: view.RResult, BreakEvenPrice: view.BreakEvenPrice, TP1Price: view.TP1Price,
		})
	}
	return closedPositionsDto{Positions: positions}
}
