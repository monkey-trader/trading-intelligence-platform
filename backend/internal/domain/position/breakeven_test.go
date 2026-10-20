package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

func str(value string) *string { return &value }

func equalPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func show(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

func TestTightestStopLoss(t *testing.T) {
	testCases := []struct {
		name     string
		side     position.PositionSide
		triggers []string
		expected *string
	}{
		{"LONG: höchster", position.Long, []string{"0.90", "0.95", "0.92"}, str("0.95")},
		{"SHORT: niedrigster", position.Short, []string{"1.10", "1.05", "1.20"}, str("1.05")},
		{"einzelner SL", position.Long, []string{"0.9"}, str("0.9")},
		{"kein SL", position.Long, nil, nil},
		{"leere Liste SHORT", position.Short, []string{}, nil},
		{"numerisch statt lexikografisch", position.Long, []string{"9", "10"}, str("10")},
		{"exakt, kein float64-Rundungsfehler", position.Long, []string{"0.1000000000000000001", "0.1"}, str("0.1000000000000000001")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := position.TightestStopLoss(testCase.side, testCase.triggers)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !equalPtr(got, testCase.expected) {
				t.Errorf("= %s, erwartet %s", show(got), show(testCase.expected))
			}
		})
	}
	t.Run("ungültiger Trigger ergibt InvalidDecimalError", func(t *testing.T) {
		_, err := position.TightestStopLoss(position.Long, []string{"abc"})
		var invalid *decimal.InvalidDecimalError
		if !errors.As(err, &invalid) {
			t.Fatalf("Fehler = %v, erwartet *decimal.InvalidDecimalError", err)
		}
	})
}

func TestNearestTakeProfit(t *testing.T) {
	testCases := []struct {
		name     string
		side     position.PositionSide
		entry    string
		triggers []string
		expected *string
	}{
		{"LONG: niedrigster über Entry", position.Long, "1.00", []string{"1.30", "1.10", "1.20"}, str("1.10")},
		{"SHORT: höchster unter Entry", position.Short, "1.00", []string{"0.70", "0.90", "0.80"}, str("0.90")},
		{"LONG: TP unter Entry wird ignoriert, wenn einer darüber liegt", position.Long, "1.00", []string{"0.90", "1.20"}, str("1.20")},
		{"SHORT: TP über Entry wird ignoriert, wenn einer darunter liegt", position.Short, "1.00", []string{"1.10", "0.80"}, str("0.80")},
		{"LONG nur falsch liegende: nächstgelegener zum Entry", position.Long, "1.00", []string{"0.50", "0.90", "0.70"}, str("0.90")},
		{"SHORT nur falsch liegende: nächstgelegener zum Entry", position.Short, "1.00", []string{"1.50", "1.10", "1.30"}, str("1.10")},
		{"LONG: TP exakt auf Entry zählt nicht als darüber, ist aber nächstgelegen", position.Long, "1.00", []string{"1.00"}, str("1.00")},
		{"kein TP", position.Long, "1.00", nil, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := position.NearestTakeProfit(testCase.side, testCase.entry, testCase.triggers)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !equalPtr(got, testCase.expected) {
				t.Errorf("= %s, erwartet %s", show(got), show(testCase.expected))
			}
		})
	}
	t.Run("ungültiger Entry ergibt InvalidDecimalError", func(t *testing.T) {
		_, err := position.NearestTakeProfit(position.Long, "x", []string{"1"})
		var invalid *decimal.InvalidDecimalError
		if !errors.As(err, &invalid) {
			t.Fatalf("Fehler = %v, erwartet *decimal.InvalidDecimalError", err)
		}
	})
}

func TestIsBreakEven(t *testing.T) {
	testCases := []struct {
		name     string
		side     position.PositionSide
		entry    string
		stopLoss *string
		expected bool
	}{
		{"LONG SL im Gewinn", position.Long, "1.00", str("1.05"), true},
		{"LONG SL exakt auf Entry", position.Long, "1.00", str("1.0"), true},
		{"LONG SL im Verlust", position.Long, "1.00", str("0.95"), false},
		{"LONG SL minimal unter Entry (exakt)", position.Long, "1.00", str("0.9999999999999999999"), false},
		{"SHORT SL im Gewinn", position.Short, "1.00", str("0.95"), true},
		{"SHORT SL exakt auf Entry", position.Short, "1.000", str("1"), true},
		{"SHORT SL im Verlust", position.Short, "1.00", str("1.05"), false},
		{"SHORT SL minimal über Entry (exakt)", position.Short, "1.00", str("1.0000000000000000001"), false},
		{"ohne SL", position.Long, "1.00", nil, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := position.IsBreakEven(testCase.side, testCase.entry, testCase.stopLoss)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if got != testCase.expected {
				t.Errorf("= %v, erwartet %v", got, testCase.expected)
			}
		})
	}
	t.Run("ungültiger SL ergibt InvalidDecimalError", func(t *testing.T) {
		_, err := position.IsBreakEven(position.Long, "1", str("1e2"))
		var invalid *decimal.InvalidDecimalError
		if !errors.As(err, &invalid) {
			t.Fatalf("Fehler = %v, erwartet *decimal.InvalidDecimalError", err)
		}
	})
}
