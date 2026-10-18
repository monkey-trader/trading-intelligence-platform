package decimal_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Fälle portiert aus backend/src/domain/shared/decimal/decimalArithmetic.test.ts.

type binaryCase struct {
	left, right, expected string
}

func runBinary(t *testing.T, operation func(string, string) (string, error), operationName string, testCases []binaryCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.left+" "+operationName+" "+testCase.right, func(t *testing.T) {
			result, err := operation(testCase.left, testCase.right)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if result != testCase.expected {
				t.Errorf("%s %s %s = %q, erwartet %q", testCase.left, operationName, testCase.right, result, testCase.expected)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	runBinary(t, decimal.Add, "+", []binaryCase{
		// unterschiedliche Nachkommazahl, Vorzeichen
		{"1.5", "2.25", "3.75"},
		{"0.1", "0.2", "0.3"},
		{"-1.5", "0.25", "-1.25"},
		{"-1.5", "-0.25", "-1.75"},
		{"10", "0.001", "10.001"},
		{"999.999", "0.001", "1000"},
		// "0" und nie "-0"
		{"1.10", "-1.1", "0"},
		{"-0.5", "0.5", "0"},
		{"-0", "0", "0"},
		{"-0.000", "0.0", "0"},
		// Großzahlen und viele Nachkommastellen
		{"123456789012345678.123456789", "0.000000001", "123456789012345678.12345679"},
		{"0.00000000000000000001", "0.00000000000000000002", "0.00000000000000000003"},
		// Normalisierung
		{"1.50", "0", "1.5"},
		{"2.000", "0", "2"},
		{"0.0", "0.00", "0"},
		{"007.10", "0", "7.1"},
	})
}

func TestSubtract(t *testing.T) {
	runBinary(t, decimal.Subtract, "-", []binaryCase{
		{"0.3", "0.1", "0.2"},
		{"1", "2.5", "-1.5"},
		{"-1", "0.25", "-1.25"},
		{"-1", "-2.5", "1.5"},
		{"81365.0", "76507", "4858"},
		// "0" und nie "-0"
		{"-1", "-1", "0"},
		{"1.50", "1.5", "0"},
		{"0", "0.000", "0"},
		// Großzahlen
		{"100000000000000000000", "0.00000000000000000001", "99999999999999999999.99999999999999999999"},
	})
}

func TestDivide(t *testing.T) {
	testCases := []struct {
		name              string
		dividend, divisor string
		scale             int
		expected          string
	}{
		{"1/3", "1", "3", 8, "0.33333333"},
		{"2/3", "2", "3", 8, "0.66666667"},
		{"10/4", "10", "4", 8, "2.5"},
		{"Liquidationsabstand", "4858", "81365.0", 8, "0.05970626"},
		{"-2/3", "-2", "3", 8, "-0.66666667"},
		{"2/-3", "2", "-3", 8, "-0.66666667"},
		{"-2/-3", "-2", "-3", 8, "0.66666667"},
		{"Hälfte weg von 0 positiv", "0.000000005", "1", 8, "0.00000001"},
		{"Hälfte weg von 0 negativ", "-0.000000005", "1", 8, "-0.00000001"},
		{"unter der Hälfte positiv", "0.000000004", "1", 8, "0"},
		{"unter der Hälfte negativ ergibt 0 statt -0", "-0.000000004", "1", 8, "0"},
		{"Skala 2", "1", "3", 2, "0.33"},
		{"Skala 0 positiv", "5", "2", 0, "3"},
		{"Skala 0 negativ", "-5", "2", 0, "-3"},
		{"Großzahl", "123456789012345678.5", "1", 8, "123456789012345678.5"},
		{"normalisiert 3.0/1.50", "3.0", "1.50", 8, "2"},
		{"0/7", "0", "7", 8, "0"},
		{"-0/7", "-0", "7", 8, "0"},
		{"1/-8 Skala 2", "1", "-8", 2, "-0.13"},
		{"1/-8 Skala 3", "1", "-8", 3, "-0.125"},
		{"-1/8 Skala 2", "-1", "8", 2, "-0.13"},
		{"-1/-8 Skala 2", "-1", "-8", 2, "0.13"},
		{"2/3 Skala 2", "2", "3", 2, "0.67"},
		{"1.123456789/1 Skala 4", "1.123456789", "1", 4, "1.1235"},
		{"1/0.003 Skala 2", "1", "0.003", 2, "333.33"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := decimal.Divide(testCase.dividend, testCase.divisor, testCase.scale)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if result != testCase.expected {
				t.Errorf("Divide(%q, %q, %d) = %q, erwartet %q",
					testCase.dividend, testCase.divisor, testCase.scale, result, testCase.expected)
			}
		})
	}
}

func TestDivide_DivisionDurchNull(t *testing.T) {
	for _, divisor := range []string{"0", "0.000", "-0"} {
		t.Run(divisor, func(t *testing.T) {
			result, err := decimal.Divide("1", divisor, 8)

			var divisionByZero *decimal.DivisionByZeroError
			if !errors.As(err, &divisionByZero) {
				t.Fatalf("Fehler = %v (%T), erwartet *DivisionByZeroError", err, err)
			}
			if result != "" {
				t.Errorf("Ergebnis = %q, erwartet leer bei Fehler", result)
			}
		})
	}
}

func TestDivide_UngueltigeSkala(t *testing.T) {
	for _, scale := range []int{-1, -8} {
		t.Run("Skala", func(t *testing.T) {
			_, err := decimal.Divide("1", "3", scale)

			var invalidScale *decimal.InvalidScaleError
			if !errors.As(err, &invalidScale) {
				t.Fatalf("Skala %d: Fehler = %v (%T), erwartet *InvalidScaleError", scale, err, err)
			}
			if invalidScale.Scale != scale {
				t.Errorf("InvalidScaleError.Scale = %d, erwartet %d", invalidScale.Scale, scale)
			}
			var invalidDecimal *decimal.InvalidDecimalError
			if errors.As(err, &invalidDecimal) {
				t.Errorf("Skala %d: darf kein *InvalidDecimalError sein (Dividend stünde im Value)", scale)
			}
		})
	}
}

func TestUngueltigeEingaben(t *testing.T) {
	invalidAmounts := []string{"1e-7", "abc", "", "1.", ".5", "+1", " 1", "1 ", "1,5", "NaN", "--1", "1.2.3", "٣"}
	operations := []struct {
		name      string
		operation func(left, right string) (string, error)
	}{
		{"Add", decimal.Add},
		{"Subtract", decimal.Subtract},
		{"Divide", func(left, right string) (string, error) { return decimal.Divide(left, right, 8) }},
	}

	for _, operation := range operations {
		for _, invalidAmount := range invalidAmounts {
			t.Run(operation.name+"/"+invalidAmount, func(t *testing.T) {
				for _, arguments := range [][2]string{{invalidAmount, "1"}, {"1", invalidAmount}} {
					result, err := operation.operation(arguments[0], arguments[1])

					var invalidDecimal *decimal.InvalidDecimalError
					if !errors.As(err, &invalidDecimal) {
						t.Errorf("%s(%q, %q): Fehler = %v (%T), erwartet *InvalidDecimalError",
							operation.name, arguments[0], arguments[1], err, err)
					}
					if result != "" {
						t.Errorf("%s(%q, %q): Ergebnis = %q, erwartet leer bei Fehler",
							operation.name, arguments[0], arguments[1], result)
					}
				}
			})
		}
	}
}

func TestDivide_UngueltigerDivisorIstKeineDivisionDurchNull(t *testing.T) {
	_, err := decimal.Divide("1", "", 8)

	var invalidDecimal *decimal.InvalidDecimalError
	if !errors.As(err, &invalidDecimal) {
		t.Fatalf("Fehler = %v (%T), erwartet *InvalidDecimalError", err, err)
	}
	var divisionByZero *decimal.DivisionByZeroError
	if errors.As(err, &divisionByZero) {
		t.Errorf("ungültiger Divisor darf nicht als Division durch 0 gemeldet werden")
	}
}

func TestFehlertexteSindNichtLeer(t *testing.T) {
	_, invalidErr := decimal.Add("x", "1")
	_, zeroErr := decimal.Divide("1", "0", 8)
	for _, err := range []error{invalidErr, zeroErr} {
		if err == nil {
			t.Fatal("Fehler erwartet")
		}
		if err.Error() == "" {
			t.Errorf("leerer Fehlertext bei %T", err)
		}
	}
}
