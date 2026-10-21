// Package account enthält die Use Cases der Kontosicht.
package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	accountinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// percentScale ist die Zahl der Nachkommastellen des Monatsprozents.
const percentScale = 2

// AccountSummary sind Kontowerte plus realisierter PnL des laufenden Monats (alles Dezimal-Strings).
type AccountSummary struct { //nolint:revive // Name ist durch die Tests und den API-Vertrag vorgegeben {
	Balance            string
	Equity             string
	AvailableMargin    string
	Month              string  // "YYYY-MM"
	MonthlyRealizedPnl string  // REALIZED_PNL + TRADING_FEE + FUNDING_FEE
	MonthlyPnlPercent  *string // nil bei Equity 0

	UnrealizedPnl        string  // unrealisierter PnL des Kontos, "0" wenn unbekannt
	UnrealizedPnlPercent *string // in Prozent der Equity, nil bei Equity <= 0 oder unbekanntem Wert
	Periods              []PeriodPnl
}

// PeriodPnl ist der realisierte PnL einer Periode ("today", "7d", "30d") samt Prozent.
type PeriodPnl struct {
	Period      string
	RealizedPnl string
	PnlPercent  *string // nil bei Basis (Equity - PnL) <= 0
}

const millisPerDay = int64(24 * 60 * 60 * 1000)

var incomeTypes = []string{position.IncomeTypeRealizedPnl, position.IncomeTypeTradingFee, position.IncomeTypeFundingFee}

// GetAccountSummaryUseCase baut die Kontoübersicht.
type GetAccountSummaryUseCase struct {
	source  accountinterfaces.BalanceSource
	incomes interfaces.IncomeRangeReader
	now     func() int64
	loc     *time.Location
}

// NewGetAccountSummaryUseCase bekommt Balance-Quelle, Income-Repository, Uhr (Epoch-ms) und Monatszeitzone.
func NewGetAccountSummaryUseCase(source accountinterfaces.BalanceSource, incomes interfaces.IncomeRangeReader, now func() int64, loc *time.Location) *GetAccountSummaryUseCase {
	return &GetAccountSummaryUseCase{source: source, incomes: incomes, now: now, loc: loc}
}

// Execute liest Balance und Monats-Income und berechnet PnL und Prozent exakt.
func (useCase *GetAccountSummaryUseCase) Execute(ctx context.Context) (AccountSummary, error) {
	balance, err := useCase.source.FetchBalance(ctx)
	if err != nil {
		return AccountSummary{}, err
	}
	window := account.MonthWindow(useCase.now(), useCase.loc)
	total, err := useCase.sumRange(ctx, window.StartMs, window.EndExclusiveMs)
	if err != nil {
		return AccountSummary{}, err
	}
	periods, err := useCase.periods(ctx, balance.Equity)
	if err != nil {
		return AccountSummary{}, err
	}
	percent, err := monthlyPercent(total, balance.Equity)
	if err != nil {
		return AccountSummary{}, err
	}
	unrealized, unrealizedPercent, err := unrealizedPnl(balance)
	if err != nil {
		return AccountSummary{}, err
	}
	return AccountSummary{
		UnrealizedPnl: unrealized, UnrealizedPnlPercent: unrealizedPercent,
		Balance: balance.Balance, Equity: balance.Equity, AvailableMargin: balance.AvailableMargin,
		Month: window.Label, MonthlyRealizedPnl: total, MonthlyPnlPercent: percent, Periods: periods,
	}, nil
}

// monthlyPercent liefert pnl*100/equity (half-up, 2 Stellen); nil bei Equity 0.
func monthlyPercent(pnl, equity string) (*string, error) {
	scaled, err := decimal.Multiply(pnl, "100")
	if err != nil {
		return nil, fmt.Errorf("prozent berechnen: %w", err)
	}
	percent, err := decimal.Divide(scaled, equity, percentScale)
	var divisionByZero *decimal.DivisionByZeroError
	if errors.As(err, &divisionByZero) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("prozent berechnen: %w", err)
	}
	return &percent, nil
}

// sumRange summiert den Income im Fenster [fromMs, toMs).
func (useCase *GetAccountSummaryUseCase) sumRange(ctx context.Context, fromMs, toMs int64) (string, error) {
	entries, err := useCase.incomes.ListIncomeByTypesInRange(ctx, incomeTypes, fromMs, toMs)
	if err != nil {
		return "", fmt.Errorf("income lesen: %w", err)
	}
	total := "0"
	for _, entry := range entries {
		if total, err = decimal.Add(total, entry.Amount); err != nil {
			return "", fmt.Errorf("income summieren: %w", err)
		}
	}
	return total, nil
}

// periods berechnet PnL und Prozent für heute (ab 00:00 lokal), 7 und 30 Tage gleitend.
func (useCase *GetAccountSummaryUseCase) periods(ctx context.Context, equity string) ([]PeriodPnl, error) {
	nowMs := useCase.now()
	local := time.UnixMilli(nowMs).In(useCase.loc)
	todayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, useCase.loc).UnixMilli()
	definitions := []struct {
		name   string
		fromMs int64
	}{
		{"today", todayStart},
		{"7d", nowMs - 7*millisPerDay},
		{"30d", nowMs - 30*millisPerDay},
	}
	result := make([]PeriodPnl, 0, len(definitions))
	for _, definition := range definitions {
		pnl, err := useCase.sumRange(ctx, definition.fromMs, nowMs)
		if err != nil {
			return nil, err
		}
		percent, err := periodPercent(pnl, equity)
		if err != nil {
			return nil, err
		}
		result = append(result, PeriodPnl{Period: definition.name, RealizedPnl: pnl, PnlPercent: percent})
	}
	return result, nil
}

// periodPercent liefert pnl*100/(equity-pnl) (half-up, 2 Stellen); nil bei Basis <= 0.
func periodPercent(pnl, equity string) (*string, error) {
	base, err := decimal.Subtract(equity, pnl)
	if err != nil {
		return nil, fmt.Errorf("basis berechnen: %w", err)
	}
	if strings.HasPrefix(base, "-") {
		return nil, nil
	}
	return monthlyPercent(pnl, base)
}

// unrealizedPnl liefert den unrealisierten PnL und dessen Prozent der Equity; ohne Wert "0" und nil.
func unrealizedPnl(balance account.Balance) (string, *string, error) {
	if balance.UnrealizedProfit == "" {
		return "0", nil, nil
	}
	if strings.HasPrefix(balance.Equity, "-") {
		return balance.UnrealizedProfit, nil, nil
	}
	percent, err := monthlyPercent(balance.UnrealizedProfit, balance.Equity)
	return balance.UnrealizedProfit, percent, err
}
