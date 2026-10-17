// Package decimal rechnet exakt mit Dezimal-Strings (math/big, kein float).
package decimal

import (
	"math/big"
	"regexp"
	"strings"
)

// decimalPattern ist das erlaubte Eingabeformat (nur ASCII-Ziffern, kein Exponent).
var decimalPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// scaledDecimal ist ein Wert units / 10^scale, damit ohne float gerechnet wird.
type scaledDecimal struct {
	units *big.Int
	scale int
}

var bigTen = big.NewInt(10)

// Add liefert die exakte, normalisierte Summe zweier Dezimal-Strings.
func Add(left, right string) (string, error) {
	leftUnits, rightUnits, scale, err := parseAligned(left, right)
	if err != nil {
		return "", err
	}
	return format(scaledDecimal{units: new(big.Int).Add(leftUnits, rightUnits), scale: scale}), nil
}

// Subtract liefert die exakte, normalisierte Differenz zweier Dezimal-Strings.
func Subtract(left, right string) (string, error) {
	leftUnits, rightUnits, scale, err := parseAligned(left, right)
	if err != nil {
		return "", err
	}
	return format(scaledDecimal{units: new(big.Int).Sub(leftUnits, rightUnits), scale: scale}), nil
}

// Divide liefert dividend/divisor mit scale Nachkommastellen, half-up weg von 0, normalisiert.
func Divide(dividend, divisor string, scale int) (string, error) {
	if scale < 0 {
		return "", &InvalidScaleError{Scale: scale}
	}
	parsedDividend, err := parse(dividend)
	if err != nil {
		return "", err
	}
	parsedDivisor, err := parse(divisor)
	if err != nil {
		return "", err
	}
	if parsedDivisor.units.Sign() == 0 {
		return "", &DivisionByZeroError{Dividend: dividend, Divisor: divisor}
	}

	// dividend / divisor = (a / 10^sa) / (b / 10^sb) = a · 10^sb / (b · 10^sa)
	numerator := new(big.Int).Mul(parsedDividend.units, powerOfTen(parsedDivisor.scale+scale))
	denominator := new(big.Int).Mul(parsedDivisor.units, powerOfTen(parsedDividend.scale))
	isNegative := (numerator.Sign() < 0) != (denominator.Sign() < 0)

	absoluteNumerator := new(big.Int).Abs(numerator)
	absoluteDenominator := new(big.Int).Abs(denominator)
	quotient, remainder := new(big.Int).QuoRem(absoluteNumerator, absoluteDenominator, new(big.Int))
	if new(big.Int).Lsh(remainder, 1).Cmp(absoluteDenominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if isNegative {
		quotient.Neg(quotient)
	}
	return format(scaledDecimal{units: quotient, scale: scale}), nil
}

func parse(rawAmount string) (scaledDecimal, error) {
	if !decimalPattern.MatchString(rawAmount) {
		return scaledDecimal{}, &InvalidDecimalError{Value: rawAmount}
	}
	integerPart, fractionPart, _ := strings.Cut(rawAmount, ".")
	units, ok := new(big.Int).SetString(integerPart+fractionPart, 10)
	if !ok {
		return scaledDecimal{}, &InvalidDecimalError{Value: rawAmount}
	}
	return scaledDecimal{units: units, scale: len(fractionPart)}, nil
}

// parseAligned parst beide Werte und bringt sie auf die größere Skala.
func parseAligned(left, right string) (*big.Int, *big.Int, int, error) {
	parsedLeft, err := parse(left)
	if err != nil {
		return nil, nil, 0, err
	}
	parsedRight, err := parse(right)
	if err != nil {
		return nil, nil, 0, err
	}
	scale := max(parsedLeft.scale, parsedRight.scale)
	return rescale(parsedLeft, scale), rescale(parsedRight, scale), scale, nil
}

func rescale(value scaledDecimal, targetScale int) *big.Int {
	return new(big.Int).Mul(value.units, powerOfTen(targetScale-value.scale))
}

func powerOfTen(exponent int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(exponent)), nil)
}

// format schreibt ohne Nullen am Ende und ohne "-0".
func format(value scaledDecimal) string {
	if value.units.Sign() == 0 {
		return "0"
	}
	sign := ""
	if value.units.Sign() < 0 {
		sign = "-"
	}
	digits := new(big.Int).Abs(value.units).String()
	if len(digits) < value.scale+1 {
		digits = strings.Repeat("0", value.scale+1-len(digits)) + digits
	}
	integerPart := digits[:len(digits)-value.scale]
	fractionPart := strings.TrimRight(digits[len(digits)-value.scale:], "0")
	if fractionPart == "" {
		return sign + integerPart
	}
	return sign + integerPart + "." + fractionPart
}
