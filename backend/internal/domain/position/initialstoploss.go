package position

import "strings"

// InitialStopLossOverride ist ein manuell gesetzter ursprünglicher SL; er hat Vorrang vor dem automatisch
// ermittelten Risiko. EntryPrice ist der Entry zum Zeitpunkt des Setzens.
type InitialStopLossOverride struct {
	PositionID      string
	EntryPrice      string
	InitialStopLoss string
	SetAt           int64
}

// NewInitialStopLossOverride validiert den SL gegen Entry und Seite (über RiskDistance) und entfernt Leerzeichen.
func NewInitialStopLossOverride(positionID string, side PositionSide, entry, rawStopLoss string, setAt int64) (InitialStopLossOverride, error) {
	stopLoss := strings.TrimSpace(rawStopLoss)
	if _, err := RiskDistance(side, entry, stopLoss); err != nil {
		return InitialStopLossOverride{}, err
	}
	return InitialStopLossOverride{PositionID: positionID, EntryPrice: entry, InitialStopLoss: stopLoss, SetAt: setAt}, nil
}
