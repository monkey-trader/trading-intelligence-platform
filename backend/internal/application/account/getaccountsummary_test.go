package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	accountinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartete Signaturen (programmer):
//   account.Balance{Balance, Equity, AvailableMargin string}
//   accountinterfaces.BalanceSource.FetchBalance(ctx) (account.Balance, error)
//   interfaces.IncomeRepository.ListIncomeByTypesInRange(ctx, types []string, fromInclusiveMs, toExclusiveMs int64) ([]position.Income, error)
//   appaccount.NewGetAccountSummaryUseCase(source, incomes interfaces.IncomeRepository, now func() int64, loc *time.Location).Execute(ctx) (appaccount.AccountSummary, error)
//   AccountSummary{Balance, Equity, AvailableMargin, Month, MonthlyRealizedPnl string; MonthlyPnlPercent *string}

type balanceFake struct {
	balance account.Balance
	err     error
}

func (fake *balanceFake) FetchBalance(context.Context) (account.Balance, error) {
	return fake.balance, fake.err
}

var _ accountinterfaces.BalanceSource = (*balanceFake)(nil)

type incomeFake struct {
	entries      []position.Income
	err          error
	types        []string
	fromMs, toMs int64
	calls        int
	// filterByTime: nur Einträge mit Time in [from,to) liefern (sonst alle, wie bisher).
	filterByTime bool
	ranges       [][2]int64 // alle abgefragten Fenster (from,to)
	interfaces.IncomeRepository
}

func (fake *incomeFake) ListIncomeByTypesInRange(_ context.Context, types []string, from, to int64) ([]position.Income, error) {
	fake.calls++
	fake.types, fake.fromMs, fake.toMs = types, from, to
	fake.ranges = append(fake.ranges, [2]int64{from, to})
	if !fake.filterByTime {
		return fake.entries, fake.err
	}
	var filtered []position.Income
	for _, entry := range fake.entries {
		if entry.Time >= from && entry.Time < to {
			filtered = append(filtered, entry)
		}
	}
	return filtered, fake.err
}

func amounts(values ...string) []position.Income {
	entries := make([]position.Income, 0, len(values))
	for _, value := range values {
		entries = append(entries, position.Income{Amount: value})
	}
	return entries
}

// run setzt die Equity (= Nenner der Prozente, wie die BingX-Kontoanzeige) auf equity.
func run(t *testing.T, equity string, income *incomeFake) (appaccount.AccountSummary, error) {
	t.Helper()
	zurich, _ := time.LoadLocation("Europe/Zurich")
	nowMs := time.Date(2026, 10, 21, 12, 0, 0, 0, zurich).UnixMilli()
	source := &balanceFake{balance: account.Balance{Balance: "999", Equity: equity, AvailableMargin: "1150.25"}}
	return appaccount.NewGetAccountSummaryUseCase(source, income, func() int64 { return nowMs }, zurich).Execute(context.Background())
}

func TestGetAccountSummary_HappyPath(t *testing.T) {
	zurich, _ := time.LoadLocation("Europe/Zurich")
	income := &incomeFake{entries: amounts("100.5", "-3.25", "-0.125")}
	summary, err := run(t, "1000", income)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Balance != "999" || summary.Equity != "1000" || summary.AvailableMargin != "1150.25" {
		t.Errorf("Kontowerte falsch: %+v", summary)
	}
	if summary.Month != "2026-10" || summary.MonthlyRealizedPnl != "97.125" {
		t.Errorf("Monat/PnL falsch: %+v", summary)
	}
	if summary.MonthlyPnlPercent == nil || *summary.MonthlyPnlPercent != "9.71" {
		t.Errorf("Prozent = %v, erwartet 9.71 (97.125*100/1000 = 9.7125)", summary.MonthlyPnlPercent)
	}
	wantTypes := map[string]bool{"REALIZED_PNL": true, "TRADING_FEE": true, "FUNDING_FEE": true}
	if len(income.types) != 3 {
		t.Fatalf("types = %v", income.types)
	}
	for _, incomeType := range income.types {
		if !wantTypes[incomeType] {
			t.Errorf("unerwarteter Typ %s", incomeType)
		}
	}
	monthFrom := time.Date(2026, 10, 1, 0, 0, 0, 0, zurich).UnixMilli()
	monthTo := time.Date(2026, 11, 1, 0, 0, 0, 0, zurich).UnixMilli()
	found := false
	for _, window := range income.ranges {
		found = found || window == [2]int64{monthFrom, monthTo}
	}
	if !found {
		t.Errorf("Monatsfenster [%d,%d) nicht abgefragt: %v", monthFrom, monthTo, income.ranges)
	}
}

