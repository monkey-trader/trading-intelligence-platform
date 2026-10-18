package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type migrationClock struct{ now time.Time }

func (clock migrationClock) Now() time.Time { return clock.now }

var backupTime = time.Date(2026, 10, 18, 12, 30, 45, 0, time.UTC)

const backupStamp = "20261018T123045Z"

// createLegacyDB baut eine TS-DB ohne schema_version mit Daten.
func createLegacyDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	statements := append([]string{}, schemaStatements...)
	statements = append(statements,
		`CREATE TABLE IF NOT EXISTS balance (ts INTEGER, equity TEXT)`,
		`INSERT INTO positions VALUES ('1234567890123456789','BTC-USDT','LONG',1,1000,2000,'1','2','3','4','5','6','7','8',10.5,1,'{}',3000)`,
		`INSERT INTO income VALUES ('t1','BTC-USDT','REALIZED_PNL','1.5',1500,'USDT','tr1','{}',3000)`,
		`INSERT INTO notes VALUES ('1234567890123456789','Breakout',4,'FOMO','Notiz',5000)`,
		`INSERT INTO import_state VALUES ('income_scan_until', 42)`,
		`INSERT INTO balance VALUES (1, '100.5')`,
	)
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

func schemaVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatalf("schema_version lesen: %v", err)
	}
	return version
}

func count(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&rows); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return rows
}

func dumpNotes(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT position_id||'|'||IFNULL(setup,'')||'|'||IFNULL(rating,'')||'|'||IFNULL(mistakes,'')||'|'||IFNULL(notes,'')||'|'||IFNULL(updated_at,'') FROM notes ORDER BY position_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var result []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		result = append(result, line)
	}
	return result
}

func backups(t *testing.T, path string) []string {
	t.Helper()
	matches, err := filepath.Glob(path + ".bak-*")
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestOpenWithClock_HebtTSDatenbankOhneDatenverlustAufVersion1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	createLegacyDB(t, path)

	db, err := OpenWithClock(path, migrationClock{backupTime})
	if err != nil {
		t.Fatalf("OpenWithClock: %v", err)
	}
	defer func() { _ = db.Close() }()

	if got := schemaVersion(t, db); got != 1 {
		t.Errorf("Version = %d, erwartet 1", got)
	}
	for table, want := range map[string]int{"positions": 1, "income": 1, "notes": 1, "import_state": 1, "balance": 1} {
		if got := count(t, db, table); got != want {
			t.Errorf("%s: %d Zeilen, erwartet %d", table, got, want)
		}
	}
	var amount string
	if err := db.QueryRow(`SELECT amount FROM income WHERE transaction_id='t1'`).Scan(&amount); err != nil || amount != "1.5" {
		t.Errorf("income.amount = %q (%v)", amount, err)
	}
}

func TestOpenWithClock_IstIdempotentUndSichertNurEinmal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	createLegacyDB(t, path)

	for run := 1; run <= 3; run++ {
		db, err := OpenWithClock(path, migrationClock{backupTime.Add(time.Duration(run) * time.Hour)})
		if err != nil {
			t.Fatalf("Lauf %d: %v", run, err)
		}
		if got := schemaVersion(t, db); got != 1 {
			t.Errorf("Lauf %d: Version = %d", run, got)
		}
		if got := count(t, db, "schema_version"); got != 1 {
			t.Errorf("Lauf %d: schema_version hat %d Zeilen, erwartet 1", run, got)
		}
		if got := count(t, db, "positions"); got != 1 {
			t.Errorf("Lauf %d: positions = %d", run, got)
		}
		_ = db.Close()
	}
	if got := backups(t, path); len(got) != 1 {
		t.Errorf("Sicherungen = %v, erwartet genau eine", got)
	}
}

