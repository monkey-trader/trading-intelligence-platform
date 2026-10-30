package position

import (
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// DistanceToLevelPercent liefert den Abstand in Gewinnrichtung in Prozent (2 Nachkommastellen,
// half-up, normalisiert): LONG (level − price) ÷ price, SHORT (price − level) ÷ price.
// Ohne Level (nil) ist das Ergebnis nil.
func DistanceToLevelPercent(side PositionSide, price string, level *string) (*string, error) {
	if _, err := NewPositionSide(string(side)); err != nil {
		return nil, err
	}
	normalizedPrice, err := decimal.Subtract(price, "0")
	if err != nil {
		return nil, err
	}
	if normalizedPrice == "0" || strings.HasPrefix(normalizedPrice, "-") {
		return nil, &PriceMustBePositiveError{Value: price}
	}
	if level == nil {
		return nil, nil
	}
	var difference string
	if side == Long {
		difference, err = decimal.Subtract(*level, price)
	} else {
		difference, err = decimal.Subtract(price, *level)
	}
	if err != nil {
		return nil, err
	}
	scaledDifference, err := decimal.Multiply(difference, "100")
	if err != nil {
		return nil, err
	}
	percent, err := decimal.Divide(scaledDifference, price, 2)
	if err != nil {
		return nil, err
	}
	return &percent, nil
}

// PriceMustBePositiveError meldet: Der Kurs ist nicht größer als 0.
type PriceMustBePositiveError struct{ Value string }

func (err *PriceMustBePositiveError) Error() string { return "price must be positive: " + err.Value }

func (err *PriceMustBePositiveError) positionDomainError() {}
