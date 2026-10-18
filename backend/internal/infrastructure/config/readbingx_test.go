package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/config"
)

// Fälle portiert aus backend/src/infrastructure/config/bingxConfig.test.ts, ergänzt um die
// BX_BASE-Prüfung (bewusste Abweichung zu TS, siehe Zuschnitt Phase 2a).

const defaultBingXBaseURL = "https://open-api.bingx.com"

func TestReadBingXConfig_Konfiguriert(t *testing.T) {
	testCases := []struct {
		name     string
		vars     map[string]string
		expected config.BingXConfig
	}{
		{
			name:     "Default-Basis",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret"},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: defaultBingXBaseURL},
		},
		{
			name:     "Werte werden getrimmt",
			vars:     map[string]string{"BX_KEY": "  key  ", "BX_SECRET": "\tsecret\n", "BX_BASE": "  https://bingx.example.test  "},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: "https://bingx.example.test"},
		},
		{
			name:     "leeres BX_BASE ergibt Default",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": "   "},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: defaultBingXBaseURL},
		},
		{
			name:     "https mit beliebigem Host",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": "https://open-api-vst.bingx.com"},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: "https://open-api-vst.bingx.com"},
		},
		{
			name:     "http mit 127.0.0.1 und Pfad-Präfix",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": "http://127.0.0.1:4010/ok"},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: "http://127.0.0.1:4010/ok"},
		},
		{
			name:     "http mit localhost",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": "http://localhost:4010"},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: "http://localhost:4010"},
		},
		{
			name:     "http mit ::1",
			vars:     map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": "http://[::1]:4010"},
			expected: config.BingXConfig{APIKey: "key", APISecret: "secret", BaseURL: "http://[::1]:4010"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bingXConfig, configured, err := config.ReadBingXConfig(environmentFrom(testCase.vars))
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !configured {
				t.Fatal("configured = false, erwartet true")
			}
			if bingXConfig != testCase.expected {
				t.Errorf("Config = %+v, erwartet %+v", redact(bingXConfig), redact(testCase.expected))
			}
		})
	}
}

func TestReadBingXConfig_NichtKonfiguriert(t *testing.T) {
	testCases := []struct {
		name string
		vars map[string]string
	}{
		{"beide fehlen", map[string]string{}},
		{"nur BX_KEY", map[string]string{"BX_KEY": "key"}},
		{"nur BX_SECRET", map[string]string{"BX_SECRET": "secret"}},
		{"BX_KEY nur Leerzeichen", map[string]string{"BX_KEY": "   ", "BX_SECRET": "secret"}},
		{"BX_SECRET nur Whitespace", map[string]string{"BX_KEY": "key", "BX_SECRET": " \t\r\n "}},
		{"nur gültiges BX_BASE", map[string]string{"BX_BASE": "https://bingx.example.test"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bingXConfig, configured, err := config.ReadBingXConfig(environmentFrom(testCase.vars))
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if configured {
				t.Error("configured = true, erwartet false")
			}
			if bingXConfig != (config.BingXConfig{}) {
				t.Errorf("Config = %+v, erwartet leer", redact(bingXConfig))
			}
		})
	}
}

func TestReadBingXConfig_UngueltigesBXBase(t *testing.T) {
	invalidBaseURLs := []string{
		"http://open-api.bingx.com",
		"http://10.0.0.1:4010",
		"http://127.0.0.2:4010",
		"http://localhost.evil.example",
		"http://127.0.0.1.nip.io",
		"ftp://127.0.0.1",
		"open-api.bingx.com",
		"https://",
		"http://",
		"//open-api.bingx.com",
		"::kaputt",
		"https://open-api.bingx.com\nX: y",
	}

	for _, baseURL := range invalidBaseURLs {
		t.Run(baseURL, func(t *testing.T) {
			vars := map[string]string{"BX_KEY": "geheim-key-4711", "BX_SECRET": "geheim-secret-0815", "BX_BASE": baseURL}

			bingXConfig, _, err := config.ReadBingXConfig(environmentFrom(vars))

			var invalidBase *config.InvalidBingXBaseURLError
			if !errors.As(err, &invalidBase) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidBingXBaseURLError", err, err)
			}
			if text := err.Error(); strings.Contains(text, "geheim-key-4711") || strings.Contains(text, "geheim-secret-0815") {
				t.Errorf("Fehlertext enthält Secrets: %q", text)
			}
			if bingXConfig.APIKey != "" || bingXConfig.APISecret != "" {
				t.Errorf("bei Fehler dürfen keine Zugangsdaten zurückkommen")
			}
		})
	}
}

func TestReadBingXConfig_UngueltigesBXBaseOhneKeys(t *testing.T) {
	// BX_BASE wird immer geprüft, auch wenn BX_KEY/BX_SECRET fehlen.
	for _, vars := range []map[string]string{
		{"BX_BASE": "http://open-api.bingx.com"},
		{"BX_KEY": "key", "BX_BASE": "ftp://127.0.0.1"},
	} {
		t.Run(vars["BX_BASE"], func(t *testing.T) {
			bingXConfig, _, err := config.ReadBingXConfig(environmentFrom(vars))

			var invalidBase *config.InvalidBingXBaseURLError
			if !errors.As(err, &invalidBase) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidBingXBaseURLError", err, err)
			}
			if bingXConfig != (config.BingXConfig{}) {
				t.Errorf("Config = %+v, erwartet leer bei Fehler", redact(bingXConfig))
			}
		})
	}
}

// redact verhindert, dass Testausgaben echte Secrets zeigen könnten.
func redact(bingXConfig config.BingXConfig) config.BingXConfig {
	if bingXConfig.APISecret != "" {
		bingXConfig.APISecret = "***"
	}
	return bingXConfig
}

func TestReadBingXConfig_NormalisiertAbschliessendeSchraegstriche(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"Host mit einem Slash", "https://open-api.bingx.com/", defaultBingXBaseURL},
		{"Host mit mehreren Slashes", "https://open-api.bingx.com///", defaultBingXBaseURL},
		{"Pfad-Präfix mit Slash", "https://host.example.test/prefix/", "https://host.example.test/prefix"},
		{"Pfad-Präfix ohne Slash bleibt", "https://host.example.test/prefix", "https://host.example.test/prefix"},
		{"mehrstufiges Präfix", "https://host.example.test/a/b//", "https://host.example.test/a/b"},
		{"Loopback mit Slash", "http://127.0.0.1:4010/", "http://127.0.0.1:4010"},
		{"getrimmt und ohne Slash", "  https://open-api.bingx.com/  ", defaultBingXBaseURL},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			vars := map[string]string{"BX_KEY": "key", "BX_SECRET": "secret", "BX_BASE": testCase.baseURL}
			bingXConfig, configured, err := config.ReadBingXConfig(func(name string) string { return vars[name] })
			if err != nil || !configured {
				t.Fatalf("configured = %v, err = %v, erwartet konfiguriert ohne Fehler", configured, err)
			}
			if bingXConfig.BaseURL != testCase.expected {
				t.Errorf("BaseURL = %q, erwartet %q", bingXConfig.BaseURL, testCase.expected)
			}
		})
	}
}
