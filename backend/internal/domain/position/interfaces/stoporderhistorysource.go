package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// StopOrderHistorySource liefert historische STOP-Orders eines Symbols im Zeitraum (Epoch-ms).
type StopOrderHistorySource interface {
	FetchStopOrders(ctx context.Context, symbol string, fromMs, toMs int64) ([]position.StopOrderRecord, error)
}

// RateLimitedError meldet, dass die Quelle wegen eines Rate-Limits abgelehnt hat. Aufrufer sollen
// dann alle Abfragen für eine Weile aussetzen (nicht nur die betroffene Position).
type RateLimitedError struct {
	Err error
}

func (err *RateLimitedError) Error() string {
	return "Rate-Limit der Order-Historie: " + err.Err.Error()
}

func (err *RateLimitedError) Unwrap() error { return err.Err }
