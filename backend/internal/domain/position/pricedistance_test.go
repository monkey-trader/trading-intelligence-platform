package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Abstand in Gewinnrichtung: positiv = Level liegt noch vor dem Kurs, negativ = Kurs ist darüber hinaus.
func TestDistanceToLevelPercent(t *testing.T) {
	testCases := []struct {
		name  string
		side  position.PositionSide
		price string
		level *string
		want  *string
	}{
		{"LONG-TP über dem Kurs ist positiv", position.Long, "100", str("110"), str("10")},
		{"LONG-BE bereits überschritten ist negativ", position.Long, "100", str("90"), str("-10")},
		{"SHORT-TP unter dem Kurs ist positiv", position.Short, "200", str("190"), str("5")},
		{"SHORT-BE bereits überschritten (Level über Kurs) ist negativ", position.Short, "200", str("210"), str("-5")},
		{"Level gleich Kurs ergibt 0", position.Long, "1.5", str("1.5"), str("0")},
		{"rundet auf 2 Nachkommastellen half-up", position.Long, "3", str("3.1"), str("3.33")},
		{"rundet SHORT half-up", position.Short, "3", str("2.9"), str("3.33")},
		{"Level unbekannt ergibt nil", position.Long, "100", nil, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := position.DistanceToLevelPercent(testCase.side, testCase.price, testCase.level)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !equalPtr(got, testCase.want) {
				t.Fatalf("got %s, want %s", show(got), show(testCase.want))
			}
		})
	}
}

func TestDistanceToLevelPercentRejectsNonPositivePrice(t *testing.T) {
	for _, price := range []string{"0", "0.0", "-1"} {
		t.Run(price, func(t *testing.T) {
			_, err := position.DistanceToLevelPercent(position.Long, price, str("100"))
			var target *position.PriceMustBePositiveError
			if !errors.As(err, &target) {
				t.Fatalf("want PriceMustBePositiveError, got %v", err)
			}
		})
	}
}

func TestDistanceToLevelPercentRejectsInvalidDecimals(t *testing.T) {
	_, err := position.DistanceToLevelPercent(position.Long, "abc", str("100"))
	var target *decimal.InvalidDecimalError
	if !errors.As(err, &target) {
		t.Fatalf("want InvalidDecimalError for price, got %v", err)
	}
	_, err = position.DistanceToLevelPercent(position.Long, "100", str("x"))
	if !errors.As(err, &target) {
		t.Fatalf("want InvalidDecimalError for level, got %v", err)
	}
}

func TestDistanceToLevelPercentRejectsInvalidSide(t *testing.T) {
	_, err := position.DistanceToLevelPercent(position.PositionSide("BOTH"), "100", str("110"))
	var target *position.InvalidPositionSideError
	if !errors.As(err, &target) {
		t.Fatalf("want InvalidPositionSideError, got %v", err)
	}
}