func TestOpenWithClock_LehntNeuereVersionAb(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	createLegacyDB(t, path)
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE schema_version (version INTEGER NOT NULL)`,
		`INSERT INTO schema_version VALUES (99)`,
	} {
		if _, err := raw.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	_ = raw.Close()

	db, err := OpenWithClock(path, migrationClock{backupTime})
	if db != nil {
		_ = db.Close()
		t.Fatal("DB trotz höherer Version geöffnet")
	}
	var tooNew *SchemaTooNewError
	if !errors.As(err, &tooNew) {
		t.Fatalf("err = %v, erwartet *SchemaTooNewError", err)
	}
	if tooNew.Found != 99 || tooNew.Supported != CurrentSchemaVersion {
		t.Errorf("Found/Supported = %d/%d, erwartet 99/%d", tooNew.Found, tooNew.Supported, CurrentSchemaVersion)
	}
	if got := backups(t, path); len(got) != 0 {
		t.Errorf("Sicherung trotz Ablehnung: %v", got)
	}
}

func TestOpenWithClock_Sicherung(t *testing.T) {
	t.Run("bestehende DB bekommt Sicherung mit Zeitstempel, Rechten 0600 und gleichem Inhalt", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal.db")
		createLegacyDB(t, path)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		db, err := OpenWithClock(path, migrationClock{backupTime})
		if err != nil {
			t.Fatal(err)
		}
		_ = db.Close()

		backupPath := path + ".bak-v0-" + backupStamp
		info, err := os.Stat(backupPath)
		if err != nil {
			t.Fatalf("Sicherung fehlt: %v (vorhanden: %v)", err, backups(t, path))
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("Rechte = %v, erwartet 0600", info.Mode().Perm())
		}
		saved, err := os.ReadFile(backupPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(saved) != string(original) {
			t.Error("Sicherung ist nicht der Stand vor der Migration")
		}
	})

	t.Run("frische DB: keine Sicherung", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal.db")
		db, err := OpenWithClock(path, migrationClock{backupTime})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		if got := schemaVersion(t, db); got != 1 {
			t.Errorf("Version = %d", got)
		}
		if got := backups(t, path); len(got) != 0 {
			t.Errorf("Sicherungen = %v, erwartet keine", got)
		}
	})

	t.Run("bereits aktuelle DB: keine neue Sicherung", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal.db")
		for run := 0; run < 2; run++ {
			db, err := OpenWithClock(path, migrationClock{backupTime})
			if err != nil {
				t.Fatal(err)
			}
			_ = db.Close()
		}
		if got := backups(t, path); len(got) != 0 {
			t.Errorf("Sicherungen = %v, erwartet keine", got)
		}
	})

	t.Run("Fehler bei der Sicherung verhindert die Migration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal.db")
		createLegacyDB(t, path)
		// Ziel der Sicherung ist schon als nicht beschreibbares Verzeichnis belegt.
		if err := os.Mkdir(path+".bak-v0-"+backupStamp, 0o700); err != nil {
			t.Fatal(err)
		}

		db, err := OpenWithClock(path, migrationClock{backupTime})
		if db != nil {
			_ = db.Close()
			t.Fatal("DB trotz Backup-Fehler geöffnet")
		}
		var backupErr *BackupError
		if !errors.As(err, &backupErr) {
			t.Fatalf("err = %v, erwartet *BackupError", err)
		}

		raw, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = raw.Close() }()
		var tableName string
		err = raw.QueryRow(`SELECT name FROM sqlite_master WHERE name='schema_version'`).Scan(&tableName)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("schema_version existiert trotz Backup-Fehler (err=%v)", err)
		}
	})
}

func TestOpenWithClock_NotesBleibenUnveraendert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	createLegacyDB(t, path)
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	before := dumpNotes(t, raw)
	_ = raw.Close()
	if len(before) != 1 {
		t.Fatalf("Fixture: %d Notizen", len(before))
	}

	db, err := OpenWithClock(path, migrationClock{backupTime})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if after := dumpNotes(t, db); !equalStrings(before, after) {
		t.Errorf("Migration änderte notes: %v -> %v", before, after)
	}

	// Import (Upsert) einer bekannten Position darf notes nicht berühren.
	_, err = db.Exec(`INSERT INTO positions VALUES ('1234567890123456789','BTC-USDT','LONG',1,1000,2000,'9','9','9','9','9','9','9','9',1,1,'{}',9000)
		ON CONFLICT(position_id) DO UPDATE SET net_profit=excluded.net_profit`)
	if err != nil {
		t.Fatal(err)
	}
	if after := dumpNotes(t, db); !equalStrings(before, after) {
		t.Errorf("Import änderte notes: %v -> %v", before, after)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
