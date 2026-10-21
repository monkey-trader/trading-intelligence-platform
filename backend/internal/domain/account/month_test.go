package account_test

import (
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
)

func TestMonthWindow(t *testing.T) {
	zurich, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}
	ms := func(year int, month time.Month, day, hour, minute int, loc *time.Location) int64 {
		return time.Date(year, month, day, hour, minute, 0, 0, loc).UnixMilli()
	}
	testCases := []struct {
		name          string
		now           int64
		loc           *time.Location
		start, end    int64
		expectedLabel string
	}{
		{"Mitte Oktober (Sommerzeit, Monat mit Zeitumstellung)", ms(2026, 10, 15, 12, 0, zurich), zurich,
			ms(2026, 10, 1, 0, 0, zurich), ms(2026, 11, 1, 0, 0, zurich), "2026-10"},
		{"Winterzeit November", ms(2026, 11, 20, 8, 0, zurich), zurich,
			ms(2026, 11, 1, 0, 0, zurich), ms(2026, 12, 1, 0, 0, zurich), "2026-11"},
		{"Dezember endet im Januar des Folgejahres", ms(2026, 12, 31, 23, 59, zurich), zurich,
			ms(2026, 12, 1, 0, 0, zurich), ms(2027, 1, 1, 0, 0, zurich), "2026-12"},
		{"Januar", ms(2027, 1, 1, 0, 0, zurich), zurich,
			ms(2027, 1, 1, 0, 0, zurich), ms(2027, 2, 1, 0, 0, zurich), "2027-01"},
		{"letzte Minute Oktober Zürich", ms(2026, 10, 31, 23, 59, zurich), zurich,
			ms(2026, 10, 1, 0, 0, zurich), ms(2026, 11, 1, 0, 0, zurich), "2026-10"},
		{"Monatswechsel: 31.10. 23:30 UTC ist in Zürich schon 1.11.", ms(2026, 10, 31, 23, 30, time.UTC), zurich,
			ms(2026, 11, 1, 0, 0, zurich), ms(2026, 12, 1, 0, 0, zurich), "2026-11"},
		{"Sommerzeit März: 1.3. bis 1.4.", ms(2026, 3, 29, 12, 0, zurich), zurich,
			ms(2026, 3, 1, 0, 0, zurich), ms(2026, 4, 1, 0, 0, zurich), "2026-03"},
		{"UTC Schaltjahr Februar", ms(2028, 2, 29, 10, 0, time.UTC), time.UTC,
			ms(2028, 2, 1, 0, 0, time.UTC), ms(2028, 3, 1, 0, 0, time.UTC), "2028-02"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			window := account.MonthWindow(testCase.now, testCase.loc)
			if window.StartMs != testCase.start || window.EndExclusiveMs != testCase.end || window.Label != testCase.expectedLabel {
				t.Errorf("got %+v, erwartet start=%d end=%d label=%s", window, testCase.start, testCase.end, testCase.expectedLabel)
			}
		})
	}
}

func TestMonthWindow_Grenzen(t *testing.T) {
	zurich, _ := time.LoadLocation("Europe/Zurich")
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, zurich).UnixMilli()
	end := time.Date(2026, 11, 1, 0, 0, 0, 0, zurich).UnixMilli()
	if got := account.MonthWindow(start, zurich); got.Label != "2026-10" || got.StartMs != start {
		t.Errorf("genau Monatsbeginn gehört zum Monat: %+v", got)
	}
	if got := account.MonthWindow(start-1, zurich); got.Label != "2026-09" {
		t.Errorf("1 ms vor Beginn ist September: %+v", got)
	}
	if got := account.MonthWindow(end-1, zurich); got.Label != "2026-10" {
		t.Errorf("1 ms vor Ende ist noch Oktober: %+v", got)
	}
	if got := account.MonthWindow(end, zurich); got.Label != "2026-11" {
		t.Errorf("genau Ende (exklusiv) ist November: %+v", got)
	}
}
