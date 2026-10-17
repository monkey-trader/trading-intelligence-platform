package config

import "strings"

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
