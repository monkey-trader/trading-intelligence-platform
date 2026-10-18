package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/clock"
)

// CurrentSchemaVersion ist die neueste bekannte Schema-Version.
const CurrentSchemaVersion = 1

const (
	backupFileMode  = 0o600
	backupTimeStamp = "20060102T150405Z"
)

// SchemaTooNewError meldet: Die DB hat eine höhere Version als dieses Programm kennt.
type SchemaTooNewError struct{ Found, Supported int }

func (err *SchemaTooNewError) Error() string {
	return fmt.Sprintf("datenbank hat Schema-Version %d, dieses Programm kennt nur bis %d", err.Found, err.Supported)
}

// BackupError meldet: Die Sicherung vor der Migration ist fehlgeschlagen; es wurde nicht migriert.
type BackupError struct {
	Path string
	Err  error
}

func (err *BackupError) Error() string {
	return fmt.Sprintf("sicherung %s fehlgeschlagen: %v", err.Path, err.Err)
}
func (err *BackupError) Unwrap() error { return err.Err }

// Open öffnet die Datenbank und migriert sie mit der Systemuhr.
func Open(path string) (*sql.DB, error) {
	return OpenWithClock(path, clock.SystemClock{})
}

// OpenWithClock wie Open, die Uhr liefert den Zeitstempel der Sicherung
// (Format 20060102T150405Z, UTC).
func OpenWithClock(path string, clock interfaces.Clock) (*sql.DB, error) {
	if path != memoryPath {
		if err := backupIfMigrationNeeded(path, clock); err != nil {
			return nil, err
		}
	}
	db, err := openConnection(path)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// backupIfMigrationNeeded sichert eine bestehende, nicht leere und ältere DB vor der Migration.
// Die Prüfung läuft ohne Pragmas, damit die Datei bis zur Kopie unverändert bleibt.
func backupIfMigrationNeeded(path string, clock interfaces.Clock) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && info.Size() == 0) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("sqlite-Datei prüfen: %w", err)
	}
	probe, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("sqlite öffnen: %w", err)
	}
	defer func() { _ = probe.Close() }()
	version, err := readVersion(probe)
	if err != nil {
		return err
	}
	if version > CurrentSchemaVersion {
		return &SchemaTooNewError{Found: version, Supported: CurrentSchemaVersion}
	}
	if version == CurrentSchemaVersion {
		return nil
	}
	// Im WAL-Modus stehen committete Daten evtl. nur in <db>-wal: vor der Kopie in die Hauptdatei übernehmen.
	if _, err := probe.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return &BackupError{Path: path, Err: fmt.Errorf("wal-checkpoint: %w", err)}
	}
	backupPath := fmt.Sprintf("%s.bak-v%d-%s", path, version, clock.Now().UTC().Format(backupTimeStamp))
	if err := copyExclusive(path, backupPath); err != nil {
		return &BackupError{Path: backupPath, Err: err}
	}
	return nil
}

// copyExclusive kopiert source nach target (0600) und überschreibt nie eine vorhandene Datei.
func copyExclusive(source, target string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, backupFileMode)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
	}()
	_, err = io.Copy(output, input)
	return err
}

// rowQuerier ist die gemeinsame Teilmenge von *sql.DB und *sql.Conn für Einzelzeilen-Abfragen.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// readVersion liefert die höchste Version; ohne Tabelle schema_version (TS-DB, leer) ist es 0.
func readVersion(querier rowQuerier) (int, error) {
	ctx := context.Background()
	var tableName string
	err := querier.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='schema_version'`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("schema_version prüfen: %w", err)
	}
	var version sql.NullInt64
	if err := querier.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("schema_version lesen: %w", err)
	}
	return int(version.Int64), nil
}

// migrate legt das Schema idempotent an und setzt schema_version auf die aktuelle Version.
// Die Version wird innerhalb von BEGIN IMMEDIATE gelesen, damit zwei Prozesse nicht gleichzeitig migrieren.
func migrate(db *sql.DB) (err error) {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("sqlite einrichten: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("sqlite einrichten: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, `ROLLBACK`)
		}
	}()
	version, err := readVersion(conn)
	if err != nil {
		return err
	}
	if version > CurrentSchemaVersion {
		return &SchemaTooNewError{Found: version, Supported: CurrentSchemaVersion}
	}
	statements := append([]string{}, schemaStatements...)
	statements = append(statements, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	for _, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("sqlite einrichten: %w", err)
		}
	}
	if version < CurrentSchemaVersion {
		if _, err := conn.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
			return fmt.Errorf("schema_version setzen: %w", err)
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO schema_version (version) VALUES (?)`, CurrentSchemaVersion); err != nil {
			return fmt.Errorf("schema_version setzen: %w", err)
		}
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("sqlite einrichten: %w", err)
	}
	committed = true
	return nil
}
