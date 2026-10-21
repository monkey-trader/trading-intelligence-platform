package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionInitialStopLossRepository speichert manuelle ursprüngliche SLs (positionId als TEXT).
type PositionInitialStopLossRepository struct {
	db *sql.DB
}

// NewPositionInitialStopLossRepository erzeugt das Repository.
func NewPositionInitialStopLossRepository(db *sql.DB) *PositionInitialStopLossRepository {
	return &PositionInitialStopLossRepository{db: db}
}

// Set legt den Override an oder ersetzt ihn.
func (repo *PositionInitialStopLossRepository) Set(ctx context.Context, override position.InitialStopLossOverride) error {
	_, err := repo.db.ExecContext(ctx, `INSERT INTO position_initial_stop_override (position_id, entry_price, initial_stop_loss, set_at)
		VALUES (?, ?, ?, ?) ON CONFLICT (position_id) DO UPDATE SET
		entry_price = excluded.entry_price, initial_stop_loss = excluded.initial_stop_loss, set_at = excluded.set_at`,
		override.PositionID, override.EntryPrice, override.InitialStopLoss, override.SetAt)
	if err != nil {
		return fmt.Errorf("ursprungs-sl speichern: %w", err)
	}
	return nil
}

// Get liest den Override; found=false, wenn er fehlt.
func (repo *PositionInitialStopLossRepository) Get(ctx context.Context, positionID string) (position.InitialStopLossOverride, bool, error) {
	override := position.InitialStopLossOverride{PositionID: positionID}
	err := repo.db.QueryRowContext(ctx, `SELECT entry_price, initial_stop_loss, set_at FROM position_initial_stop_override WHERE position_id = ?`,
		positionID).Scan(&override.EntryPrice, &override.InitialStopLoss, &override.SetAt)
	if errors.Is(err, sql.ErrNoRows) {
		return position.InitialStopLossOverride{}, false, nil
	}
	if err != nil {
		return position.InitialStopLossOverride{}, false, fmt.Errorf("ursprungs-sl lesen: %w", err)
	}
	return override, true, nil
}

// Delete entfernt den Override (idempotent).
func (repo *PositionInitialStopLossRepository) Delete(ctx context.Context, positionID string) error {
	if _, err := repo.db.ExecContext(ctx, `DELETE FROM position_initial_stop_override WHERE position_id = ?`, positionID); err != nil {
		return fmt.Errorf("ursprungs-sl löschen: %w", err)
	}
	return nil
}
