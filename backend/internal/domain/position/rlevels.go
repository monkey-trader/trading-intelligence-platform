package position

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

const (
	// rLevelMaxMultiple ist das höchste CRV-Ziel (3R).
	rLevelMaxMultiple = 3
	// rLevelMinMultiple ist das niedrigste angezeigte CRV-Ziel (2R).
	rLevelMinMultiple = 2
	// priceScale und percentScale begrenzen nur nicht abbrechende Dezimalbrüche; übliche Werte sind exakt.
	priceScale     = 18
	percentScale   = 8
	rMultipleScale = 4
)

// StopLossAtEntryError meldet: Der SL liegt auf dem Entry, es gibt kein Risiko.
type StopLossAtEntryError struct{ Entry, StopLoss string }

func (err *StopLossAtEntryError) Error() string {
	return fmt.Sprintf("SL %s liegt auf dem Entry %s", err.StopLoss, err.Entry)
}

// StopLossWrongSideError meldet: Der SL liegt auf der Gewinnseite des Entry.
type StopLossWrongSideError struct {
	Side            PositionSide
	Entry, StopLoss string
}

func (err *StopLossWrongSideError) Error() string {
	return fmt.Sprintf("SL %s liegt für %s auf der falschen Seite des Entry %s", err.StopLoss, err.Side, err.Entry)
}

// RLevel ist ein CRV-Level: Vielfaches des Risikos, Preis und relativer Abstand zum Entry (SHORT negativ).
type RLevel struct {
	Multiple int
	Price    string
	Percent  string
	// CoincidesWithTakeProfit ist nur beim 3R-Level im Chart gesetzt (sonst nil).
	CoincidesWithTakeProfit *bool
}

// LevelAt liefert das Level mit dem Vielfachen multiple; false, wenn es fehlt.
func LevelAt(levels []RLevel, multiple int) (RLevel, bool) {
	for _, level := range levels {
		if level.Multiple == multiple {
			return level, true
		}
	}
	return RLevel{}, false
}

// tpCoincidenceTolerance ist der größte Abstand (Anteil des Entry), bei dem ein TP als auf dem Level liegend gilt.
var tpCoincidenceTolerance = big.NewRat(1, 1000)

// TakeProfitCoincides meldet, ob ein TP-Trigger höchstens 0.1 % des Entry vom Level entfernt liegt (Grenze gehört dazu).
func TakeProfitCoincides(entry, levelPrice string, triggers []string) (bool, error) {
	entryValue, err := parseExact(entry)
	if err != nil {
		return false, err
	}
	level, err := parseExact(levelPrice)
	if err != nil {
		return false, err
	}
	limit := new(big.Rat).Mul(new(big.Rat).Abs(entryValue), tpCoincidenceTolerance)
	for _, trigger := range triggers {
		triggerValue, err := parseExact(trigger)
		if err != nil {
			return false, err
		}
		distance := new(big.Rat).Abs(new(big.Rat).Sub(triggerValue, level))
		if distance.Cmp(limit) <= 0 {
			return true, nil
		}
	}
	return false, nil
}

// CandleRange sind Hoch und Tief einer Kerze ab OpenTime (Epoch-ms).
type CandleRange struct {
	OpenTime  int64
	High, Low string
}

// PositionRisk ist das beim ersten Sehen gespeicherte Risiko einer Position.
type PositionRisk struct {
	PositionID      string
	EntryPrice      string
	InitialStopLoss string
	FirstSeenAt     int64
}

// riskDistanceRat liefert |Entry - SL| exakt; SL auf dem Entry oder auf der falschen Seite sind Fehler.
func riskDistanceRat(side PositionSide, entry, stopLoss string) (entryValue, risk *big.Rat, err error) {
	entryValue, err = parseExact(entry)
	if err != nil {
		return nil, nil, err
	}
	stopValue, err := parseExact(stopLoss)
	if err != nil {
		return nil, nil, err
	}
	risk = new(big.Rat).Sub(entryValue, stopValue)
	if side != Long {
		risk.Neg(risk)
	}
	switch risk.Sign() {
	case 0:
		return nil, nil, &StopLossAtEntryError{Entry: entry, StopLoss: stopLoss}
	case -1:
		return nil, nil, &StopLossWrongSideError{Side: side, Entry: entry, StopLoss: stopLoss}
	}
	return entryValue, risk, nil
}

// RiskDistance liefert die Risiko-Distanz (1R in Preiseinheiten).
func RiskDistance(side PositionSide, entry, initialStopLoss string) (string, error) {
	_, risk, err := riskDistanceRat(side, entry, initialStopLoss)
	if err != nil {
		return "", err
	}
	return formatRat(risk, priceScale), nil
}

// RLevels liefert die Level 2R und 3R aus Entry und ursprünglichem SL.
func RLevels(side PositionSide, entry, initialStopLoss string) ([]RLevel, error) {
	entryValue, risk, err := riskDistanceRat(side, entry, initialStopLoss)
	if err != nil {
		return nil, err
	}
	levels := make([]RLevel, 0, rLevelMaxMultiple-rLevelMinMultiple+1)
	for multiple := rLevelMinMultiple; multiple <= rLevelMaxMultiple; multiple++ {
		move := new(big.Rat).Mul(risk, big.NewRat(int64(multiple), 1))
		if side != Long {
			move.Neg(move)
		}
		price := new(big.Rat).Add(entryValue, move)
		percent := new(big.Rat).Quo(move, entryValue)
		levels = append(levels, RLevel{Multiple: multiple, Price: formatRat(price, priceScale), Percent: formatRat(percent, percentScale)})
	}
	return levels, nil
}

// CurrentRMultiple liefert den gerundeten Gewinn in R ((Mark - Entry) / Risiko, SHORT gespiegelt).
func CurrentRMultiple(side PositionSide, entry, risk, markPrice string) (string, error) {
	entryValue, err := parseExact(entry)
	if err != nil {
		return "", err
	}
	riskValue, err := parseExact(risk)
	if err != nil {
		return "", err
	}
	markValue, err := parseExact(markPrice)
	if err != nil {
		return "", err
	}
	if riskValue.Sign() == 0 {
		return "", &decimal.DivisionByZeroError{Dividend: markPrice, Divisor: risk}
	}
	move := new(big.Rat).Sub(markValue, entryValue)
	if side != Long {
		move.Neg(move)
	}
	return formatRat(move.Quo(move, riskValue), rMultipleScale), nil
}

// ReachedAt liefert den Beginn der ersten Kerze ab openTime, die das Level berührt (LONG High >=, SHORT Low <=).
// Die Kerzen müssen aufsteigend nach OpenTime sortiert sein, sonst ist "erste" nicht die früheste.
func ReachedAt(side PositionSide, levelPrice string, openTime int64, candles []CandleRange) (*int64, error) {
	level, err := parseExact(levelPrice)
	if err != nil {
		return nil, err
	}
	for _, candle := range candles {
		if candle.OpenTime < openTime {
			continue
		}
		high, err := parseExact(candle.High)
		if err != nil {
			return nil, err
		}
		low, err := parseExact(candle.Low)
		if err != nil {
			return nil, err
		}
		if (side == Long && high.Cmp(level) >= 0) || (side != Long && low.Cmp(level) <= 0) {
			reachedAt := candle.OpenTime
			return &reachedAt, nil
		}
	}
	return nil, nil
}

// formatRat rundet auf höchstens scale Nachkommastellen und kürzt Nullen ("0" statt "-0").
func formatRat(value *big.Rat, scale int) string {
	text := value.FloatString(scale)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	if text == "-0" {
		return "0"
	}
	return text
}
