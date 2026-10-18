package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openMemory(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	db.SetMaxOpenConns(1) // bei :memory: sonst eine DB pro Verbindung
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpen_LegtSchemaAnUndIstIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tip.db")
	for run := 1; run <= 2; run++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open Lauf %d: %v", run, err)
		}
		for _, table := range []string{"positions", "notes", "import_state"} {
			var name string
			err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
			if err != nil {
				t.Errorf("Lauf %d: Tabelle %s fehlt: %v", run, table, err)
			}
		}
		var indexName string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='positions_symbol_close_time'`).Scan(&indexName); err != nil {
			t.Errorf("Lauf %d: Index fehlt: %v", run, err)
		}
		_ = db.Close()
	}
}

func TestOpen_ForeignKeysAktiv(t *testing.T) {
	db := openMemory(t)
	var enabled int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
		t.Fatalf("foreign_keys = %d, err %v", enabled, err)
	}
	_, err := db.Exec(`INSERT INTO notes (position_id) VALUES ('gibt-es-nicht')`)
	if err == nil {
		t.Fatal("notes ohne Position muss am Fremdschlüssel scheitern")
	}
}

func TestOpen_DateiNutztWAL(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "wal.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, err %v", mode, err)
	}
}

func TestOpen_UngueltigerPfadLiefertFehler(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "fehlt", "x", "tip.db")); err == nil {
		t.Fatal("Fehler erwartet")
	}
}

func TestOpen_PragmasGeltenFuerJedeVerbindung(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	first, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := db.Conn(ctx) // erzwingt eine zweite Pool-Verbindung
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	for name, conn := range map[string]*sql.Conn{"erste": first, "zweite": second} {
		var enabled int
		if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
			t.Errorf("%s Verbindung: foreign_keys = %d, err %v", name, enabled, err)
		}
	}
}

func TestOpen_DateiHatNurBesitzerRechte(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rechte.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("Rechte %o, erwartet keine für Gruppe/Andere", perm)
	}
}
