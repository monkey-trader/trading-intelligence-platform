package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// SwingLongRepository speichert das Swing-Long-Flag (Tabelle position_swing_long).
type SwingLongRepository struct {
	db *sql.DB
}

// NewSwingLongRepository erzeugt das Repository.
func NewSwingLongRepository(db *sql.DB) *SwingLongRepository {
	return &SwingLongRepository{db: db}
}

// Set markiert die Position (idempotent).
func (repo *SwingLongRepository) Set(ctx context.Context, positionID string, setAt int64) error {
	_, err := repo.db.ExecContext(ctx,
		`INSERT INTO position_swing_long (position_id, set_at) VALUES (?, ?) ON CONFLICT(position_id) DO UPDATE SET set_at = excluded.set_at`,
		positionID, setAt)
	if err != nil {
		return fmt.Errorf("swing long speichern: %w", err)
	}
	return nil
}

// Unset entfernt das Flag (idempotent).
func (repo *SwingLongRepository) Unset(ctx context.Context, positionID string) error {
	if _, err := repo.db.ExecContext(ctx, `DELETE FROM position_swing_long WHERE position_id = ?`, positionID); err != nil {
		return fmt.Errorf("swing long löschen: %w", err)
	}
	return nil
}

// IsSet sagt, ob das Flag gesetzt ist.
func (repo *SwingLongRepository) IsSet(ctx context.Context, positionID string) (bool, error) {
	var found int
	err := repo.db.QueryRowContext(ctx, `SELECT 1 FROM position_swing_long WHERE position_id = ?`, positionID).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("swing long lesen: %w", err)
	}
	return true, nil
}

// ListIDs liefert alle markierten IDs.
func (repo *SwingLongRepository) ListIDs(ctx context.Context) ([]string, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT position_id FROM position_swing_long ORDER BY position_id`)
	if err != nil {
		return nil, fmt.Errorf("swing long listen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("swing long lesen: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
