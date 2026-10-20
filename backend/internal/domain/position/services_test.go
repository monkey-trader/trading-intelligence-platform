package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Fälle portiert aus services/liquidationDistanceRatio.test.ts und summarizeOpenPositions.test.ts.

func openPositionWithProfit(positionID, unrealizedProfit string) position.OpenPosition {
	return position.OpenPosition{
		PositionID:       positionID,
		Symbol:           "SUI-USDT",
		Side:             position.Long,
		Isolated:         true,
		PositionAmount:   "500",
		AvgOpenPrice:     "0.9600",
		MarkPrice:        "1.0100",
		Leverage:         10,
		InitialMargin:    "48.0000",
		Margin:           "60.5000",
		PositionValue:    "505",
		UnrealizedProfit: unrealizedProfit,
		RealisedProfit:   "-0.2400",
		PnlRatio:         "0.5208",
		LiquidationPrice: "0.865",
		OpenTime:         1791500000000,
		UpdateTime:       1791600000000,
	}
}

func TestSummarizeOpenPositions(t *testing.T) {
	testCases := []struct {
		name      string
		positions []position.OpenPosition
		expected  string
	}{
		{
			name: "gemischte Vorzeichen exakt",
			positions: []position.OpenPosition{
				openPositionWithProfit("1", "25.0000"),
				openPositionWithProfit("2", "-6.2500"),
				openPositionWithProfit("3", "0.00000001"),
			},
			expected: "18.75000001",
		},
		{
			name:      "ohne Float-Fehler",
			positions: []position.OpenPosition{openPositionWithProfit("1", "0.1"), openPositionWithProfit("2", "0.2")},
			expected:  "0.3",
		},
		{
			name:      "0 statt -0",
			positions: []position.OpenPosition{openPositionWithProfit("1", "-3.50"), openPositionWithProfit("2", "3.5")},
			expected:  "0",
		},
		{
			name:      "eine Position wird normalisiert",
			positions: []position.OpenPosition{openPositionWithProfit("1", "25.0000")},
			expected:  "25",
		},
		{name: "leere Liste", positions: []position.OpenPosition{}, expected: "0"},
		{name: "nil", positions: nil, expected: "0"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			total, err := position.SummarizeOpenPositions(testCase.positions)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if total != testCase.expected {
				t.Errorf("Summe = %q, erwartet %q", total, testCase.expected)
			}
		})
	}
}

func TestSummarizeOpenPositions_UngueltigerPnL(t *testing.T) {
	_, err := position.SummarizeOpenPositions([]position.OpenPosition{openPositionWithProfit("1", "1e2")})

	var invalidDecimal *decimal.InvalidDecimalError
	if !errors.As(err, &invalidDecimal) {
		t.Fatalf("Fehler = %v (%T), erwartet *decimal.InvalidDecimalError", err, err)
	}
}
