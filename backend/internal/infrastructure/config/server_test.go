package config_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/config"
)

var defaultAllowedHosts = []string{"127.0.0.1:3000", "localhost:3000", "[::1]:3000"}

// environmentFrom baut eine lookupEnv-Funktion wie os.Getenv: Nicht gesetzte Variablen ergeben "".
func environmentFrom(variables map[string]string) func(string) string {
	return func(name string) string { return variables[name] }
}

func withExtra(base []string, extra ...string) []string {
	combined := append([]string{}, base...)
	return append(combined, extra...)
}

func TestReadServerConfig_Erfolg(t *testing.T) {
	testCases := []struct {
		name        string
		environment map[string]string
		expected    config.ServerConfig
	}{
		{
			name:        "nutzt 127.0.0.1:3000, wenn nichts gesetzt ist",
			environment: map[string]string{},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name:        "behandelt leere Werte wie nicht gesetzt",
			environment: map[string]string{"HOST": "", "PORT": ""},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name:        "behandelt reine Leerzeichen wie nicht gesetzt",
			environment: map[string]string{"HOST": "   ", "PORT": "  "},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name:        "trimmt HOST und PORT",
			environment: map[string]string{"HOST": "  localhost ", "PORT": " 8080 "},
			expected: config.ServerConfig{
				Host: "localhost", Port: 8080,
				AllowedHosts: []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"},
			},
		},
		{
			name:        "übernimmt gültige Loopback-Werte aus der Umgebung",
			environment: map[string]string{"HOST": "localhost", "PORT": "8080"},
			expected: config.ServerConfig{
				Host: "localhost", Port: 8080,
				AllowedHosts: []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"},
			},
		},
		{
			name:        "erlaubt für PORT=8080 die Loopback-Hosts mit Port 8080",
			environment: map[string]string{"PORT": "8080"},
			expected: config.ServerConfig{
				Host: "127.0.0.1", Port: 8080,
				AllowedHosts: []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"},
			},
		},
		{
			name:        "Grenzwerte PORT=1",
			environment: map[string]string{"PORT": "1"},
			expected: config.ServerConfig{
				Host: "127.0.0.1", Port: 1,
				AllowedHosts: []string{"127.0.0.1:1", "localhost:1", "[::1]:1"},
			},
		},
		{
			name:        "Grenzwerte PORT=65535",
			environment: map[string]string{"PORT": "65535"},
			expected: config.ServerConfig{
				Host: "127.0.0.1", Port: 65535,
				AllowedHosts: []string{"127.0.0.1:65535", "localhost:65535", "[::1]:65535"},
			},
		},
		{
			name:        "ergänzt den gebundenen Loopback-Host 127.0.0.2",
			environment: map[string]string{"HOST": "127.0.0.2", "PORT": "3000"},
			expected: config.ServerConfig{
				Host: "127.0.0.2", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "127.0.0.2:3000"),
			},
		},
		{
			name:        "nimmt den gebundenen Host 127.0.0.1 nicht doppelt auf",
			environment: map[string]string{"HOST": "127.0.0.1", "PORT": "3000"},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name:        "schreibt den gebundenen IPv6-Loopback-Host mit Klammern und nicht doppelt",
			environment: map[string]string{"HOST": "::1", "PORT": "3000"},
			expected:    config.ServerConfig{Host: "::1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name: "erlaubt einen Nicht-Loopback-Host mit ALLOW_REMOTE=true",
			environment: map[string]string{
				"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": "journal.example:3000",
			},
			expected: config.ServerConfig{
				Host: "0.0.0.0", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "journal.example:3000"),
			},
		},
		{
			name: "ergänzt bei ALLOW_REMOTE=true die Einträge aus ALLOWED_HOSTS (getrimmt, kommagetrennt)",
			environment: map[string]string{
				"HOST": "0.0.0.0", "ALLOW_REMOTE": "true",
				"ALLOWED_HOSTS": " journal.example:3000 ,192.168.1.10:3000 ",
			},
			expected: config.ServerConfig{
				Host: "0.0.0.0", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "journal.example:3000", "192.168.1.10:3000"),
			},
		},
		{
			name: "ignoriert leere Einträge in ALLOWED_HOSTS",
			environment: map[string]string{
				"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": "journal.example:3000, ,",
			},
			expected: config.ServerConfig{
				Host: "0.0.0.0", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "journal.example:3000"),
			},
		},
		{
			name: "akzeptiert IPv6 mit Klammern in ALLOWED_HOSTS",
			environment: map[string]string{
				"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": "[fe80::1]:3000",
			},
			expected: config.ServerConfig{
				Host: "0.0.0.0", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "[fe80::1]:3000"),
			},
		},
		{
			name: "ALLOW_REMOTE=true mit Loopback-Host: gebundener Host plus ALLOWED_HOSTS",
			environment: map[string]string{
				"HOST": "127.0.0.2", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": "journal.example:3000",
			},
			expected: config.ServerConfig{
				Host: "127.0.0.2", Port: 3000,
				AllowedHosts: withExtra(defaultAllowedHosts, "127.0.0.2:3000", "journal.example:3000"),
			},
		},
		{
			name:        "ignoriert ALLOWED_HOSTS ohne ALLOW_REMOTE=true",
			environment: map[string]string{"ALLOWED_HOSTS": "evil.example:3000"},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
		{
			name:        "ignoriert ungültiges ALLOWED_HOSTS ohne ALLOW_REMOTE=true",
			environment: map[string]string{"ALLOWED_HOSTS": "kein host"},
			expected:    config.ServerConfig{Host: "127.0.0.1", Port: 3000, AllowedHosts: defaultAllowedHosts},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			serverConfig, err := config.ReadServerConfig(environmentFrom(testCase.environment))
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if !reflect.DeepEqual(serverConfig, testCase.expected) {
				t.Fatalf("Konfiguration = %#v, erwartet %#v", serverConfig, testCase.expected)
			}
		})
	}
}

