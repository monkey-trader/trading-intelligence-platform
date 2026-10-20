package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionRiskRepository speichert das ursprüngliche Risiko je Position (positionId als TEXT).
type PositionRiskRepository struct {
	db *sql.DB
}

// NewPositionRiskRepository erzeugt das Repository.
func NewPositionRiskRepository(db *sql.DB) *PositionRiskRepository {
	return &PositionRiskRepository{db: db}
}

// SaveIfAbsent legt den Eintrag an und überschreibt nie einen vorhandenen.
func (repo *PositionRiskRepository) SaveIfAbsent(ctx context.Context, risk position.PositionRisk) (bool, error) {
	result, err := repo.db.ExecContext(ctx, `INSERT INTO position_risk (position_id, entry_price, initial_stop_loss, first_seen_at)
		VALUES (?, ?, ?, ?) ON CONFLICT (position_id) DO NOTHING`,
		risk.PositionID, risk.EntryPrice, risk.InitialStopLoss, risk.FirstSeenAt)
	if err != nil {
		return false, fmt.Errorf("position-risiko speichern: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("position-risiko speichern: %w", err)
	}
	return affected > 0, nil
}

// Get liest den Eintrag; found=false, wenn er fehlt.
func (repo *PositionRiskRepository) Get(ctx context.Context, positionID string) (position.PositionRisk, bool, error) {
	risk := position.PositionRisk{PositionID: positionID}
	err := repo.db.QueryRowContext(ctx, `SELECT entry_price, initial_stop_loss, first_seen_at FROM position_risk WHERE position_id = ?`,
		positionID).Scan(&risk.EntryPrice, &risk.InitialStopLoss, &risk.FirstSeenAt)
	if errors.Is(err, sql.ErrNoRows) {
		return position.PositionRisk{}, false, nil
	}
	if err != nil {
		return position.PositionRisk{}, false, fmt.Errorf("position-risiko lesen: %w", err)
	}
	return risk, true, nil
}
