package position

import (
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// SummarizeOpenPositions liefert die exakte Summe des unrealisierten PnL, "0" bei leerer Liste.
func SummarizeOpenPositions(positions []OpenPosition) (string, error) {
	total := "0"
	for _, openPosition := range positions {
		sum, err := decimal.Add(total, openPosition.UnrealizedProfit)
		if err != nil {
			return "", err
		}
		total = sum
	}
	return total, nil
}
