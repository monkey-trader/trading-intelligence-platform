package account_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// Erwartet: GetAccountSummaryUseCase.WithSwingLong(interfaces.SwingLongSpanSource), AccountSummary.SwingLongUnrealizedPnl
// und SwingLongRealizedPnl. UnrealizedPnl, MonthlyRealizedPnl und Periods enthalten keine Swing-Long-Anteile.

type spanFake struct {
	spans []position.SwingLongSpan
	err   error
}

func (fake *spanFake) ListSwingLongSpans(context.Context) ([]position.SwingLongSpan, error) {
	return fake.spans, fake.err
}

var _ interfaces.SwingLongSpanSource = (*spanFake)(nil)

var swingZurich, _ = time.LoadLocation("Europe/Zurich")

// swingNow ist 2026-10-21 12:00 Zürich (wie run in getaccountsummary_test.go).
var swingNow = time.Date(2026, 10, 21, 12, 0, 0, 0, swingZurich).UnixMilli()

func daysAgo(days int) int64 { return swingNow - int64(days)*24*60*60*1000 }

func swingIncome(symbol, amount string, timeMs int64) position.Income {
	return position.Income{Symbol: symbol, Amount: amount, Time: timeMs, IncomeType: position.IncomeTypeRealizedPnl}
}

func runSwing(t *testing.T, unrealized string, spans []position.SwingLongSpan, entries []position.Income) (appaccount.AccountSummary, error) {
	t.Helper()
	source := &balanceFake{balance: account.Balance{Balance: "999", Equity: "1000", AvailableMargin: "1", UnrealizedProfit: unrealized}}
	income := &incomeFake{entries: entries, filterByTime: true}
	return appaccount.NewGetAccountSummaryUseCase(source, income, func() int64 { return swingNow }, swingZurich).
		WithSwingLong(&spanFake{spans: spans}).Execute(context.Background())
}

func TestGetAccountSummary_SwingLongTrenntUnrealisiertenPnl(t *testing.T) {
	spans := []position.SwingLongSpan{
		{PositionID: "1", Symbol: "BTC-USDT", OpenTime: daysAgo(60), UnrealizedProfit: "-200.5"},
		{PositionID: "2", Symbol: "ETH-USDT", OpenTime: daysAgo(40), CloseTime: daysAgo(30)},
	}
	summary, err := runSwing(t, "-150.5", spans, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.UnrealizedPnl != "50" || summary.SwingLongUnrealizedPnl != "-200.5" {
		t.Errorf("unrealized=%s swing=%s, erwartet 50 / -200.5", summary.UnrealizedPnl, summary.SwingLongUnrealizedPnl)
	}
	if summary.UnrealizedPnlPercent == nil || *summary.UnrealizedPnlPercent != "5" {
		t.Errorf("Prozent beziehen sich auf den Wert ohne Swing Long (50/1000): %v", summary.UnrealizedPnlPercent)
	}
}

func TestGetAccountSummary_SwingLongTrenntMonatsUndPeriodenPnl(t *testing.T) {
	spans := []position.SwingLongSpan{
		{PositionID: "1", Symbol: "BTC-USDT", OpenTime: daysAgo(100), UnrealizedProfit: "0"},
	}
	entries := []position.Income{
		swingIncome("SUI-USDT", "10", daysAgo(1)),
		swingIncome("BTC-USDT", "-40", daysAgo(1)),
		swingIncome("BTC-USDT", "-5", daysAgo(0)),
	}
	summary, err := runSwing(t, "0", spans, entries)
	if err != nil {
		t.Fatal(err)
	}
	if summary.MonthlyRealizedPnl != "10" || summary.SwingLongRealizedPnl != "-45" {
		t.Errorf("monat=%s swing=%s, erwartet 10 / -45", summary.MonthlyRealizedPnl, summary.SwingLongRealizedPnl)
	}
	for _, period := range summary.Periods {
		if period.Period == "7d" && period.RealizedPnl != "10" {
			t.Errorf("7d enthält Swing Long: %s", period.RealizedPnl)
		}
	}
}

func TestGetAccountSummary_SwingLongNaeherungRespektiertZeitraumGeschlossenerPositionen(t *testing.T) {
	spans := []position.SwingLongSpan{
		{PositionID: "1", Symbol: "BTC-USDT", OpenTime: daysAgo(5), CloseTime: daysAgo(3)},
	}
	entries := []position.Income{
		swingIncome("BTC-USDT", "-1", daysAgo(5)-1), // eins vor Eröffnung: normal
		swingIncome("BTC-USDT", "-2", daysAgo(5)),   // genau Eröffnung: Swing
		swingIncome("BTC-USDT", "-4", daysAgo(3)),   // genau Schluss: Swing
		swingIncome("BTC-USDT", "-8", daysAgo(3)+1), // eins nach Schluss: normal
	}
	summary, err := runSwing(t, "0", spans, entries)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SwingLongRealizedPnl != "-6" || summary.MonthlyRealizedPnl != "-9" {
		t.Errorf("swing=%s monat=%s, erwartet -6 / -9", summary.SwingLongRealizedPnl, summary.MonthlyRealizedPnl)
	}
}

func TestGetAccountSummary_SwingLongOffenePositionZaehltBisJetzt(t *testing.T) {
	spans := []position.SwingLongSpan{{PositionID: "1", Symbol: "BTC-USDT", OpenTime: daysAgo(2), UnrealizedProfit: "0"}}
	entries := []position.Income{
		swingIncome("BTC-USDT", "-3", daysAgo(3)), // vor Eröffnung: normal
		swingIncome("BTC-USDT", "-7", swingNow-1), // offen: Swing
	}
	summary, err := runSwing(t, "0", spans, entries)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SwingLongRealizedPnl != "-7" || summary.MonthlyRealizedPnl != "-3" {
		t.Errorf("swing=%s monat=%s, erwartet -7 / -3", summary.SwingLongRealizedPnl, summary.MonthlyRealizedPnl)
	}
}

func TestGetAccountSummary_SwingLongOhneMarkierteSummenSindNull(t *testing.T) {
	summary, err := runSwing(t, "12.5", nil, []position.Income{swingIncome("BTC-USDT", "4", daysAgo(1))})
	if err != nil {
		t.Fatal(err)
	}
	if summary.SwingLongUnrealizedPnl != "0" || summary.SwingLongRealizedPnl != "0" ||
		summary.UnrealizedPnl != "12.5" || summary.MonthlyRealizedPnl != "4" {
		t.Errorf("summary = %+v", summary)
	}
}

func TestGetAccountSummary_SwingLongOhneWithSwingLongSindFelderNull(t *testing.T) {
	summary, err := run(t, "1000", &incomeFake{entries: amounts("5")})
	if err != nil {
		t.Fatal(err)
	}
	if summary.SwingLongUnrealizedPnl != "0" || summary.SwingLongRealizedPnl != "0" {
		t.Errorf("summary = %+v", summary)
	}
}

func TestGetAccountSummary_SwingLongQuellenfehlerWerdenGemeldet(t *testing.T) {
	boom := errors.New("db kaputt")
	source := &balanceFake{balance: account.Balance{Equity: "1000"}}
	useCase := appaccount.NewGetAccountSummaryUseCase(source, &incomeFake{}, func() int64 { return swingNow }, swingZurich).
		WithSwingLong(&spanFake{err: boom})
	if _, err := useCase.Execute(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}
