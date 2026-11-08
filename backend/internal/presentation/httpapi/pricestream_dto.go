package httpapi

import appprice "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"

type positionPriceDTO struct {
	PositionID                 string  `json:"positionId"`
	Symbol                     string  `json:"symbol"`
	Price                      string  `json:"price"`
	Change24hPercent           string  `json:"change24hPercent"`
	DistanceToBreakEvenPercent *string `json:"distanceToBreakEvenPercent"`
	DistanceToTp1Percent       *string `json:"distanceToTp1Percent"`
}

type priceSnapshotDTO struct {
	Prices    []positionPriceDTO `json:"prices"`
	FetchedAt int64              `json:"fetchedAt"`
	Stale     bool               `json:"stale"`
}

func toPriceSnapshotDTO(snapshot appprice.PriceSnapshot) priceSnapshotDTO {
	prices := make([]positionPriceDTO, 0, len(snapshot.Prices))
	for _, entry := range snapshot.Prices {
		prices = append(prices, positionPriceDTO{
			PositionID: entry.PositionID, Symbol: entry.Symbol, Price: entry.Price,
			Change24hPercent:           entry.Change24hPercent,
			DistanceToBreakEvenPercent: entry.DistanceToBreakEvenPercent,
			DistanceToTp1Percent:       entry.DistanceToTp1Percent,
		})
	}
	return priceSnapshotDTO{Prices: prices, FetchedAt: snapshot.FetchedAt, Stale: snapshot.Stale}
}
