package position_test

import (
	"errors"
	"math"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// Fälle basiert auf positionHistory.page0.json und sync.py save_position().

func validClosedPositionInput() position.ClosedPositionInput {
	return position.ClosedPositionInput{
		PositionID:     "9000000000000000000",
		Symbol:         "ONDO-USDT",
		Side:           "LONG",
		Isolated:       true,
		OpenTime:       1780000000000,
		CloseTime:      1780018017000,
		AvgOpenPrice:   "0.4120",
		AvgClosePrice:  "0.4188",
		RealisedProfit: "1.7000",
		NetProfit:      "1.5123",
		PositionAmount: "250.00",
		ClosedAmount:   "250.00",
		Leverage:       7,
		Commission:     "-0.05121506352000001",
		TotalFunding:   "-0.1356",
		FullyClosed:    true,
		Raw:            `{"positionId":"9000000000000000000"}`,
	}
}

func TestNewClosedPosition_HappyPath(t *testing.T) {
	expected := position.ClosedPosition{
		PositionID:     "9000000000000000000",
		Symbol:         "ONDO-USDT",
		Side:           position.Long,
		Isolated:       true,
		OpenTime:       1780000000000,
		CloseTime:      1780018017000,
		AvgOpenPrice:   "0.4120",
		AvgClosePrice:  "0.4188",
		RealisedProfit: "1.7000",
		NetProfit:      "1.5123",
		PositionAmount: "250.00",
		ClosedAmount:   "250.00",
		Leverage:       7,
		Commission:     "-0.05121506352000001",
		TotalFunding:   "-0.1356",
		FullyClosed:    true,
		Raw:            `{"positionId":"9000000000000000000"}`,
	}

	created, err := position.NewClosedPosition(validClosedPositionInput())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if created != expected {
		t.Errorf("Position =\n%+v\nerwartet\n%+v", created, expected)
	}
}

func TestNewClosedPosition_ShortPositionWithNegativePnL(t *testing.T) {
	input := validClosedPositionInput()
	input.Side = "SHORT"
	input.RealisedProfit = "-0.6318"
	input.NetProfit = "-0.7999"
	input.AvgOpenPrice = "0.3985"
	input.AvgClosePrice = "0.3950"
	input.Leverage = 3
	input.PositionAmount = "180.50"
	input.ClosedAmount = "180.50"

	created, err := position.NewClosedPosition(input)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if created.Side != position.Short {
		t.Errorf("Side = %s, erwartet SHORT", created.Side)
	}
	if created.RealisedProfit != "-0.6318" {
		t.Errorf("RealisedProfit = %s, erwartet -0.6318", created.RealisedProfit)
	}
	if created.Leverage != 3 {
		t.Errorf("Leverage = %v, erwartet 3", created.Leverage)
	}
}

func TestNewClosedPosition_CloseTimeBeforeOpenTime_Error(t *testing.T) {
	input := validClosedPositionInput()
	input.CloseTime = input.OpenTime - 1000

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}

	var updateBeforeOpenErr *position.UpdateBeforeOpenError
	if !errors.As(err, &updateBeforeOpenErr) {
		t.Errorf("Fehler ist nicht UpdateBeforeOpenError: %T", err)
	}
}

func TestNewClosedPosition_InvalidOpenTime_Error(t *testing.T) {
	input := validClosedPositionInput()
	input.OpenTime = 0

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}

	var invalidTsErr *position.InvalidTimestampError
	if !errors.As(err, &invalidTsErr) {
		t.Errorf("Fehler ist nicht InvalidTimestampError: %T", err)
	}
}

func TestNewClosedPosition_InvalidPositionID_Error(t *testing.T) {
	input := validClosedPositionInput()
	input.PositionID = "not-all-digits"

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}

	var invalidIDErr *position.InvalidPositionIDError
	if !errors.As(err, &invalidIDErr) {
		t.Errorf("Fehler ist nicht InvalidPositionIDError: %T", err)
	}
}

