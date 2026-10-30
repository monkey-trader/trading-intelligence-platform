// Package price enthält den Kern der Live-Kurse.
package price

import "errors"

// Quote ist der letzte Handelspreis eines Symbols. Zahlen bleiben Dezimal-Strings, At ist Epoch-ms des Abrufs.
type Quote struct {
	Symbol           string
	Price            string
	Change24hPercent string
	At               int64
}

// ErrPriceUnavailable meldet: Die Kursquelle liefert gerade keine Kurse.
var ErrPriceUnavailable = errors.New("price: Kursquelle nicht verfügbar")
