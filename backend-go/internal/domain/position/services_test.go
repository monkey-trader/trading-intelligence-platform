package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/decimal"
)

// Fälle portiert aus services/liquidationDistanceRatio.test.ts und summarizeOpenPositions.test.ts.

func TestLiquidationDistanceRatio(t *testing.T) {
	testCases := []struct {
		name             string
		side             position.PositionSide
		markPrice        string
		liquidationPrice string
		expected         string
	}{
		// (81365.0 − 76507) / 81365.0 = 0.0597062619… → 0.05970626
		{"LONG", position.Long, "81365.0", "76507", "0.05970626"},
		// (1.0100 − 0.865) / 1.0100 = 0.1435643564… → 0.14356436
		{"LONG kaufmännisch gerundet", position.Long, "1.0100", "0.865", "0.14356436"},
		// (4410.5 − 4175.00) / 4175.00 = 0.0564071856… → 0.05640719
		{"SHORT", position.Short, "4175.00", "4410.5", "0.05640719"},
		{"LONG negativ jenseits Liquidation", position.Long, "70000", "76507", "-0.09295714"},
		{"SHORT negativ jenseits Liquidation", position.Short, "4500", "4410.5", "-0.01988889"},
		{"LONG genau auf Liquidation", position.Long, "100.0", "100", "0"},
		{"SHORT genau auf Liquidation", position.Short, "100", "100.00", "0"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ratio, err := position.LiquidationDistanceRatio(testCase.side, testCase.markPrice, testCase.liquidationPrice)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if ratio == nil {
				t.Fatalf("Ergebnis nil, erwartet %q", testCase.expected)
			}
			if *ratio != testCase.expected {
				t.Errorf("Ergebnis = %q, erwartet %q", *ratio, testCase.expected)
			}
		})
	}
}

func TestLiquidationDistanceRatio_KeinAbstand(t *testing.T) {
	testCases := []struct {
		name             string
		side             position.PositionSide
		markPrice        string
		liquidationPrice string
	}{
		{"LONG ohne Liquidationspreis", position.Long, "81365.0", "0"},
		{"SHORT ohne Liquidationspreis", position.Short, "81365.0", "0"},
		{"Liquidationspreis 0.000", position.Long, "81365.0", "0.000"},
		{"Liquidationspreis -0", position.Long, "81365.0", "-0"},
		{"LONG Mark 0", position.Long, "0", "76507"},
		{"SHORT Mark 0", position.Short, "0", "76507"},
		{"LONG Mark -1", position.Long, "-1", "1"},
		{"SHORT Mark -1", position.Short, "-1", "1"},
		{"LONG Mark -0.5", position.Long, "-0.5", "1"},
		{"SHORT Mark -0.5", position.Short, "-0.5", "1"},
		{"negativer Liquidationspreis", position.Long, "100", "-5"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ratio, err := position.LiquidationDistanceRatio(testCase.side, testCase.markPrice, testCase.liquidationPrice)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if ratio != nil {
				t.Errorf("Ergebnis = %q, erwartet nil", *ratio)
			}
		})
	}
}

func TestLiquidationDistanceRatio_UngueltigerPreis(t *testing.T) {
	testCases := []struct {
		name             string
		side             position.PositionSide
		markPrice        string
		liquidationPrice string
	}{
		{"Mark abc", position.Long, "abc", "76507"},
		{"Liquidation 1e3", position.Short, "81365.0", "1e3"},
		// Beide Preise werden immer geprüft: ein ungültiger Wert geht nie still als nil durch.
		{"Liquidation 0 und Mark ungültig", position.Long, "abc", "0"},
		{"Mark 0 und Liquidation ungültig", position.Long, "0", "x"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ratio, err := position.LiquidationDistanceRatio(testCase.side, testCase.markPrice, testCase.liquidationPrice)

			var invalidDecimal *decimal.InvalidDecimalError
			if !errors.As(err, &invalidDecimal) {
				t.Fatalf("Fehler = %v (%T), erwartet *decimal.InvalidDecimalError", err, err)
			}
			if ratio != nil {
				t.Errorf("Ergebnis = %q, erwartet nil bei Fehler", *ratio)
			}
		})
	}
}

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
