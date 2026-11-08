package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// SwingLongSpanSource liefert Symbol und Zeitraum der markierten Positionen: geschlossene aus der
// Tabelle positions, offene aus dem Snapshot der offenen Positionen.
type SwingLongSpanSource struct {
	db   *sql.DB
	open interfaces.OpenPositionSource
}

// NewSwingLongSpanSource erzeugt die Quelle.
func NewSwingLongSpanSource(db *sql.DB, open interfaces.OpenPositionSource) *SwingLongSpanSource {
	return &SwingLongSpanSource{db: db, open: open}
}

// ListSwingLongSpans liefert die Zeiträume aller markierten Positionen.
func (source *SwingLongSpanSource) ListSwingLongSpans(ctx context.Context) ([]position.SwingLongSpan, error) {
	flagged, err := NewSwingLongRepository(source.db).ListIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(flagged) == 0 {
		return nil, nil
	}
	isFlagged := make(map[string]bool, len(flagged))
	for _, id := range flagged {
		isFlagged[id] = true
	}
	snapshot, err := source.open.FetchOpenPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("offene positionen für swing long lesen: %w", err)
	}
	spans := []position.SwingLongSpan{}
	openIDs := map[string]bool{}
	for _, openPosition := range snapshot.Positions {
		openIDs[openPosition.PositionID] = true
		if isFlagged[openPosition.PositionID] {
			spans = append(spans, position.SwingLongSpan{
				PositionID: openPosition.PositionID, Symbol: openPosition.Symbol,
				OpenTime: openPosition.OpenTime, UnrealizedProfit: openPosition.UnrealizedProfit,
			})
		}
	}
	rows, err := source.db.QueryContext(ctx,
		`SELECT p.position_id, p.symbol, p.open_time, p.close_time FROM positions p
		 JOIN position_swing_long s ON s.position_id = p.position_id ORDER BY p.position_id`)
	if err != nil {
		return nil, fmt.Errorf("swing-long-positionen lesen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var span position.SwingLongSpan
		if err := rows.Scan(&span.PositionID, &span.Symbol, &span.OpenTime, &span.CloseTime); err != nil {
			return nil, fmt.Errorf("swing-long-position lesen: %w", err)
		}
		if !openIDs[span.PositionID] {
			spans = append(spans, span)
		}
	}
	return spans, rows.Err()
}
