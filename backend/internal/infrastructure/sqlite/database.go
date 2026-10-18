// Package sqlite enthält die SQLite-Adapter der Repository-Ports.
package sqlite

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"

	_ "modernc.org/sqlite" // Treiber ohne cgo
)

const memoryPath = ":memory:"

// Open öffnet die Datenbank, aktiviert Fremdschlüssel und busy_timeout (sowie WAL bei Dateien)
// und legt das Schema an. Die Pragmas stehen im DSN, damit sie für jede Pool-Verbindung gelten.
// Die Datei entsteht mit Rechten 0600; das Verzeichnis muss existieren.
func openConnection(path string) (*sql.DB, error) {
	pragmas := []string{"foreign_keys(1)", "busy_timeout(5000)"}
	if path != memoryPath {
		pragmas = append(pragmas, "journal_mode(WAL)")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("sqlite-Datei anlegen: %w", err)
		}
		_ = file.Close()
	}
	query := url.Values{"_pragma": pragmas}
	dsn := (&url.URL{Scheme: "file", Opaque: url.PathEscape(path), RawQuery: query.Encode()}).String()
	if path == memoryPath {
		dsn = "file::memory:?" + query.Encode()
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite öffnen: %w", err)
	}
	if path == memoryPath {
		// Jede Verbindung hätte sonst ihre eigene In-Memory-DB.
		db.SetMaxOpenConns(1)
	}
	return db, nil
}
