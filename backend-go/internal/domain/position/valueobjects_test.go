package position_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// Fälle portiert aus backend/src/domain/position/valueObjects/*.test.ts.

// assertPositionDomainError prüft, dass err vom Typ Target ist und als PositionDomainError gilt.
func assertPositionDomainError[Target error](t *testing.T, err error) {
	t.Helper()
	var target Target
	if !errors.As(err, &target) {
		t.Fatalf("Fehler = %v (%T), erwartet %T", err, err, target)
	}
	var domainError position.PositionDomainError
	if !errors.As(err, &domainError) {
		t.Errorf("Fehler %T erfüllt PositionDomainError nicht", err)
	}
	if err.Error() == "" {
		t.Errorf("leerer Fehlertext bei %T", err)
	}
}

func TestNewDecimalAmount_Gueltig(t *testing.T) {
	for _, rawAmount := range []string{"-0.05121506352000001", "0", "12", "-3.5", "250.00", "0.4120", "3412.55"} {
		t.Run(rawAmount, func(t *testing.T) {
			amount, err := position.NewDecimalAmount(rawAmount)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if amount.Value() != rawAmount {
				t.Errorf("Value = %q, erwartet unverändert %q", amount.Value(), rawAmount)
			}
		})
	}
}

func TestNewDecimalAmount_Ungueltig(t *testing.T) {
	for _, rawAmount := range []string{"", "abc", "1e5", "1.", ".5", "+1", " 1", "1 ", "1,5", "--1", "NaN"} {
		t.Run(rawAmount, func(t *testing.T) {
			_, err := position.NewDecimalAmount(rawAmount)

			assertPositionDomainError[*position.InvalidDecimalAmountError](t, err)
		})
	}
}

func TestNewEpochMillis(t *testing.T) {
	t.Run("gültig", func(t *testing.T) {
		millis, err := position.NewEpochMillis(1780000000000)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if millis.Value() != 1780000000000 {
			t.Errorf("Value = %d, erwartet 1780000000000", millis.Value())
		}
	})
	t.Run("1 ist gültig", func(t *testing.T) {
		if _, err := position.NewEpochMillis(1); err != nil {
			t.Errorf("unerwarteter Fehler: %v", err)
		}
	})

	for _, rawTimestamp := range []int64{0, -1, math.MinInt64} {
		t.Run("ungültig", func(t *testing.T) {
			_, err := position.NewEpochMillis(rawTimestamp)

			assertPositionDomainError[*position.InvalidTimestampError](t, err)
		})
	}
}

func TestNewLeverage(t *testing.T) {
	for _, rawLeverage := range []float64{1, 7, 15, 2.5, 125} {
		t.Run("gültig", func(t *testing.T) {
			leverage, err := position.NewLeverage(rawLeverage)
			if err != nil {
				t.Fatalf("Hebel %v: unerwarteter Fehler: %v", rawLeverage, err)
			}
			if leverage.Value() != rawLeverage {
				t.Errorf("Value = %v, erwartet %v", leverage.Value(), rawLeverage)
			}
		})
	}

	for _, rawLeverage := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1)} {
		t.Run("ungültig", func(t *testing.T) {
			_, err := position.NewLeverage(rawLeverage)

			assertPositionDomainError[*position.InvalidLeverageError](t, err)
		})
	}
}

func TestNewPositionID(t *testing.T) {
	t.Run("19-stellige ID exakt als String", func(t *testing.T) {
		positionID, err := position.NewPositionID("9000000000000000001")
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if positionID.Value() != "9000000000000000001" {
			t.Errorf("Value = %q", positionID.Value())
		}
	})

	for _, rawID := range []string{"", "abc", "12a", "-1", "1.5", " 123", "123 ", "١٢٣"} {
		t.Run("ungültig "+rawID, func(t *testing.T) {
			_, err := position.NewPositionID(rawID)

			assertPositionDomainError[*position.InvalidPositionIDError](t, err)
		})
	}
}

