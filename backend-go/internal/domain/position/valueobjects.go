package position

import (
	"math"
	"regexp"
)

// PositionSide ist die Richtung einer Position.
type PositionSide string

// Erlaubte Seiten.
const (
	Long  PositionSide = "LONG"
	Short PositionSide = "SHORT"
)

var (
	decimalAmountPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	digitsOnlyPattern    = regexp.MustCompile(`^\d+$`)
	tradingSymbolPattern = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+$`)
)

// DecimalAmount ist ein Betrag als unveränderter Dezimal-String.
type DecimalAmount struct{ value string }

// Value liefert den Betrag.
func (amount DecimalAmount) Value() string { return amount.value }

// EpochMillis ist ein Zeitpunkt in Millisekunden, größer als 0.
type EpochMillis struct{ value int64 }

// Value liefert die Millisekunden.
func (millis EpochMillis) Value() int64 { return millis.value }

// Leverage ist ein endlicher Hebel größer als 0.
type Leverage struct{ value float64 }

// Value liefert den Hebel.
func (leverage Leverage) Value() float64 { return leverage.value }

// PositionID ist die BingX-Positions-ID, nur Ziffern, immer String.
type PositionID struct{ value string }

// Value liefert die ID.
func (positionID PositionID) Value() string { return positionID.value }

// TradingSymbol ist ein Symbol der Form BASIS-QUOTE.
type TradingSymbol struct{ value string }

// Value liefert das Symbol.
func (symbol TradingSymbol) Value() string { return symbol.value }

// NewDecimalAmount prüft ^-?\d+(\.\d+)?$ und übernimmt den Wert unverändert.
func NewDecimalAmount(rawAmount string) (DecimalAmount, error) {
	if !decimalAmountPattern.MatchString(rawAmount) {
		return DecimalAmount{}, &InvalidDecimalAmountError{Value: rawAmount}
	}
	return DecimalAmount{value: rawAmount}, nil
}

// NewEpochMillis verlangt einen Wert größer als 0.
func NewEpochMillis(rawTimestamp int64) (EpochMillis, error) {
	if rawTimestamp <= 0 {
		return EpochMillis{}, &InvalidTimestampError{Value: rawTimestamp}
	}
	return EpochMillis{value: rawTimestamp}, nil
}

// NewLeverage verlangt einen endlichen Wert größer als 0.
func NewLeverage(rawLeverage float64) (Leverage, error) {
	if math.IsNaN(rawLeverage) || math.IsInf(rawLeverage, 0) || rawLeverage <= 0 {
		return Leverage{}, &InvalidLeverageError{Value: rawLeverage}
	}
	return Leverage{value: rawLeverage}, nil
}

// NewPositionID verlangt ^\d+$.
func NewPositionID(rawID string) (PositionID, error) {
	if !digitsOnlyPattern.MatchString(rawID) {
		return PositionID{}, &InvalidPositionIDError{Value: rawID}
	}
	return PositionID{value: rawID}, nil
}

// NewPositionSide akzeptiert genau "LONG" oder "SHORT".
func NewPositionSide(rawSide string) (PositionSide, error) {
	switch PositionSide(rawSide) {
	case Long, Short:
		return PositionSide(rawSide), nil
	default:
		return "", &InvalidPositionSideError{Value: rawSide}
	}
}

// NewTradingSymbol verlangt ^[A-Z0-9]+-[A-Z0-9]+$.
func NewTradingSymbol(rawSymbol string) (TradingSymbol, error) {
	if !tradingSymbolPattern.MatchString(rawSymbol) {
		return TradingSymbol{}, &InvalidTradingSymbolError{Value: rawSymbol}
	}
	return TradingSymbol{value: rawSymbol}, nil
}
