package position_test

import (
	"errors"
	"math"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func intPtr(value int) *int       { return &value }
func int64Ptr(value int64) *int64 { return &value }

func TestClosedPositionQuery_Defaults(t *testing.T) {
	query, err := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{})
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if query.Limit() != 100 || query.From() != 0 || query.To() != math.MaxInt64 || query.Symbol() != "" {
		t.Errorf("Defaults: limit=%d from=%d to=%d symbol=%q", query.Limit(), query.From(), query.To(), query.Symbol())
	}
}

func TestClosedPositionQuery_Limit(t *testing.T) {
	testCases := []struct {
		name    string
		limit   int
		invalid bool
	}{
		{"0 ist ungültig", 0, true},
		{"-1 ist ungültig", -1, true},
		{"1 ist gültig", 1, false},
		{"500 ist gültig", 500, false},
		{"501 ist ungültig", 501, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query, err := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{Limit: intPtr(testCase.limit)})
			var limitErr *position.InvalidLimitError
			if testCase.invalid {
				if !errors.As(err, &limitErr) || limitErr.Value != testCase.limit {
					t.Fatalf("erwartet *InvalidLimitError(%d), war %v", testCase.limit, err)
				}
				return
			}
			if err != nil || query.Limit() != testCase.limit {
				t.Errorf("limit=%d err=%v", query.Limit(), err)
			}
		})
	}
}

func TestClosedPositionQuery_Zeitraum(t *testing.T) {
	testCases := []struct {
		name     string
		from, to int64
		invalid  bool
	}{
		{"from = to ist gültig", 1000, 1000, false},
		{"from < to ist gültig", 999, 1000, false},
		{"from > to ist ungültig", 1001, 1000, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query, err := position.NewClosedPositionQuery(position.ClosedPositionQueryInput{
				From: int64Ptr(testCase.from), To: int64Ptr(testCase.to), Symbol: "SUI-USDT"})
			var rangeErr *position.InvalidTimeRangeError
			if testCase.invalid {
				if !errors.As(err, &rangeErr) || rangeErr.From != testCase.from || rangeErr.To != testCase.to {
					t.Fatalf("erwartet *InvalidTimeRangeError, war %v", err)
				}
				return
			}
			if err != nil || query.From() != testCase.from || query.To() != testCase.to || query.Symbol() != "SUI-USDT" {
				t.Errorf("from=%d to=%d symbol=%q err=%v", query.From(), query.To(), query.Symbol(), err)
			}
		})
	}
}
