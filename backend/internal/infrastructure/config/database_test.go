package config_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/config"
)

func TestReadDatabasePath(t *testing.T) {
	appSupport := filepath.Join("home", "Library", "Application Support")
	configDir := func() (string, error) { return appSupport, nil }
	failingDir := func() (string, error) { return "", errors.New("kein Home") }

	testCases := []struct {
		name      string
		variables map[string]string
		dir       func() (string, error)
		want      string
		wantErr   bool
	}{
		{name: "Default unter Application Support", variables: map[string]string{}, dir: configDir,
			want: filepath.Join(appSupport, "tip", "journal.db")},
		{name: "JOURNAL_DB überschreibt", variables: map[string]string{"JOURNAL_DB": "/tmp/x/j.db"}, dir: configDir,
			want: "/tmp/x/j.db"},
		{name: "JOURNAL_DB wird getrimmt", variables: map[string]string{"JOURNAL_DB": "  /tmp/x/j.db \n"}, dir: configDir,
			want: "/tmp/x/j.db"},
		{name: "nur Leerzeichen gilt als nicht gesetzt", variables: map[string]string{"JOURNAL_DB": "   "}, dir: configDir,
			want: filepath.Join(appSupport, "tip", "journal.db")},
		{name: "JOURNAL_DB gesetzt: Config-Dir wird nicht gebraucht", variables: map[string]string{"JOURNAL_DB": "/a/b.db"}, dir: failingDir,
			want: "/a/b.db"},
		{name: "ohne JOURNAL_DB und ohne Config-Dir: Fehler", variables: map[string]string{}, dir: failingDir, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := config.ReadDatabasePath(environmentFrom(testCase.variables), testCase.dir)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if got != testCase.want {
				t.Errorf("Pfad = %q, erwartet %q", got, testCase.want)
			}
		})
	}
}
