package interfaces

import "context"

// ImportStateRepository speichert den Import-Fortschritt als Schlüssel/Wert (Epoch-ms).
type ImportStateRepository interface {
	// Get liefert den Wert; found=false, wenn der Schlüssel fehlt.
	Get(ctx context.Context, key string) (value int64, found bool, err error)
	// Set legt den Schlüssel an oder überschreibt ihn.
	Set(ctx context.Context, key string, value int64) error
	// Delete entfernt den Schlüssel (fehlender Schlüssel ist kein Fehler).
	Delete(ctx context.Context, key string) error
	// ListKeys liefert alle Schlüssel mit Präfix, sortiert. `_` und `%` sind keine Platzhalter.
	ListKeys(ctx context.Context, prefix string) ([]string, error)
}
