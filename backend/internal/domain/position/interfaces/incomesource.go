package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// IncomeSource liefert kontoweite Income-Einträge im Zeitraum [from, to] (beide inklusive), inklusive Paging.
type IncomeSource interface {
	FetchIncome(ctx context.Context, from, to int64) ([]position.Income, error)
}
