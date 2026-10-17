package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/config"
)

func TestMissingBingXVariables(t *testing.T) {
	testCases := []struct {
		name            string
		variables       map[string]string
		expectedMissing []string
	}{
		{name: "beide fehlen", variables: map[string]string{}, expectedMissing: []string{"BX_KEY", "BX_SECRET"}},
		{name: "beide gesetzt", variables: map[string]string{"BX_KEY": "key", "BX_SECRET": "secret"}, expectedMissing: nil},
		{name: "nur BX_KEY gesetzt", variables: map[string]string{"BX_KEY": "key"}, expectedMissing: []string{"BX_SECRET"}},
		{name: "nur BX_SECRET gesetzt", variables: map[string]string{"BX_SECRET": "secret"}, expectedMissing: []string{"BX_KEY"}},
		{
			name:            "beide leer",
			variables:       map[string]string{"BX_KEY": "", "BX_SECRET": ""},
			expectedMissing: []string{"BX_KEY", "BX_SECRET"},
		},
		{
			name:            "nur Leerzeichen gilt als fehlend",
			variables:       map[string]string{"BX_KEY": "   ", "BX_SECRET": "secret"},
			expectedMissing: []string{"BX_KEY"},
		},
		{
			name:            "Tab und Zeilenumbruch gelten als fehlend",
			variables:       map[string]string{"BX_KEY": "key", "BX_SECRET": " \t\r\n "},
			expectedMissing: []string{"BX_SECRET"},
		},
		{
			name:            "Werte mit Leerzeichen außenrum gelten als gesetzt",
			variables:       map[string]string{"BX_KEY": "  key  ", "BX_SECRET": "\tsecret\n"},
			expectedMissing: nil,
		},
		{
			name:            "BX_BASE spielt keine Rolle",
			variables:       map[string]string{"BX_BASE": "https://example.invalid"},
			expectedMissing: []string{"BX_KEY", "BX_SECRET"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			missingVariables := config.MissingBingXVariables(environmentFrom(testCase.variables))

			if len(missingVariables) == 0 && len(testCase.expectedMissing) == 0 {
				return
			}
			if !slices.Equal(missingVariables, testCase.expectedMissing) {
				t.Errorf("MissingBingXVariables = %q, erwartet %q", missingVariables, testCase.expectedMissing)
			}
		})
	}
}

func TestMissingBingXVariables_GibtNieWerteZurueck(t *testing.T) {
	const secretValue = "geheim-4711"
	missingVariables := config.MissingBingXVariables(environmentFrom(map[string]string{"BX_KEY": secretValue}))

	for _, entry := range missingVariables {
		if strings.Contains(entry, secretValue) {
			t.Fatalf("Ergebnis enthält einen Wert: %q", missingVariables)
		}
	}
	if !slices.Equal(missingVariables, []string{"BX_SECRET"}) {
		t.Errorf("MissingBingXVariables = %q, erwartet [BX_SECRET]", missingVariables)
	}
}
