package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// Fälle portiert aus backend/src/domain/position/factories/OpenPositionFactory.test.ts.

func validOpenPositionInput() position.OpenPositionInput {
	return position.OpenPositionInput{
		PositionID:       "9100000000000000001",
		Symbol:           "SUI-USDT",
		Side:             "LONG",
		Isolated:         true,
		PositionAmount:   "500",
		AvgOpenPrice:     "0.9600",
		MarkPrice:        "1.0100",
		Leverage:         10,
		InitialMargin:    "48.0000",
		Margin:           "60.5000",
		PositionValue:    "505",
		UnrealizedProfit: "25.0000",
		RealisedProfit:   "-0.2400",
		PnlRatio:         "0.5208",
		LiquidationPrice: "0.865",
		OpenTime:         1791500000000,
		UpdateTime:       1791600000000,
	}
}

func TestNewOpenPosition_HappyPath(t *testing.T) {
	expected := position.OpenPosition{
		PositionID:       "9100000000000000001",
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
		UnrealizedProfit: "25.0000",
		RealisedProfit:   "-0.2400",
		PnlRatio:         "0.5208",
		LiquidationPrice: "0.865",
		OpenTime:         1791500000000,
		UpdateTime:       1791600000000,
	}

	created, err := position.NewOpenPosition(validOpenPositionInput())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if created != expected {
		t.Errorf("Position =\n%+v\nerwartet\n%+v", created, expected)
	}
}

