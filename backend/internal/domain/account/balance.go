// Package account enthält die Kontosicht: Guthaben und Monatsfenster.
package account

// Balance sind die Kontowerte als exakte Dezimal-Strings.
type Balance struct {
	Balance         string
	Equity          string
	AvailableMargin string
	// UnrealizedProfit ist der unrealisierte PnL des Kontos; leer, wenn BingX das Feld nicht liefert.
	UnrealizedProfit string
}
