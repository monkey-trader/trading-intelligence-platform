package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
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
