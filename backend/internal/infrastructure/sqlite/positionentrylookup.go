package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// PositionEntryLookup liest Seite und Einstiegspreis aus der Tabelle positions.
type PositionEntryLookup struct {
	db *sql.DB
}

// NewPositionEntryLookup erzeugt den Lookup.
func NewPositionEntryLookup(db *sql.DB) *PositionEntryLookup {
	return &PositionEntryLookup{db: db}
}

// FindEntry liefert found=false, wenn die Position fehlt.
func (lookup *PositionEntryLookup) FindEntry(ctx context.Context, positionID string) (position.PositionEntry, bool, error) {
	var side, avgOpenPrice string
	err := lookup.db.QueryRowContext(ctx, `SELECT side, avg_open_price FROM positions WHERE position_id = ?`, positionID).
		Scan(&side, &avgOpenPrice)
	if errors.Is(err, sql.ErrNoRows) {
		return position.PositionEntry{}, false, nil
	}
	if err != nil {
		return position.PositionEntry{}, false, fmt.Errorf("position %s lesen: %w", positionID, err)
	}
	return position.PositionEntry{Side: position.PositionSide(side), AvgOpenPrice: avgOpenPrice}, true, nil
}
