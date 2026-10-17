package position

import (
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/decimal"
)

// ratioScale ist die Anzahl Nachkommastellen des Liquidationsabstands.
const ratioScale = 8

// LiquidationDistanceRatio liefert den relativen Abstand zur Liquidation: LONG (mark−liq)/mark, SHORT (liq−mark)/mark, Skala 8.
// nil, wenn ein Preis ≤ 0 ist. Ungültige Preise → *decimal.InvalidDecimalError.
func LiquidationDistanceRatio(side PositionSide, markPrice, liquidationPrice string) (*string, error) {
	// Beide Preise immer prüfen, damit ein ungültiger Wert nie still als nil durchgeht.
	hasLiquidationPrice, err := isPositive(liquidationPrice)
	if err != nil {
		return nil, err
	}
	hasMarkPrice, err := isPositive(markPrice)
	if err != nil {
		return nil, err
	}
	if !hasLiquidationPrice || !hasMarkPrice {
		return nil, nil
	}

	var distance string
	if side == Long {
		distance, err = decimal.Subtract(markPrice, liquidationPrice)
	} else {
		distance, err = decimal.Subtract(liquidationPrice, markPrice)
	}
	if err != nil {
		return nil, err
	}
	ratio, err := decimal.Divide(distance, markPrice, ratioScale)
	if err != nil {
		return nil, err
	}
	return &ratio, nil
}

// isPositive prüft, ob der Betrag echt größer als 0 ist ("0.000" zählt als 0).
func isPositive(amount string) (bool, error) {
	normalized, err := decimal.Subtract(amount, "0")
	if err != nil {
		return false, err
	}
	return normalized != "0" && !strings.HasPrefix(normalized, "-"), nil
}

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
