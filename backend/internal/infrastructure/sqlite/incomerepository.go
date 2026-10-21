package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
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

// HasIncomeForSymbol prüft, ob für das Symbol (exakt) ein REALIZED_PNL in [from, to] liegt.
// Gebühren und Funding gehören auch zu noch offenen Positionen und belegen keine Schließung.
func (lookup *IncomeLookup) HasIncomeForSymbol(ctx context.Context, symbol string, from, to int64) (bool, error) {
	var exists bool
	err := lookup.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM income WHERE symbol = ? AND income_type = 'REALIZED_PNL' AND time >= ? AND time <= ?)`, symbol, from, to).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("income für %s prüfen: %w", symbol, err)
	}
	return exists, nil
}

// ListIncomeSymbols liefert die Symbole mit REALIZED_PNL (eindeutig, sortiert).
func (repo *IncomeRepository) ListIncomeSymbols(ctx context.Context) ([]string, error) {
	rows, err := repo.db.QueryContext(ctx,
		`SELECT DISTINCT symbol FROM income WHERE income_type = 'REALIZED_PNL' ORDER BY symbol`)
	if err != nil {
		return nil, fmt.Errorf("income-symbole lesen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	symbols := []string{}
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("income-symbol lesen: %w", err)
		}
		symbols = append(symbols, symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("income-symbole lesen: %w", err)
	}
	return symbols, nil
}

// ListIncomeByTypesInRange liefert Einträge der Typen mit time in [fromInclusiveMs, toExclusiveMs).
func (repo *IncomeRepository) ListIncomeByTypesInRange(ctx context.Context, types []string, fromInclusiveMs, toExclusiveMs int64) ([]position.Income, error) {
	entries := []position.Income{}
	if len(types) == 0 {
		return entries, nil
	}
	arguments := make([]any, 0, len(types)+2)
	for _, incomeType := range types {
		arguments = append(arguments, incomeType)
	}
	arguments = append(arguments, fromInclusiveMs, toExclusiveMs)
	query := `SELECT transaction_id, symbol, income_type, amount, time, asset, trade_id, raw FROM income WHERE income_type IN (` +
		strings.TrimSuffix(strings.Repeat("?,", len(types)), ",") + `) AND time >= ? AND time < ? ORDER BY time, transaction_id`
	rows, err := repo.db.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("income nach typ lesen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var entry position.Income
		if err := rows.Scan(&entry.TranID, &entry.Symbol, &entry.IncomeType, &entry.Amount, &entry.Time, &entry.Asset, &entry.TradeID, &entry.Raw); err != nil {
			return nil, fmt.Errorf("income-eintrag lesen: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("income nach typ lesen: %w", err)
	}
	return entries, nil
}
