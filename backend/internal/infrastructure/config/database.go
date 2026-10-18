package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReadDatabasePath liefert JOURNAL_DB (getrimmt), sonst <userConfigDir>/tip/journal.db.
// Schlägt userConfigDir fehl und ist JOURNAL_DB leer, kommt dessen Fehler zurück.
func ReadDatabasePath(lookupEnv func(string) string, userConfigDir func() (string, error)) (string, error) {
	if explicitPath := strings.TrimSpace(lookupEnv("JOURNAL_DB")); explicitPath != "" {
		return explicitPath, nil
	}
	configDir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("config-verzeichnis ermitteln: %w", err)
	}
	return filepath.Join(configDir, "tip", "journal.db"), nil
}
