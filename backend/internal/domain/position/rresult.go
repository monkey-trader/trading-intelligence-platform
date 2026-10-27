package position

import "math/big"

// RResult liefert netProfit ÷ (|Entry − SL| × positionAmount) in R, auf 4 Nachkommastellen gerundet.
func RResult(netProfit, positionAmount string, risk PositionRisk) (string, error) {
	profit, err := parseExact(netProfit)
	if err != nil {
		return "", err
	}
	amount, err := parseExact(positionAmount)
	if err != nil {
		return "", err
	}
	entry, err := parseExact(risk.EntryPrice)
	if err != nil {
		return "", err
	}
	stopLoss, err := parseExact(risk.InitialStopLoss)
	if err != nil {
		return "", err
	}
	distance := new(big.Rat).Abs(new(big.Rat).Sub(entry, stopLoss))
	totalRisk := distance.Mul(distance, amount)
	if totalRisk.Sign() == 0 {
		return "", &StopLossAtEntryError{Entry: risk.EntryPrice, StopLoss: risk.InitialStopLoss}
	}
	return formatRat(profit.Quo(profit, totalRisk), rMultipleScale), nil
}
