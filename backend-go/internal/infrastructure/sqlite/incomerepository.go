package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

// IncomeRepository speichert Income in SQLite.
type IncomeRepository struct {
	db  *sql.DB
	now func() time.Time
}

// NewIncomeRepository bekommt die Uhr für imported_at injiziert.
func NewIncomeRepository(db *sql.DB, now func() time.Time) *IncomeRepository {
	return &IncomeRepository{db: db, now: now}
}

const upsertIncomeSQL = `INSERT INTO income (transaction_id, symbol, income_type, amount, time, asset, trade_id, raw, imported_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(transaction_id) DO UPDATE SET
	symbol = excluded.symbol, income_type = excluded.income_type, amount = excluded.amount,
	time = excluded.time, asset = excluded.asset, trade_id = excluded.trade_id,
	raw = excluded.raw, imported_at = excluded.imported_at`

// UpsertIncomeBatch speichert alle Einträge in einer Transaktion (alles oder nichts).
func (repo *IncomeRepository) UpsertIncomeBatch(ctx context.Context, entries []position.Income) (err error) {
	transaction, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("income-transaktion starten: %w", err)
	}
	defer func() {
		if err != nil {
			_ = transaction.Rollback()
		}
	}()
	importedAt := repo.now().UnixMilli()
	for _, entry := range entries {
		if _, err = transaction.ExecContext(ctx, upsertIncomeSQL, entry.TranID, entry.Symbol, entry.IncomeType,
			entry.Amount, entry.Time, entry.Asset, entry.TradeID, entry.Raw, importedAt); err != nil {
			return fmt.Errorf("income %s speichern: %w", entry.TranID, err)
		}
	}
	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("income-transaktion abschließen: %w", err)
	}
	return nil
}

// IncomeLookup beantwortet, ob Income für Symbol und Zeitraum existiert.
type IncomeLookup struct{ db *sql.DB }

// NewIncomeLookup baut den Lookup.
func NewIncomeLookup(db *sql.DB) *IncomeLookup { return &IncomeLookup{db: db} }

// HasIncomeForSymbol prüft, ob für das Symbol (exakt) ein Eintrag in [from, to] liegt.
func (lookup *IncomeLookup) HasIncomeForSymbol(ctx context.Context, symbol string, from, to int64) (bool, error) {
	var exists bool
	err := lookup.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM income WHERE symbol = ? AND time >= ? AND time <= ?)`, symbol, from, to).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("income für %s prüfen: %w", symbol, err)
	}
	return exists, nil
}