func TestGetAccountSummary_Prozent(t *testing.T) {
	testCases := []struct {
		name, balance string
		income        []string
		pnl           string
		percent       *string
	}{
		{"half-up aufrunden", "1000", []string{"123.456"}, "123.456", ptr("12.35")},
		{"negativ", "200", []string{"-10"}, "-10", ptr("-5")},
		{"negativ half-up weg von null", "100", []string{"-1.005"}, "-1.005", ptr("-1.01")},
		{"kein Income", "500", nil, "0", ptr("0")},
		{"Balance 0 ergibt null", "0", []string{"5"}, "5", nil},
		{"Balance 0.0 ergibt null", "0.00", []string{"5"}, "5", nil},
		{"Balance 0 ohne Income", "0", nil, "0", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			summary, err := run(t, testCase.balance, &incomeFake{entries: amounts(testCase.income...)})
			if err != nil {
				t.Fatal(err)
			}
			if summary.MonthlyRealizedPnl != testCase.pnl {
				t.Errorf("PnL = %s, erwartet %s", summary.MonthlyRealizedPnl, testCase.pnl)
			}
			switch {
			case testCase.percent == nil && summary.MonthlyPnlPercent != nil:
				t.Errorf("Prozent = %s, erwartet nil", *summary.MonthlyPnlPercent)
			case testCase.percent != nil && (summary.MonthlyPnlPercent == nil || *summary.MonthlyPnlPercent != *testCase.percent):
				t.Errorf("Prozent = %v, erwartet %s", summary.MonthlyPnlPercent, *testCase.percent)
			}
		})
	}
}

func ptr(value string) *string { return &value }

