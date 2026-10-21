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

// IncomeRangeReader liest Income-Einträge nach Typ aus einem Zeitfenster.
type IncomeRangeReader interface {
	// ListIncomeByTypesInRange liefert Einträge der Typen im Fenster [fromInclusiveMs, toExclusiveMs).
	ListIncomeByTypesInRange(ctx context.Context, types []string, fromInclusiveMs, toExclusiveMs int64) ([]position.Income, error)
}
