package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ImportStateRepository speichert den Import-Fortschritt in SQLite.
type ImportStateRepository struct {
	db *sql.DB
}

// NewImportStateRepository erzeugt das Repository.
func NewImportStateRepository(db *sql.DB) *ImportStateRepository {
	return &ImportStateRepository{db: db}
}

// Get liefert den Wert; found=false, wenn der Schlüssel fehlt.
func (repo *ImportStateRepository) Get(ctx context.Context, key string) (int64, bool, error) {
	var value int64
	err := repo.db.QueryRowContext(ctx, `SELECT value FROM import_state WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("import_state %q lesen: %w", key, err)
	}
	return value, true, nil
}

// Set legt den Schlüssel an oder überschreibt ihn.
func (repo *ImportStateRepository) Set(ctx context.Context, key string, value int64) error {
	_, err := repo.db.ExecContext(ctx,
		`INSERT INTO import_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	if err != nil {
		return fmt.Errorf("import_state %q schreiben: %w", key, err)
	}
	return nil
}

// Delete entfernt den Schlüssel; ein fehlender Schlüssel ist kein Fehler.
func (repo *ImportStateRepository) Delete(ctx context.Context, key string) error {
	if _, err := repo.db.ExecContext(ctx, `DELETE FROM import_state WHERE key = ?`, key); err != nil {
		return fmt.Errorf("import_state %q löschen: %w", key, err)
	}
	return nil
}

// ListKeys liefert alle Schlüssel mit Präfix, sortiert (substr statt LIKE, daher ohne Platzhalter).
func (repo *ImportStateRepository) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	rows, err := repo.db.QueryContext(ctx,
		`SELECT key FROM import_state WHERE substr(key, 1, length(?1)) = ?1 ORDER BY key`, prefix)
	if err != nil {
		return nil, fmt.Errorf("import_state auflisten: %w", err)
	}
	defer func() { _ = rows.Close() }()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("import_state Schlüssel lesen: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("import_state auflisten: %w", err)
	}
	return keys, nil
}
