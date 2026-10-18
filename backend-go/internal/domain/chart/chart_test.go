package chart_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/decimal"
)

func TestParseCandleInterval(t *testing.T) {
	validCases := map[string]chart.CandleInterval{
		"5m": chart.Interval5m, "15m": chart.Interval15m, "1h": chart.Interval1h,
		"4h": chart.Interval4h, "1d": chart.Interval1d,
	}
	for raw, expected := range validCases {
		t.Run("gültig "+raw, func(t *testing.T) {
			interval, err := chart.ParseCandleInterval(raw)
			if err != nil || interval != expected || string(interval) != raw {
				t.Errorf("ParseCandleInterval(%q) = %q, %v", raw, interval, err)
			}
		})
	}
	// Nur die fünf Spec-Werte: BingX kennt mehr (1m, 1w ...), die API nicht. Groß-/Kleinschreibung zählt.
	for _, raw := range []string{"", "1m", "30m", "1w", "1M", "1H", "15M", " 15m", "15"} {
		t.Run("ungültig "+raw, func(t *testing.T) {
			_, err := chart.ParseCandleInterval(raw)
			var unsupported *chart.UnsupportedIntervalError
			if !errors.As(err, &unsupported) {
				t.Fatalf("erwartet *UnsupportedIntervalError, erhalten %T", err)
			}
			if unsupported.Value != raw {
				t.Errorf("Value = %q, erwartet %q", unsupported.Value, raw)
			}
		})
	}
}

func TestStandardwerteUndGrenzen(t *testing.T) {
	if chart.DefaultInterval != chart.Interval15m || chart.DefaultLimit != 300 ||
		chart.MinLimit != 1 || chart.MaxLimit != 1000 {
		t.Errorf("Standardwerte: %q %d %d %d", chart.DefaultInterval, chart.DefaultLimit, chart.MinLimit, chart.MaxLimit)
	}
}

func TestParseLimit_Grenzen(t *testing.T) {
	for _, limit := range []int{1, 2, 300, 999, 1000} {
		if got, err := chart.ParseLimit(limit); err != nil || got != limit {
			t.Errorf("ParseLimit(%d) = %d, %v", limit, got, err)
		}
	}
	for _, limit := range []int{-1, 0, 1001, 5000} {
		_, err := chart.ParseLimit(limit)
		var outOfRange *chart.LimitOutOfRangeError
		if !errors.As(err, &outOfRange) || outOfRange.Value != limit {
			t.Errorf("ParseLimit(%d): erwartet *LimitOutOfRangeError mit Value, erhalten %v", limit, err)
		}
	}
}

func TestParseTriggerPrice(t *testing.T) {
	for _, raw := range []string{"0.8875", "1", "0.00000001", "83557.1"} {
		if got, err := chart.ParseTriggerPrice(raw); err != nil || got != raw {
			t.Errorf("ParseTriggerPrice(%q) = %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"0", "0.0000", "-0.5", "-1"} {
		_, err := chart.ParseTriggerPrice(raw)
		var notPositive *chart.TriggerPriceMustBePositiveError
		if !errors.As(err, &notPositive) || notPositive.Value != raw {
			t.Errorf("ParseTriggerPrice(%q): erwartet *TriggerPriceMustBePositiveError, erhalten %v", raw, err)
		}
	}
	for _, raw := range []string{"", "abc", "1e3", "1,5"} {
		_, err := chart.ParseTriggerPrice(raw)
		var invalid *decimal.InvalidDecimalError
		if !errors.As(err, &invalid) {
			t.Errorf("ParseTriggerPrice(%q): erwartet *decimal.InvalidDecimalError, erhalten %v", raw, err)
		}
	}
}

func TestDistanceRatio(t *testing.T) {
	testCases := []struct {
		name, entry, trigger, expected string
	}{
		// Handrechnung SUI-Fixture, Entry 0.9600: (0.8875-0.96)/0.96 = -0.0725/0.96 = -0.0755208333...
		{"SL einer LONG unter Entry ist negativ", "0.9600", "0.8875", "-0.07552083"},
		{"TP 1.2000", "0.9600", "1.2000", "0.25000000"},
		{"TP 1.3500", "0.9600", "1.3500", "0.40625000"},
		{"Trigger gleich Entry", "0.9600", "0.9600", "0.00000000"},
		{"SHORT: SL über Entry positiv", "100", "107.5", "0.07500000"},
		{"Rundung half-up weg von 0 auf 8 Stellen", "3", "4", "0.33333333"},
		{"Rundung negativ", "3", "2", "-0.33333333"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := chart.DistanceRatio(testCase.entry, testCase.trigger)
			if err != nil || got != testCase.expected {
				t.Errorf("DistanceRatio(%s, %s) = %q, %v; erwartet %q", testCase.entry, testCase.trigger, got, err, testCase.expected)
			}
		})
	}
}

func TestDistanceRatio_Fehler(t *testing.T) {
	_, err := chart.DistanceRatio("0", "1")
	var divisionByZero *decimal.DivisionByZeroError
	if !errors.As(err, &divisionByZero) {
		t.Errorf("Entry 0: erwartet *decimal.DivisionByZeroError, erhalten %v", err)
	}
	_, err = chart.DistanceRatio("abc", "1")
	var invalid *decimal.InvalidDecimalError
	if !errors.As(err, &invalid) {
		t.Errorf("Entry ungültig: erwartet *decimal.InvalidDecimalError, erhalten %v", err)
	}
}

func TestFehlerTexteUndUnwrap(t *testing.T) {
	cause := errors.New("ursache")
	unavailable := &chart.DataUnavailableError{Reason: "unreachable", Cause: cause}
	if !errors.Is(unavailable, cause) {
		t.Error("DataUnavailableError muss die Ursache über Unwrap freigeben")
	}
	for _, err := range []error{
		unavailable, &chart.PositionNotFoundError{PositionID: "1"}, &chart.UnsupportedIntervalError{Value: "x"},
		&chart.LimitOutOfRangeError{Value: 0}, &chart.TriggerPriceMustBePositiveError{Value: "0"},
		&chart.KlineParameterError{Code: 109400, Msg: "m"}, &chart.SymbolNotFoundError{Symbol: "FOO-USDT"},
	} {
		if err.Error() == "" {
			t.Errorf("%T hat keinen Fehlertext", err)
		}
	}
}
