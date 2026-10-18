// Package interfaces enthält die Ports des Chart-Kerns.
package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
)

// CandleSource liefert die neuesten Kerzen eines Symbols, aufsteigend nach Öffnungszeit.
type CandleSource interface {
	FetchCandles(ctx context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error)
}

// CandleRepository speichert Kerzen dauerhaft (Cache).
type CandleRepository interface {
	// UpsertCandles speichert oder aktualisiert Kerzen per (symbol, interval, openTime).
	UpsertCandles(ctx context.Context, candles []chart.Candle) error
	// ListCandles liefert die neuesten limit Kerzen, aufsteigend (nie nil).
	ListCandles(ctx context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error)
}

// ProtectiveOrderSource liefert die offenen SL-/TP-Orders.
type ProtectiveOrderSource interface {
	FetchProtectiveOrders(ctx context.Context) (chart.ProtectiveOrdersSnapshot, error)
}
