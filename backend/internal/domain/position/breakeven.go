package position

import (
	"math/big"
	"regexp"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

var decimalPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// parseExact parst einen Dezimal-String exakt (kein float); ungültige Eingaben ergeben InvalidDecimalError.
func parseExact(raw string) (*big.Rat, error) {
	rational, ok := new(big.Rat).SetString(raw)
	if !decimalPattern.MatchString(raw) || !ok {
		return nil, &decimal.InvalidDecimalError{Value: raw}
	}
	return rational, nil
}

// TightestStopLoss liefert den engsten SL-Trigger (LONG höchster, SHORT niedrigster); nil ohne Trigger.
func TightestStopLoss(side PositionSide, triggers []string) (*string, error) {
	var best *string
	var bestValue *big.Rat
	for index := range triggers {
		value, err := parseExact(triggers[index])
		if err != nil {
			return nil, err
		}
		if bestValue == nil || (side == Long && value.Cmp(bestValue) > 0) || (side != Long && value.Cmp(bestValue) < 0) {
			best, bestValue = &triggers[index], value
		}
	}
	return copyString(best), nil
}

// NearestTakeProfit liefert den nächsten TP (LONG niedrigster über Entry, SHORT höchster unter Entry,
// sonst der dem Entry nächstgelegene); nil ohne Trigger.
func NearestTakeProfit(side PositionSide, entry string, triggers []string) (*string, error) {
	entryValue, err := parseExact(entry)
	if err != nil {
		return nil, err
	}
	var best, fallback *string
	var bestDistance, fallbackDistance *big.Rat
	for index := range triggers {
		value, err := parseExact(triggers[index])
		if err != nil {
			return nil, err
		}
		signed := new(big.Rat).Sub(value, entryValue)
		if side != Long {
			signed.Neg(signed)
		}
		distance := new(big.Rat).Abs(signed)
		if signed.Sign() > 0 && (bestDistance == nil || distance.Cmp(bestDistance) < 0) {
			best, bestDistance = &triggers[index], distance
		}
		if fallbackDistance == nil || distance.Cmp(fallbackDistance) < 0 {
			fallback, fallbackDistance = &triggers[index], distance
		}
	}
	if best == nil {
		best = fallback
	}
	return copyString(best), nil
}

// IsBreakEven ist true, wenn ein SL existiert und nicht im Verlust liegt (LONG: >= Entry, SHORT: <= Entry).
func IsBreakEven(side PositionSide, entry string, stopLoss *string) (bool, error) {
	if stopLoss == nil {
		return false, nil
	}
	entryValue, err := parseExact(entry)
	if err != nil {
		return false, err
	}
	stopValue, err := parseExact(*stopLoss)
	if err != nil {
		return false, err
	}
	comparison := stopValue.Cmp(entryValue)
	if side == Long {
		return comparison >= 0, nil
	}
	return comparison <= 0, nil
}

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	duplicate := *value
	return &duplicate
}
