package httpapi

import appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"

// openPositionDto ist das Schema OpenPosition aus docs/openapi.yaml, Felder in Spec-Reihenfolge.
type openPositionDto struct {
	PositionID              string      `json:"positionId"`
	Symbol                  string      `json:"symbol"`
	Side                    string      `json:"side"`
	Isolated                bool        `json:"isolated"`
	PositionAmount          string      `json:"positionAmount"`
	AvgOpenPrice            string      `json:"avgOpenPrice"`
	MarkPrice               string      `json:"markPrice"`
	Leverage                float64     `json:"leverage"`
	InitialMargin           string      `json:"initialMargin"`
	Margin                  string      `json:"margin"`
	PositionValue           string      `json:"positionValue"`
	UnrealizedProfit        string      `json:"unrealizedProfit"`
	RealisedProfit          string      `json:"realisedProfit"`
	PnlRatio                string      `json:"pnlRatio"`
	StopLoss                *string     `json:"stopLoss"`
	TakeProfit              *string     `json:"takeProfit"`
	BreakEven               *bool       `json:"breakEven"`
	InitialStopLoss         *string     `json:"initialStopLoss,omitempty"`
	RiskDistance            *string     `json:"riskDistance,omitempty"`
	CurrentRMultiple        *string     `json:"currentRMultiple,omitempty"`
	BreakEvenTriggerReached *bool       `json:"breakEvenTriggerReached,omitempty"`
	Reached2RAt             *int64      `json:"reached2RAt,omitempty"`
	Levels                  []rLevelDto `json:"levels,omitempty"`
	OpenTime                int64       `json:"openTime"`
	UpdateTime              int64       `json:"updateTime"`
}

// openPositionsSnapshotDto ist das Schema OpenPositionsSnapshot; positions ist nie null.
type openPositionsSnapshotDto struct {
	Positions             []openPositionDto `json:"positions"`
	TotalUnrealizedProfit string            `json:"totalUnrealizedProfit"`
	FetchedAt             int64             `json:"fetchedAt"`
}

// toOpenPositionsSnapshotDto bildet das Ergebnis des Use Cases auf den API-Vertrag ab.
func toOpenPositionsSnapshotDto(overview appposition.OpenPositionsOverview) openPositionsSnapshotDto {
	positions := make([]openPositionDto, 0, len(overview.Positions))
	for _, entry := range overview.Positions {
		positions = append(positions, openPositionDto{
			PositionID:              entry.PositionID,
			Symbol:                  entry.Symbol,
			Side:                    string(entry.Side),
			Isolated:                entry.Isolated,
			PositionAmount:          entry.PositionAmount,
			AvgOpenPrice:            entry.AvgOpenPrice,
			MarkPrice:               entry.MarkPrice,
			Leverage:                entry.Leverage,
			InitialMargin:           entry.InitialMargin,
			Margin:                  entry.Margin,
			PositionValue:           entry.PositionValue,
			UnrealizedProfit:        entry.UnrealizedProfit,
			RealisedProfit:          entry.RealisedProfit,
			PnlRatio:                entry.PnlRatio,
			StopLoss:                entry.StopLoss,
			TakeProfit:              entry.TakeProfit,
			BreakEven:               entry.BreakEven,
			InitialStopLoss:         entry.InitialStopLoss,
			RiskDistance:            entry.RiskDistance,
			CurrentRMultiple:        entry.CurrentRMultiple,
			BreakEvenTriggerReached: entry.BreakEvenTriggerReached,
			Reached2RAt:             entry.Reached2RAt,
			Levels:                  toRLevelDtos(entry.Levels),
			OpenTime:                entry.OpenTime,
			UpdateTime:              entry.UpdateTime,
		})
	}
	return openPositionsSnapshotDto{
		Positions:             positions,
		TotalUnrealizedProfit: overview.TotalUnrealizedProfit,
		FetchedAt:             overview.FetchedAt,
	}
}
