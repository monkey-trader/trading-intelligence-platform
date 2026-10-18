package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
)

// CandleRepository speichert Kerzen als Cache in SQLite (Dezimalwerte als TEXT).
type CandleRepository struct {
	db *sql.DB
}

// NewCandleRepository erzeugt das Repository.
func NewCandleRepository(db *sql.DB) *CandleRepository {
	return &CandleRepository{db: db}
}

// UpsertCandles speichert alle Kerzen in EINER Transaktion; vorhandene (symbol, interval, open_time) werden aktualisiert.
func (repo *CandleRepository) UpsertCandles(ctx context.Context, candles []chart.Candle) (err error) {
	if len(candles) == 0 {
		return nil
	}
	transaction, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("kerzen speichern: transaktion starten: %w", err)
	}
	defer func() {
		if err != nil {
			_ = transaction.Rollback()
		}
	}()
	for _, candle := range candles {
		_, err = transaction.ExecContext(ctx, `INSERT INTO candles (symbol, interval, open_time, open, high, low, close, volume)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (symbol, interval, open_time) DO UPDATE SET
				open = excluded.open, high = excluded.high, low = excluded.low,
				close = excluded.close, volume = excluded.volume`,
			candle.Symbol, string(candle.Interval), candle.OpenTime,
			candle.Open, candle.High, candle.Low, candle.Close, candle.Volume)
		if err != nil {
			return fmt.Errorf("kerze %s %s %d speichern: %w", candle.Symbol, candle.Interval, candle.OpenTime, err)
		}
	}
	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("kerzen speichern: commit: %w", err)
	}
	return nil
}

// ListCandles liefert die neuesten limit Kerzen, aufsteigend nach open_time (nie nil).
func (repo *CandleRepository) ListCandles(ctx context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT open_time, open, high, low, close, volume FROM (
			SELECT open_time, open, high, low, close, volume FROM candles
			WHERE symbol = ? AND interval = ? ORDER BY open_time DESC LIMIT ?
		) ORDER BY open_time ASC`, symbol, string(interval), limit)
	if err != nil {
		return nil, fmt.Errorf("kerzen lesen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	candles := []chart.Candle{}
	for rows.Next() {
		candle := chart.Candle{Symbol: symbol, Interval: interval}
		if err := rows.Scan(&candle.OpenTime, &candle.Open, &candle.High, &candle.Low, &candle.Close, &candle.Volume); err != nil {
			return nil, fmt.Errorf("kerze lesen: %w", err)
		}
		candles = append(candles, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kerzen lesen: %w", err)
	}
	return candles, nil
}