func TestNewOpenPosition_Varianten(t *testing.T) {
	testCases := []struct {
		name   string
		modify func(*position.OpenPositionInput)
		check  func(*testing.T, position.OpenPosition)
	}{
		{
			name: "Dezimal-Strings exakt, SHORT",
			modify: func(input *position.OpenPositionInput) {
				input.Side = "SHORT"
				input.RealisedProfit = "-0.5187"
				input.UnrealizedProfit = "-6.2500"
				input.PnlRatio = "-0.0904"
				input.InitialMargin = "0.05121506352000001"
			},
			check: func(t *testing.T, created position.OpenPosition) {
				if created.Side != position.Short || created.RealisedProfit != "-0.5187" ||
					created.UnrealizedProfit != "-6.2500" || created.PnlRatio != "-0.0904" ||
					created.InitialMargin != "0.05121506352000001" {
					t.Errorf("Felder verändert: %+v", created)
				}
			},
		},
		{
			name:   "liquidationPrice 0 (keine Liquidation)",
			modify: func(input *position.OpenPositionInput) { input.LiquidationPrice = "0" },
			check: func(t *testing.T, created position.OpenPosition) {
				if created.LiquidationPrice != "0" {
					t.Errorf("LiquidationPrice = %q", created.LiquidationPrice)
				}
			},
		},
		{
			name:   "Dezimal-Hebel",
			modify: func(input *position.OpenPositionInput) { input.Leverage = 2.5 },
			check: func(t *testing.T, created position.OpenPosition) {
				if created.Leverage != 2.5 {
					t.Errorf("Leverage = %v", created.Leverage)
				}
			},
		},
		{
			name: "updateTime gleich openTime",
			modify: func(input *position.OpenPositionInput) {
				input.OpenTime = 1791500000000
				input.UpdateTime = 1791500000000
			},
			check: func(t *testing.T, created position.OpenPosition) {
				if created.UpdateTime != 1791500000000 || created.OpenTime != 1791500000000 {
					t.Errorf("Zeiten = %d/%d", created.OpenTime, created.UpdateTime)
				}
			},
		},
		{
			name:   "isolated false",
			modify: func(input *position.OpenPositionInput) { input.Isolated = false },
			check: func(t *testing.T, created position.OpenPosition) {
				if created.Isolated {
					t.Errorf("Isolated = true, erwartet false")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := validOpenPositionInput()
			testCase.modify(&input)

			created, err := position.NewOpenPosition(input)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			testCase.check(t, created)
		})
	}
}

func TestNewOpenPosition_UngueltigeFelder(t *testing.T) {
	testCases := []struct {
		name   string
		modify func(*position.OpenPositionInput)
		assert func(*testing.T, error)
	}{
		{"positionId", func(input *position.OpenPositionInput) { input.PositionID = "abc" }, assertPositionDomainError[*position.InvalidPositionIDError]},
		{"symbol", func(input *position.OpenPositionInput) { input.Symbol = "sui" }, assertPositionDomainError[*position.InvalidTradingSymbolError]},
		{"side", func(input *position.OpenPositionInput) { input.Side = "BOTH" }, assertPositionDomainError[*position.InvalidPositionSideError]},
		{"positionAmount", func(input *position.OpenPositionInput) { input.PositionAmount = "abc" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"avgOpenPrice", func(input *position.OpenPositionInput) { input.AvgOpenPrice = "" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"markPrice", func(input *position.OpenPositionInput) { input.MarkPrice = "1e3" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"initialMargin", func(input *position.OpenPositionInput) { input.InitialMargin = "1,5" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"margin", func(input *position.OpenPositionInput) { input.Margin = ".5" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"positionValue", func(input *position.OpenPositionInput) { input.PositionValue = "+1" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"unrealizedProfit", func(input *position.OpenPositionInput) { input.UnrealizedProfit = "NaN" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"realisedProfit", func(input *position.OpenPositionInput) { input.RealisedProfit = "x" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"pnlRatio", func(input *position.OpenPositionInput) { input.PnlRatio = "25%" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"liquidationPrice", func(input *position.OpenPositionInput) { input.LiquidationPrice = "1e-7" }, assertPositionDomainError[*position.InvalidDecimalAmountError]},
		{"leverage", func(input *position.OpenPositionInput) { input.Leverage = 0 }, assertPositionDomainError[*position.InvalidLeverageError]},
		{"openTime", func(input *position.OpenPositionInput) { input.OpenTime = 0 }, assertPositionDomainError[*position.InvalidTimestampError]},
		{"updateTime", func(input *position.OpenPositionInput) { input.UpdateTime = -1 }, assertPositionDomainError[*position.InvalidTimestampError]},
		{
			"updateTime vor openTime",
			func(input *position.OpenPositionInput) {
				input.OpenTime = 1791500000000
				input.UpdateTime = 1791499999999
			},
			assertPositionDomainError[*position.UpdateBeforeOpenError],
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := validOpenPositionInput()
			testCase.modify(&input)

			created, err := position.NewOpenPosition(input)

			testCase.assert(t, err)
			if created != (position.OpenPosition{}) {
				t.Errorf("bei Fehler erwartet leere Position, bekommen %+v", created)
			}
		})
	}
}

// Prüfreihenfolge wie OpenPositionFactory.ts: erst Zeiten, dann UpdateBeforeOpen, dann Felder.
func TestNewOpenPosition_Pruefreihenfolge(t *testing.T) {
	testCases := []struct {
		name   string
		modify func(*position.OpenPositionInput)
		assert func(*testing.T, error)
	}{
		{
			name: "ungültige openTime vor ungültiger positionId",
			modify: func(input *position.OpenPositionInput) {
				input.OpenTime = 0
				input.PositionID = "abc"
			},
			assert: assertPositionDomainError[*position.InvalidTimestampError],
		},
		{
			name: "UpdateBeforeOpen vor ungültigem Symbol",
			modify: func(input *position.OpenPositionInput) {
				input.UpdateTime = input.OpenTime - 1
				input.Symbol = "kaputt"
			},
			assert: assertPositionDomainError[*position.UpdateBeforeOpenError],
		},
		{
			name: "positionId vor symbol",
			modify: func(input *position.OpenPositionInput) {
				input.PositionID = "abc"
				input.Symbol = "kaputt"
			},
			assert: assertPositionDomainError[*position.InvalidPositionIDError],
		},
		{
			name: "markPrice vor leverage",
			modify: func(input *position.OpenPositionInput) {
				input.MarkPrice = "x"
				input.Leverage = 0
			},
			assert: assertPositionDomainError[*position.InvalidDecimalAmountError],
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := validOpenPositionInput()
			testCase.modify(&input)

			_, err := position.NewOpenPosition(input)

			testCase.assert(t, err)
		})
	}
}

func TestNewOpenPosition_UpdateBeforeOpenNenntWerte(t *testing.T) {
	input := validOpenPositionInput()
	input.OpenTime = 1791500000000
	input.UpdateTime = 1791499999999

	_, err := position.NewOpenPosition(input)

	var updateBeforeOpen *position.UpdateBeforeOpenError
	if !errors.As(err, &updateBeforeOpen) {
		t.Fatalf("Fehler = %v, erwartet *UpdateBeforeOpenError", err)
	}
	if updateBeforeOpen.OpenTime != 1791500000000 || updateBeforeOpen.UpdateTime != 1791499999999 ||
		updateBeforeOpen.PositionID != "9100000000000000001" {
		t.Errorf("Felder = %+v", updateBeforeOpen)
	}
}
