package position

import (
	"context"
	"fmt"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

// ImportStartedAtKey hält den Zeitpunkt des ersten Serverstarts (Epoch-ms). Davor liegende
// Historie wird nicht importiert.
const ImportStartedAtKey = "import_started_at"

// ResolveImportStart liefert den Startpunkt des Imports: beim ersten Aufruf jetzt (wird gespeichert),
// danach immer derselbe Zeitpunkt.
func ResolveImportStart(ctx context.Context, state interfaces.ImportStateRepository, now time.Time) (int64, error) {
	saved, found, err := state.Get(ctx, ImportStartedAtKey)
	if err != nil {
		return 0, fmt.Errorf("import-start lesen: %w", err)
	}
	if found {
		return saved, nil
	}
	started := now.UnixMilli()
	if err := state.Set(ctx, ImportStartedAtKey, started); err != nil {
		return 0, fmt.Errorf("import-start speichern: %w", err)
	}
	return started, nil
}