func TestReadServerConfig_ErlaubtLoopbackHosts(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "127.0.0.2", "127.255.255.254", "localhost", "::1"} {
		t.Run(host, func(t *testing.T) {
			serverConfig, err := config.ReadServerConfig(environmentFrom(map[string]string{"HOST": host}))
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if serverConfig.Host != host {
				t.Fatalf("Host = %q, erwartet %q", serverConfig.Host, host)
			}
		})
	}
}

func TestReadServerConfig_LehntNichtLoopbackHostOhneAllowRemoteAb(t *testing.T) {
	testCases := []struct {
		name        string
		environment map[string]string
		host        string
	}{
		{name: "0.0.0.0", environment: map[string]string{"HOST": "0.0.0.0"}, host: "0.0.0.0"},
		{name: "192.168.1.10", environment: map[string]string{"HOST": "192.168.1.10"}, host: "192.168.1.10"},
		{name: "::", environment: map[string]string{"HOST": "::"}, host: "::"},
		{name: "example.org", environment: map[string]string{"HOST": "example.org"}, host: "example.org"},
		{name: "LOCALHOST (wie TS: nur exakt localhost)", environment: map[string]string{"HOST": "LOCALHOST"}, host: "LOCALHOST"},
		{name: "127.0.0.1.example.org", environment: map[string]string{"HOST": "127.0.0.1.example.org"}, host: "127.0.0.1.example.org"},
		{name: "127.0.0", environment: map[string]string{"HOST": "127.0.0"}, host: "127.0.0"},
		{name: "getrimmter Host im Fehler", environment: map[string]string{"HOST": " example.org "}, host: "example.org"},
		{
			name:        "ALLOW_REMOTE=yes reicht nicht",
			environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "yes", "ALLOWED_HOSTS": "journal.example:3000"},
			host:        "0.0.0.0",
		},
		{
			name:        "ALLOW_REMOTE=TRUE reicht nicht",
			environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "TRUE", "ALLOWED_HOSTS": "journal.example:3000"},
			host:        "0.0.0.0",
		},
		{
			name:        "Host wird vor dem Port geprüft",
			environment: map[string]string{"HOST": "0.0.0.0", "PORT": "abc"},
			host:        "0.0.0.0",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := config.ReadServerConfig(environmentFrom(testCase.environment))
			var remoteHostError *config.RemoteHostNotAllowedError
			if !errors.As(err, &remoteHostError) {
				t.Fatalf("Fehler = %v (%T), erwartet *RemoteHostNotAllowedError", err, err)
			}
			if remoteHostError.Host != testCase.host {
				t.Fatalf("Host im Fehler = %q, erwartet %q", remoteHostError.Host, testCase.host)
			}
			if !strings.Contains(err.Error(), "ALLOW_REMOTE") || !strings.Contains(err.Error(), testCase.host) {
				t.Fatalf("Fehlermeldung %q nennt ALLOW_REMOTE oder den Host nicht", err.Error())
			}
		})
	}
}

