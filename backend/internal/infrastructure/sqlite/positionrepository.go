package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionRepository speichert geschlossene Positionen in SQLite.
type PositionRepository struct {
	db  *sql.DB
	now func() time.Time
}

// NewPositionRepository bekommt die Uhr für imported_at injiziert.
func NewPositionRepository(db *sql.DB, now func() time.Time) *PositionRepository {
	return &PositionRepository{db: db, now: now}
}

const upsertPositionSQL = `INSERT INTO positions (position_id, symbol, side, isolated, open_time, close_time,
	avg_open_price, avg_close_price, realised_profit, net_profit, position_amount, closed_amount,
	commission, total_funding, leverage, fully_closed, raw, imported_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(position_id) DO UPDATE SET
	symbol = excluded.symbol, side = excluded.side, isolated = excluded.isolated,
	open_time = excluded.open_time, close_time = excluded.close_time,
	avg_open_price = excluded.avg_open_price, avg_close_price = excluded.avg_close_price,
	realised_profit = excluded.realised_profit, net_profit = excluded.net_profit,
	position_amount = excluded.position_amount, closed_amount = excluded.closed_amount,
	commission = excluded.commission, total_funding = excluded.total_funding,
	leverage = excluded.leverage, fully_closed = excluded.fully_closed,
	raw = excluded.raw, imported_at = excluded.imported_at`

// UpsertClosedPosition legt die Position an oder aktualisiert sie (notes bleiben unberührt).
func (repo *PositionRepository) UpsertClosedPosition(ctx context.Context, closed position.ClosedPosition) error {
	_, err := repo.db.ExecContext(ctx, upsertPositionSQL,
		closed.PositionID, closed.Symbol, string(closed.Side), boolToInt(closed.Isolated),
		closed.OpenTime, closed.CloseTime, closed.AvgOpenPrice, closed.AvgClosePrice,
		closed.RealisedProfit, closed.NetProfit, closed.PositionAmount, closed.ClosedAmount,
		closed.Commission, closed.TotalFunding, closed.Leverage, boolToInt(closed.FullyClosed),
		closed.Raw, repo.now().UnixMilli())
	if err != nil {
		return fmt.Errorf("position %s speichern: %w", closed.PositionID, err)
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// UpsertClosedPositionBatch speichert alle Positionen in einer Transaktion (alles oder nichts).
func (repo *PositionRepository) UpsertClosedPositionBatch(ctx context.Context, positions []position.ClosedPosition) (err error) {
	transaction, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("positions-transaktion starten: %w", err)
	}
	defer func() {
		if err != nil {
			_ = transaction.Rollback()
		}
	}()
	importedAt := repo.now().UnixMilli()
	for _, closed := range positions {
		if _, err = transaction.ExecContext(ctx, upsertPositionSQL,
			closed.PositionID, closed.Symbol, string(closed.Side), boolToInt(closed.Isolated),
			closed.OpenTime, closed.CloseTime, closed.AvgOpenPrice, closed.AvgClosePrice,
			closed.RealisedProfit, closed.NetProfit, closed.PositionAmount, closed.ClosedAmount,
			closed.Commission, closed.TotalFunding, closed.Leverage, boolToInt(closed.FullyClosed),
			closed.Raw, importedAt); err != nil {
			return fmt.Errorf("position %s speichern: %w", closed.PositionID, err)
		}
	}
	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("positions-transaktion abschließen: %w", err)
	}
	return nil
}

// Platzhalter einer noch offenen Position: close_time 0, fully_closed 0. Das Schließen überschreibt
// die Zeile per Upsert mit den echten Werten (gleiche position_id, notes bleiben dran).
const recordOpenPositionSQL = `INSERT INTO positions (position_id, symbol, side, isolated, open_time, close_time,
	avg_open_price, avg_close_price, realised_profit, net_profit, position_amount, closed_amount,
	commission, total_funding, leverage, fully_closed, raw, imported_at)
VALUES (?, ?, ?, ?, ?, 0, ?, '0', '0', '0', ?, '0', '0', '0', ?, 0, ?, ?)
ON CONFLICT(position_id) DO NOTHING`

// RecordOpenPositions legt unbekannte offene Positionen an; vorhandene Zeilen bleiben unberührt.
func (repo *PositionRepository) RecordOpenPositions(ctx context.Context, positions []position.OpenPosition) (err error) {
	if len(positions) == 0 {
		return nil
	}
	transaction, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("offene-positionen-transaktion starten: %w", err)
	}
	defer func() {
		if err != nil {
			_ = transaction.Rollback()
		}
	}()
	importedAt := repo.now().UnixMilli()
	for _, open := range positions {
		raw, marshalErr := json.Marshal(open)
		if marshalErr != nil {
			err = fmt.Errorf("offene position %s serialisieren: %w", open.PositionID, marshalErr)
			return err
		}
		if _, err = transaction.ExecContext(ctx, recordOpenPositionSQL,
			open.PositionID, open.Symbol, string(open.Side), boolToInt(open.Isolated), open.OpenTime,
			open.AvgOpenPrice, open.PositionAmount, open.Leverage, string(raw), importedAt); err != nil {
			return fmt.Errorf("offene position %s speichern: %w", open.PositionID, err)
		}
	}
	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("offene-positionen-transaktion abschließen: %w", err)
	}
	return nil
}

const listClosedPositionsSQL = `SELECT position_id, symbol, side, isolated, open_time, close_time,
	avg_open_price, avg_close_price, realised_profit, net_profit, position_amount, closed_amount,
	commission, total_funding, leverage, fully_closed, raw
FROM positions
WHERE fully_closed = 1 AND close_time > 0 AND close_time >= ? AND close_time <= ? AND (? = '' OR symbol = ?)
ORDER BY close_time DESC, position_id DESC
LIMIT ?`

// ListClosedPositions liest vollständig geschlossene Positionen, neueste Schließung zuerst.
// Platzhalter offener Positionen (close_time 0, fully_closed 0) bleiben außen vor.
func (repo *PositionRepository) ListClosedPositions(ctx context.Context, query position.ClosedPositionQuery) ([]position.ClosedPosition, error) {
	rows, err := repo.db.QueryContext(ctx, listClosedPositionsSQL,
		query.From(), query.To(), query.Symbol(), query.Symbol(), query.Limit())
	if err != nil {
		return nil, fmt.Errorf("geschlossene positionen lesen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	listed := []position.ClosedPosition{}
	for rows.Next() {
		var item position.ClosedPosition
		var side string
		var isolated, fullyClosed int
		if err := rows.Scan(&item.PositionID, &item.Symbol, &side, &isolated, &item.OpenTime, &item.CloseTime,
			&item.AvgOpenPrice, &item.AvgClosePrice, &item.RealisedProfit, &item.NetProfit, &item.PositionAmount,
			&item.ClosedAmount, &item.Commission, &item.TotalFunding, &item.Leverage, &fullyClosed, &item.Raw); err != nil {
			return nil, fmt.Errorf("geschlossene position lesen: %w", err)
		}
		item.Side, item.Isolated, item.FullyClosed = position.PositionSide(side), isolated != 0, fullyClosed != 0
		listed = append(listed, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("geschlossene positionen lesen: %w", err)
	}
	return listed, nil
}
