package config

import (
	"net/url"
	"strings"
)

// defaultBingXBaseURL ist die echte BingX-API.
const defaultBingXBaseURL = "https://open-api.bingx.com"

// loopbackHosts sind die einzigen Hosts, die BX_BASE mit http nutzen darf (z.B. ein lokaler Fake).
var loopbackHosts = map[string]bool{"127.0.0.1": true, "localhost": true, "::1": true}

// bingXRequiredVariables sind die Pflichtvariablen für den BingX-Zugang in fester Reihenfolge.
var bingXRequiredVariables = []string{"BX_KEY", "BX_SECRET"}

// MissingBingXVariables liefert die Namen der fehlenden BingX-Pflichtvariablen in der
// Reihenfolge BX_KEY, BX_SECRET. Leere Werte und reine Leerzeichen gelten als fehlend
// (wie missingBingXVariables in backend/src/infrastructure/config/bingxConfig.ts).
// Gibt nie Werte zurück, nur Variablennamen.
func MissingBingXVariables(lookupEnv func(string) string) []string {
	var missingVariables []string
	for _, variableName := range bingXRequiredVariables {
		if strings.TrimSpace(lookupEnv(variableName)) == "" {
			missingVariables = append(missingVariables, variableName)
		}
	}
	return missingVariables
}

// BingXConfig sind die BingX-Zugangsdaten. Niemals loggen.
type BingXConfig struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

// InvalidBingXBaseURLError meldet: BX_BASE ist keine erlaubte Basis-URL (https, oder http nur mit Loopback-Host).
// Der Wert selbst steht nicht in der Meldung, weil er Zugangsdaten (user:pass@) enthalten könnte.
type InvalidBingXBaseURLError struct {
	BaseURL string
	Reason  string
}

func (err *InvalidBingXBaseURLError) Error() string {
	return "BX_BASE ist ungültig: " + err.Reason
}

// ReadBingXConfig liest BX_KEY, BX_SECRET und BX_BASE (getrimmt). Fehlen Key oder Secret,
// kommt (BingXConfig{}, false, nil). Ein ungültiges BX_BASE ergibt *InvalidBingXBaseURLError.
// Abweichung zu TS: TS prüft BX_BASE nicht.
func ReadBingXConfig(lookupEnv func(string) string) (BingXConfig, bool, error) {
	baseURL := strings.TrimSpace(lookupEnv("BX_BASE"))
	if baseURL == "" {
		baseURL = defaultBingXBaseURL
	}
	// Abschließende Schrägstriche weg, weil der Client den Pfad direkt anhängt (Pfad-Präfix bleibt).
	baseURL = strings.TrimRight(baseURL, "/")
	// Immer prüfen, auch ohne Keys: Ein Tippfehler soll beim Start auffallen.
	if err := validateBaseURL(baseURL); err != nil {
		return BingXConfig{}, false, err
	}

	apiKey := strings.TrimSpace(lookupEnv("BX_KEY"))
	apiSecret := strings.TrimSpace(lookupEnv("BX_SECRET"))
	if apiKey == "" || apiSecret == "" {
		return BingXConfig{}, false, nil
	}
	return BingXConfig{APIKey: apiKey, APISecret: apiSecret, BaseURL: baseURL}, true, nil
}

// validateBaseURL erlaubt https mit beliebigem Host und http nur mit Loopback-Host.
// Zugangsdaten, Query und Fragment sind verboten, weil der Client Pfad und Query anhängt.
func validateBaseURL(baseURL string) error {
	invalid := func(reason string) error { return &InvalidBingXBaseURLError{BaseURL: baseURL, Reason: reason} }

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return invalid("keine gültige URL")
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return invalid("Host fehlt")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return invalid("Zugangsdaten, Query und Fragment sind nicht erlaubt")
	}
	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if loopbackHosts[parsed.Hostname()] {
			return nil
		}
		return invalid("http nur mit 127.0.0.1, localhost oder ::1, sonst https")
	default:
		return invalid("nur https (oder http mit Loopback-Host) erlaubt")
	}
}
