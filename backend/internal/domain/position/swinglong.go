package position

// SwingLongSpan ist der Zeitraum einer Swing-Long-Position (docs/domain.md, Näherung für den realisierten PnL).
type SwingLongSpan struct {
	PositionID       string
	Symbol           string
	OpenTime         int64  // Epoch-ms
	CloseTime        int64  // Epoch-ms; 0 = noch offen
	UnrealizedProfit string // nur bei offenen Positionen, sonst ""
}

// Covers sagt, ob ein Income mit Symbol und Zeit zu dieser Position zählt: gleiches Symbol und
// OpenTime <= timeMs <= CloseTime (offen: nach oben unbegrenzt).
func (span SwingLongSpan) Covers(symbol string, timeMs int64) bool {
	return symbol == span.Symbol && timeMs >= span.OpenTime && (span.CloseTime == 0 || timeMs <= span.CloseTime)
}