func TestNewClosedPosition_InvalidSymbol_Error(t *testing.T) {
	input := validClosedPositionInput()
	input.Symbol = "invalid"

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}

	var invalidSymbolErr *position.InvalidTradingSymbolError
	if !errors.As(err, &invalidSymbolErr) {
		t.Errorf("Fehler ist nicht InvalidTradingSymbolError: %T", err)
	}
}

func TestNewClosedPosition_InvalidLeverage_Error(t *testing.T) {
	testCases := []struct {
		name     string
		leverage float64
	}{
		{"negative", -1},
		{"zero", 0},
		{"Infinity", math.Inf(1)},
		{"NaN", math.NaN()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := validClosedPositionInput()
			input.Leverage = testCase.leverage

			_, err := position.NewClosedPosition(input)
			if err == nil {
				t.Fatalf("Fehler erwartet, aber keiner geliefert")
			}

			var invalidLevErr *position.InvalidLeverageError
			if !errors.As(err, &invalidLevErr) {
				t.Errorf("Fehler ist nicht InvalidLeverageError: %T", err)
			}
		})
	}
}

func TestNewClosedPosition_InvalidDecimalAmount_Error(t *testing.T) {
	testCases := []struct {
		name  string
		field string
		value string
	}{
		{"InvalidAvgOpenPrice", "AvgOpenPrice", "not-a-decimal"},
		{"InvalidAvgClosePrice", "AvgClosePrice", "abc"},
		{"InvalidRealisedProfit", "RealisedProfit", "1.2.3"},
		{"InvalidNetProfit", "NetProfit", "x1.5"},
		{"InvalidPositionAmount", "PositionAmount", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := validClosedPositionInput()
			switch tc.field {
			case "AvgOpenPrice":
				input.AvgOpenPrice = tc.value
			case "AvgClosePrice":
				input.AvgClosePrice = tc.value
			case "RealisedProfit":
				input.RealisedProfit = tc.value
			case "NetProfit":
				input.NetProfit = tc.value
			case "PositionAmount":
				input.PositionAmount = tc.value
			}

			_, err := position.NewClosedPosition(input)
			if err == nil {
				t.Fatalf("Fehler erwartet, aber keiner geliefert")
			}

			var invalidDecimalErr *position.InvalidDecimalAmountError
			if !errors.As(err, &invalidDecimalErr) {
				t.Errorf("Fehler ist nicht InvalidDecimalAmountError: %T", err)
			}
		})
	}
}

func TestNewClosedPosition_FirstErrorWins(t *testing.T) {
	// Mehrere Fehler: Der erste in der Validierungsreihenfolge sollte gemeldet werden.
	// Reihenfolge wie in NewOpenPosition: OpenTime, CloseTime, UpdateBeforeOpen, dann Felder.
	input := validClosedPositionInput()
	input.OpenTime = 0         // InvalidTimestampError
	input.AvgOpenPrice = "x.5" // InvalidDecimalAmountError

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}

	var invalidTsErr *position.InvalidTimestampError
	if !errors.As(err, &invalidTsErr) {
		t.Errorf("Erwarteter InvalidTimestampError (erster Fehler), aber bekam: %T", err)
	}
}

func TestNewClosedPosition_CloseTimeEqualsOpenTime_Valid(t *testing.T) {
	input := validClosedPositionInput()
	input.CloseTime = input.OpenTime

	created, err := position.NewClosedPosition(input)
	if err != nil {
		t.Fatalf("unerwarteter Fehler bei CloseTime == OpenTime: %v", err)
	}
	if created.OpenTime != input.OpenTime || created.CloseTime != input.OpenTime {
		t.Errorf("OpenTime/CloseTime = %d/%d, erwartet beide %d", created.OpenTime, created.CloseTime, input.OpenTime)
	}
}

func TestNewClosedPosition_InvalidCloseTime_Error(t *testing.T) {
	input := validClosedPositionInput()
	input.CloseTime = 0

	_, err := position.NewClosedPosition(input)
	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
	var invalidTimestampErr *position.InvalidTimestampError
	if !errors.As(err, &invalidTimestampErr) {
		t.Errorf("Fehler ist nicht InvalidTimestampError: %T", err)
	}
}