func TestGetAccountSummary_Fehler(t *testing.T) {
	boom := errors.New("boom")
	t.Run("Fehler der Balance-Quelle wird durchgereicht", func(t *testing.T) {
		zurich, _ := time.LoadLocation("Europe/Zurich")
		source := &balanceFake{err: boom}
		_, err := appaccount.NewGetAccountSummaryUseCase(source, &incomeFake{}, func() int64 { return 1_800_000_000_000 }, zurich).
			Execute(context.Background())
		if !errors.Is(err, boom) {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("Unavailable-Fehler bleibt per errors.As erreichbar", func(t *testing.T) {
		zurich, _ := time.LoadLocation("Europe/Zurich")
		source := &balanceFake{err: &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}}
		_, err := appaccount.NewGetAccountSummaryUseCase(source, &incomeFake{}, func() int64 { return 1_800_000_000_000 }, zurich).
			Execute(context.Background())
		var unavailable *position.OpenPositionsUnavailableError
		if !errors.As(err, &unavailable) || unavailable.Reason != position.ReasonRateLimited {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("Fehler der Income-Abfrage wird durchgereicht", func(t *testing.T) {
		_, err := run(t, "1000", &incomeFake{err: boom})
		if !errors.Is(err, boom) {
			t.Errorf("err=%v", err)
		}
	})
	t.Run("ungültiger Income-Betrag ist Fehler", func(t *testing.T) {
		_, err := run(t, "1000", &incomeFake{entries: amounts("abc")})
		if err == nil {
			t.Error("Fehler erwartet")
		}
	})
}

func incomeAt(amount string, timeMs int64) position.Income {
	return position.Income{Amount: amount, Time: timeMs}
}

func TestGetAccountSummary_Perioden(t *testing.T) {
	zurich, _ := time.LoadLocation("Europe/Zurich")
	nowMs := time.Date(2026, 10, 21, 12, 0, 0, 0, zurich).UnixMilli()
	todayStart := time.Date(2026, 10, 21, 0, 0, 0, 0, zurich).UnixMilli()
	const day = int64(24 * 60 * 60 * 1000)

	pnlOf := func(t *testing.T, summary appaccount.AccountSummary) map[string]string {
		t.Helper()
		if len(summary.Periods) != 3 ||
			summary.Periods[0].Period != "today" || summary.Periods[1].Period != "7d" || summary.Periods[2].Period != "30d" {
			t.Fatalf("Perioden = %+v, erwartet today,7d,30d", summary.Periods)
		}
		result := map[string]string{}
		for _, period := range summary.Periods {
			result[period.Period] = period.RealizedPnl
		}
		return result
	}

	testCases := []struct {
		name    string
		entries []position.Income
		want    [3]string // today, 7d, 30d
	}{
		{"nur heute", []position.Income{incomeAt("10", todayStart+1)}, [3]string{"10", "10", "10"}},
		{"genau Tagesbeginn zählt zu heute", []position.Income{incomeAt("4", todayStart)}, [3]string{"4", "4", "4"}},
		{"1 ms vor Tagesbeginn nur 7d und 30d", []position.Income{incomeAt("5", todayStart-1)}, [3]string{"0", "5", "5"}},
		{"nur 7d", []position.Income{incomeAt("-2", nowMs-3*day)}, [3]string{"0", "-2", "-2"}},
		{"7d-Grenze inklusive", []position.Income{incomeAt("1", nowMs-7*day)}, [3]string{"0", "1", "1"}},
		{"7d + 1 ms nur 30d", []position.Income{incomeAt("3", nowMs-7*day-1)}, [3]string{"0", "0", "3"}},
		{"nur 30d", []position.Income{incomeAt("7.5", nowMs-20*day)}, [3]string{"0", "0", "7.5"}},
		{"30d + 1 ms außerhalb", []position.Income{incomeAt("9", nowMs-30*day-1)}, [3]string{"0", "0", "0"}},
		{"jetzt ist exklusiv", []position.Income{incomeAt("9", nowMs)}, [3]string{"0", "0", "0"}},
		{"gemischt", []position.Income{incomeAt("10", todayStart+5), incomeAt("-2", nowMs-3*day), incomeAt("7.5", nowMs-20*day)},
			[3]string{"10", "8", "15.5"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			summary, err := run(t, "1000", &incomeFake{entries: testCase.entries, filterByTime: true})
			if err != nil {
				t.Fatal(err)
			}
			got := pnlOf(t, summary)
			if got["today"] != testCase.want[0] || got["7d"] != testCase.want[1] || got["30d"] != testCase.want[2] {
				t.Errorf("PnL today/7d/30d = %s/%s/%s, erwartet %v", got["today"], got["7d"], got["30d"], testCase.want)
			}
		})
	}
}

func TestGetAccountSummary_PeriodenFenster(t *testing.T) {
	zurich, _ := time.LoadLocation("Europe/Zurich")
	nowMs := time.Date(2026, 10, 21, 12, 0, 0, 0, zurich).UnixMilli()
	todayStart := time.Date(2026, 10, 21, 0, 0, 0, 0, zurich).UnixMilli()
	const day = int64(24 * 60 * 60 * 1000)
	income := &incomeFake{}
	if _, err := run(t, "1000", income); err != nil {
		t.Fatal(err)
	}
	for _, want := range [][2]int64{{todayStart, nowMs}, {nowMs - 7*day, nowMs}, {nowMs - 30*day, nowMs}} {
		found := false
		for _, window := range income.ranges {
			found = found || window == want
		}
		if !found {
			t.Errorf("Fenster %v nicht abgefragt: %v", want, income.ranges)
		}
	}
}

func TestGetAccountSummary_PeriodenProzent(t *testing.T) {
	zurich, _ := time.LoadLocation("Europe/Zurich")
	nowMs := time.Date(2026, 10, 21, 12, 0, 0, 0, zurich).UnixMilli()
	todayStart := time.Date(2026, 10, 21, 0, 0, 0, 0, zurich).UnixMilli()
	testCases := []struct {
		name, equity, pnl string
		percent           *string
	}{
		// Basis = Equity - PnL
		{"Beispiel", "1100", "100", ptr("10")},           // 100*100/1000
		{"half-up", "1000", "10", ptr("1.01")},           // 1000/990 = 1.0101
		{"negativ", "900", "-100", ptr("-10")},           // -10000/1000
		{"PnL 0", "500", "0", ptr("0")},                  // 0/500
		{"Basis 0 ergibt null", "100", "100", nil},       // Equity - PnL = 0
		{"Basis negativ ergibt null", "50", "100", nil},  // -50
		{"Equity 0 ohne PnL ergibt null", "0", "0", nil}, // Basis 0
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			income := &incomeFake{entries: []position.Income{incomeAt(testCase.pnl, todayStart+1)}, filterByTime: true}
			summary, err := run(t, testCase.equity, income)
			if err != nil {
				t.Fatal(err)
			}
			if len(summary.Periods) != 3 {
				t.Fatalf("Perioden = %+v", summary.Periods)
			}
			for _, period := range summary.Periods {
				if period.RealizedPnl != testCase.pnl {
					t.Errorf("%s PnL = %s, erwartet %s", period.Period, period.RealizedPnl, testCase.pnl)
				}
				switch {
				case testCase.percent == nil && period.PnlPercent != nil:
					t.Errorf("%s Prozent = %s, erwartet nil", period.Period, *period.PnlPercent)
				case testCase.percent != nil && (period.PnlPercent == nil || *period.PnlPercent != *testCase.percent):
					t.Errorf("%s Prozent = %v, erwartet %s", period.Period, period.PnlPercent, *testCase.percent)
				}
			}
		})
	}
	_ = nowMs
}

func TestGetAccountSummary_UnrealisierterPnl(t *testing.T) {
	testCases := []struct {
		name, equity, unrealized, wantPnl string
		wantPercent                       *string
	}{
		{"Prozent der Equity", "2000", "125.08", "125.08", ptr("6.25")},
		{"Verlust", "2000", "-50", "-50", ptr("-2.5")},
		{"Wert unbekannt", "2000", "", "0", nil},
		{"Equity 0", "0", "5", "5", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			zurich, _ := time.LoadLocation("Europe/Zurich")
			nowMs := time.Date(2026, 10, 21, 12, 0, 0, 0, zurich).UnixMilli()
			source := &balanceFake{balance: account.Balance{Balance: "1", Equity: testCase.equity, AvailableMargin: "1", UnrealizedProfit: testCase.unrealized}}

			summary, err := appaccount.NewGetAccountSummaryUseCase(source, &incomeFake{}, func() int64 { return nowMs }, zurich).Execute(context.Background())

			if err != nil {
				t.Fatal(err)
			}
			if summary.UnrealizedPnl != testCase.wantPnl {
				t.Errorf("unrealisiert = %s, erwartet %s", summary.UnrealizedPnl, testCase.wantPnl)
			}
			switch {
			case testCase.wantPercent == nil && summary.UnrealizedPnlPercent != nil:
				t.Errorf("Prozent = %s, erwartet nil", *summary.UnrealizedPnlPercent)
			case testCase.wantPercent != nil && (summary.UnrealizedPnlPercent == nil || *summary.UnrealizedPnlPercent != *testCase.wantPercent):
				t.Errorf("Prozent = %v, erwartet %s", summary.UnrealizedPnlPercent, *testCase.wantPercent)
			}
		})
	}
}
