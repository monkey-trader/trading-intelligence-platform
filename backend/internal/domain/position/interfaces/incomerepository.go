package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// IncomeRepository speichert Income-Einträge dauerhaft.
type IncomeRepository interface {
	// UpsertIncomeBatch speichert alle Einträge in einer Transaktion (tranId ist Schlüssel).
	UpsertIncomeBatch(ctx context.Context, entries []position.Income) error
}
