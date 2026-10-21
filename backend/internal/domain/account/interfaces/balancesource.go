// Package interfaces enthält die Ports der Kontosicht.
package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
)

// BalanceSource liefert die aktuellen Kontowerte.
type BalanceSource interface {
	FetchBalance(ctx context.Context) (account.Balance, error)
}
