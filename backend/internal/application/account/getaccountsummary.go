// Package account enthält die Use Cases der Kontosicht.
package account

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

	SwingLongUnrealizedPnl string // unrealisierter PnL der offenen Swing-Long-Positionen, "0" ohne
	SwingLongRealizedPnl   string // realisierter Monats-PnL laut Näherung, "0" ohne
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
	spans   interfaces.SwingLongSpanSource // optional
}

// NewGetAccountSummaryUseCase bekommt Balance-Quelle, Income-Repository, Uhr (Epoch-ms) und Monatszeitzone.
func NewGetAccountSummaryUseCase(source accountinterfaces.BalanceSource, incomes interfaces.IncomeRangeReader, now func() int64, loc *time.Location) *GetAccountSummaryUseCase {
	return &GetAccountSummaryUseCase{source: source, incomes: incomes, now: now, loc: loc}
}

// WithSwingLong trennt Swing Long aus unrealisiertem und Monats-PnL (optional).
func (useCase *GetAccountSummaryUseCase) WithSwingLong(spans interfaces.SwingLongSpanSource) *GetAccountSummaryUseCase {
	useCase.spans = spans
	return useCase
}

// Execute liest Balance und Monats-Income und berechnet PnL und Prozent exakt.
func (useCase *GetAccountSummaryUseCase) Execute(ctx context.Context) (AccountSummary, error) {
	balance, err := useCase.source.FetchBalance(ctx)
	if err != nil {
		return AccountSummary{}, err
	}
	var spans []position.SwingLongSpan
	if useCase.spans != nil {
		if spans, err = useCase.spans.ListSwingLongSpans(ctx); err != nil {
			return AccountSummary{}, fmt.Errorf("swing-long-zeiträume lesen: %w", err)
		}
	}
	window := account.MonthWindow(useCase.now(), useCase.loc)
	total, swingRealized, err := useCase.sumRange(ctx, window.StartMs, window.EndExclusiveMs, spans)
	if err != nil {
		return AccountSummary{}, err
	}
	periods, err := useCase.periods(ctx, balance.Equity, spans)
	if err != nil {
		return AccountSummary{}, err
	}
	percent, err := monthlyPercent(total, balance.Equity)
	if err != nil {
		return AccountSummary{}, err
	}
	swingUnrealized, err := sumSwingLongUnrealized(spans)
	if err != nil {
		return AccountSummary{}, err
	}
	if balance.UnrealizedProfit != "" {
		if balance.UnrealizedProfit, err = decimal.Subtract(balance.UnrealizedProfit, swingUnrealized); err != nil {
			return AccountSummary{}, fmt.Errorf("unrealisierten pnl trennen: %w", err)
		}
	}
	unrealized, unrealizedPercent, err := unrealizedPnl(balance)
	if err != nil {
		return AccountSummary{}, err
	}
	return AccountSummary{
		UnrealizedPnl: unrealized, UnrealizedPnlPercent: unrealizedPercent,
		Balance: balance.Balance, Equity: balance.Equity, AvailableMargin: balance.AvailableMargin,
		Month: window.Label, MonthlyRealizedPnl: total, MonthlyPnlPercent: percent, Periods: periods,
		SwingLongUnrealizedPnl: swingUnrealized, SwingLongRealizedPnl: swingRealized,
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

// sumSwingLongUnrealized summiert den unrealisierten PnL der offenen Swing-Long-Positionen ("0" ohne).
func sumSwingLongUnrealized(spans []position.SwingLongSpan) (string, error) {
	total := "0"
	for _, span := range spans {
		if span.UnrealizedProfit == "" {
			continue
		}
		var err error
		if total, err = decimal.Add(total, span.UnrealizedProfit); err != nil {
			return "", fmt.Errorf("swing-long-pnl summieren: %w", err)
		}
	}
	return total, nil
}

// sumRange summiert den Income im Fenster [fromMs, toMs) getrennt in normal und Swing Long (Näherung).
func (useCase *GetAccountSummaryUseCase) sumRange(ctx context.Context, fromMs, toMs int64, spans []position.SwingLongSpan) (normal, swing string, err error) {
	entries, err := useCase.incomes.ListIncomeByTypesInRange(ctx, incomeTypes, fromMs, toMs)
	if err != nil {
		return "", "", fmt.Errorf("income lesen: %w", err)
	}
	normal, swing = "0", "0"
	for _, entry := range entries {
		target := &normal
		if slices.ContainsFunc(spans, func(span position.SwingLongSpan) bool { return span.Covers(entry.Symbol, entry.Time) }) {
			target = &swing
		}
		if *target, err = decimal.Add(*target, entry.Amount); err != nil {
			return "", "", fmt.Errorf("income summieren: %w", err)
		}
	}
	return normal, swing, nil
}

// periods berechnet PnL und Prozent für heute (ab 00:00 lokal), 7 und 30 Tage gleitend.
func (useCase *GetAccountSummaryUseCase) periods(ctx context.Context, equity string, spans []position.SwingLongSpan) ([]PeriodPnl, error) {
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
		pnl, _, err := useCase.sumRange(ctx, definition.fromMs, nowMs, spans)
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