func TestNewPositionSide(t *testing.T) {
	for _, testCase := range []struct {
		raw      string
		expected position.PositionSide
	}{
		{"LONG", position.Long},
		{"SHORT", position.Short},
	} {
		t.Run(testCase.raw, func(t *testing.T) {
			side, err := position.NewPositionSide(testCase.raw)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if side != testCase.expected {
				t.Errorf("Seite = %q, erwartet %q", side, testCase.expected)
			}
		})
	}

	for _, rawSide := range []string{"long", "short", "BOTH", "", "BUY", " LONG"} {
		t.Run("ungültig "+rawSide, func(t *testing.T) {
			side, err := position.NewPositionSide(rawSide)

			assertPositionDomainError[*position.InvalidPositionSideError](t, err)
			if side != "" {
				t.Errorf("Seite = %q, erwartet leer bei Fehler", side)
			}
		})
	}
}

func TestPositionSideKonstanten(t *testing.T) {
	if position.Long != "LONG" || position.Short != "SHORT" {
		t.Errorf("Long = %q, Short = %q", position.Long, position.Short)
	}
}

func TestNewTradingSymbol(t *testing.T) {
	for _, rawSymbol := range []string{"BTC-USDT", "ONDO-USDT", "NCCOGOLD2USD-USDT", "1000PEPE-USDT"} {
		t.Run(rawSymbol, func(t *testing.T) {
			symbol, err := position.NewTradingSymbol(rawSymbol)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if symbol.Value() != rawSymbol {
				t.Errorf("Value = %q, erwartet %q", symbol.Value(), rawSymbol)
			}
		})
	}

	for _, rawSymbol := range []string{"", "BTCUSDT", "btc-usdt", "BTC-", "-USDT", "BTC-USDT-X", "BTC_USDT", " BTC-USDT", "BTC-USDT\n"} {
		t.Run("ungültig "+rawSymbol, func(t *testing.T) {
			_, err := position.NewTradingSymbol(rawSymbol)

			assertPositionDomainError[*position.InvalidTradingSymbolError](t, err)
		})
	}
}

func TestOpenPositionsUnavailableError(t *testing.T) {
	allReasons := []position.UnavailableReason{
		position.ReasonNotConfigured,
		position.ReasonRateLimited,
		position.ReasonUnreachable,
		position.ReasonUpstreamError,
		position.ReasonInvalidResponse,
	}
	expectedValues := []string{"not-configured", "rate-limited", "unreachable", "upstream-error", "invalid-response"}

	for index, reason := range allReasons {
		t.Run(string(reason), func(t *testing.T) {
			if string(reason) != expectedValues[index] {
				t.Errorf("Reason = %q, erwartet %q", reason, expectedValues[index])
			}
			var err error = &position.OpenPositionsUnavailableError{Reason: reason}

			var unavailable *position.OpenPositionsUnavailableError
			if !errors.As(err, &unavailable) || unavailable.Reason != reason {
				t.Fatalf("errors.As/Reason passt nicht: %v", err)
			}
			if err.Error() == "" {
				t.Errorf("leerer Fehlertext")
			}
		})
	}

	t.Run("Ursache bleibt über Unwrap erhalten", func(t *testing.T) {
		originalError := errors.New("ursprünglicher Adapterfehler")
		err := &position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError, Cause: originalError}

		if !errors.Is(err, originalError) {
			t.Errorf("errors.Is findet die Ursache nicht")
		}
		if !errors.Is(err.Unwrap(), originalError) {
			t.Errorf("Unwrap = %v, erwartet die Ursache", err.Unwrap())
		}
	})

	t.Run("ohne Ursache liefert Unwrap nil", func(t *testing.T) {
		err := &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}
		if err.Unwrap() != nil {
			t.Errorf("Unwrap = %v, erwartet nil", err.Unwrap())
		}
	})

	t.Run("ist kein PositionDomainError", func(t *testing.T) {
		var err error = &position.OpenPositionsUnavailableError{Reason: position.ReasonUnreachable}
		var domainError position.PositionDomainError
		if errors.As(err, &domainError) {
			t.Errorf("OpenPositionsUnavailableError darf kein PositionDomainError sein")
		}
	})

	t.Run("Fehlertext nennt den Grund", func(t *testing.T) {
		err := &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}
		if got := err.Error(); got == "" || !strings.Contains(got, "rate-limited") {
			t.Errorf("Error() = %q, erwartet den Grund rate-limited", got)
		}
	})
}