func TestReadServerConfig_LehntUngueltigenPortAb(t *testing.T) {
	// "1e3" und "0x10" akzeptiert TS über Number(), Go bewusst nicht (dokumentierte Abweichung).
	for _, rawPort := range []string{"abc", "3000.5", "0", "65536", "-1", "1e3", "0x10", "99999999999999999999"} {
		t.Run(rawPort, func(t *testing.T) {
			_, err := config.ReadServerConfig(environmentFrom(map[string]string{"PORT": rawPort}))
			var invalidPortError *config.InvalidPortError
			if !errors.As(err, &invalidPortError) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidPortError", err, err)
			}
			if invalidPortError.Value != rawPort {
				t.Fatalf("Value im Fehler = %q, erwartet %q", invalidPortError.Value, rawPort)
			}
			expectedMessage := `Ungültiger PORT: "` + rawPort + `"`
			if err.Error() != expectedMessage {
				t.Fatalf("Fehlermeldung = %q, erwartet %q", err.Error(), expectedMessage)
			}
		})
	}
}

func TestReadServerConfig_VerlangtAllowedHostsBeiAllowRemote(t *testing.T) {
	testCases := []struct {
		name        string
		environment map[string]string
	}{
		{name: "nicht gesetzt", environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "true"}},
		{name: "leer", environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": ""}},
		{name: "nur Leerzeichen", environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": "   "}},
		{name: "nur Kommas", environment: map[string]string{"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": " , "}},
		{name: "auch mit Loopback-Host", environment: map[string]string{"ALLOW_REMOTE": "true"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := config.ReadServerConfig(environmentFrom(testCase.environment))
			var missingAllowedHostsError *config.MissingAllowedHostsError
			if !errors.As(err, &missingAllowedHostsError) {
				t.Fatalf("Fehler = %v (%T), erwartet *MissingAllowedHostsError", err, err)
			}
			if !strings.Contains(err.Error(), "ALLOWED_HOSTS") {
				t.Fatalf("Fehlermeldung %q nennt ALLOWED_HOSTS nicht", err.Error())
			}
		})
	}
}

func TestReadServerConfig_LehntUngueltigenAllowedHostsEintragAb(t *testing.T) {
	testCases := []struct {
		allowedHosts  string
		expectedEntry string
	}{
		{allowedHosts: "meinrechner.local", expectedEntry: "meinrechner.local"},
		{allowedHosts: "meinrechner.local:", expectedEntry: "meinrechner.local:"},
		{allowedHosts: "meinrechner.local:abc", expectedEntry: "meinrechner.local:abc"},
		{allowedHosts: "a b:3000", expectedEntry: "a b:3000"},
		{allowedHosts: "meinrechner.local:123456", expectedEntry: "meinrechner.local:123456"},
		{allowedHosts: "fe80::1:3000", expectedEntry: "fe80::1:3000"},
		{allowedHosts: "journal.example:3000, kaputt ", expectedEntry: "kaputt"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.allowedHosts, func(t *testing.T) {
			_, err := config.ReadServerConfig(environmentFrom(map[string]string{
				"HOST": "0.0.0.0", "ALLOW_REMOTE": "true", "ALLOWED_HOSTS": testCase.allowedHosts,
			}))
			var invalidAllowedHostError *config.InvalidAllowedHostError
			if !errors.As(err, &invalidAllowedHostError) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidAllowedHostError", err, err)
			}
			if invalidAllowedHostError.Entry != testCase.expectedEntry {
				t.Fatalf("Entry im Fehler = %q, erwartet %q", invalidAllowedHostError.Entry, testCase.expectedEntry)
			}
			if !strings.Contains(err.Error(), "ALLOWED_HOSTS") || !strings.Contains(err.Error(), testCase.expectedEntry) {
				t.Fatalf("Fehlermeldung %q nennt ALLOWED_HOSTS oder den Eintrag nicht", err.Error())
			}
		})
	}
}
