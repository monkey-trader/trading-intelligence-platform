// Package position enthält den Domain-Kern für Positionen.
package position

import "fmt"

// PositionDomainError ist die gemeinsame Sicht auf alle fachlichen Positions-Fehler.
// errors.As mit einem Ziel vom Typ PositionDomainError funktioniert für alle *Invalid…Error.
type PositionDomainError interface {
	error
	positionDomainError()
}

// InvalidDecimalAmountError meldet: Der Betrag ist kein Dezimal-String.
type InvalidDecimalAmountError struct{ Value string }

// InvalidTimestampError meldet: Der Zeitstempel ist nicht größer als 0.
type InvalidTimestampError struct{ Value int64 }

// InvalidLeverageError meldet: Der Hebel ist nicht endlich oder nicht größer als 0.
type InvalidLeverageError struct{ Value float64 }

// InvalidPositionIDError meldet: Die Positions-ID besteht nicht nur aus Ziffern.
type InvalidPositionIDError struct{ Value string }

// InvalidPositionSideError meldet: Die Seite ist weder LONG noch SHORT.
type InvalidPositionSideError struct{ Value string }

// InvalidTradingSymbolError meldet: Das Symbol hat nicht die Form BASIS-QUOTE.
type InvalidTradingSymbolError struct{ Value string }

// UpdateBeforeOpenError meldet: Die letzte Änderung liegt vor der Eröffnung.
type UpdateBeforeOpenError struct {
	PositionID string
	OpenTime   int64
	UpdateTime int64
}

// SuspiciousEmptyPositionHistoryError meldet: Income-Einträge gefunden, aber positionHistory ist leer.
type SuspiciousEmptyPositionHistoryError struct {
	Symbol string
	From   int64
	To     int64
}

func (err *InvalidDecimalAmountError) Error() string {
	return fmt.Sprintf("Ungültiger Betrag: %q", err.Value)
}

func (err *InvalidTimestampError) Error() string {
	return fmt.Sprintf("Ungültiger Zeitstempel: %d", err.Value)
}

func (err *InvalidLeverageError) Error() string {
	return fmt.Sprintf("Ungültiger Hebel: %v", err.Value)
}

func (err *InvalidPositionIDError) Error() string {
	return fmt.Sprintf("Ungültige Positions-ID: %q", err.Value)
}

func (err *InvalidPositionSideError) Error() string {
	return fmt.Sprintf("Ungültige Seite: %q", err.Value)
}

func (err *InvalidTradingSymbolError) Error() string {
	return fmt.Sprintf("Ungültiges Symbol: %q", err.Value)
}

func (err *UpdateBeforeOpenError) Error() string {
	return fmt.Sprintf("Änderungszeit %d liegt vor Eröffnungszeit %d (Position %s).",
		err.UpdateTime, err.OpenTime, err.PositionID)
}

func (err *SuspiciousEmptyPositionHistoryError) Error() string {
	return fmt.Sprintf("Verdächtig: Income vorhanden für %s, aber positionHistory ist leer (%d–%d).",
		err.Symbol, err.From, err.To)
}

func (*InvalidDecimalAmountError) positionDomainError()           {}
func (*InvalidTimestampError) positionDomainError()               {}
func (*InvalidLeverageError) positionDomainError()                {}
func (*InvalidPositionIDError) positionDomainError()              {}
func (*InvalidPositionSideError) positionDomainError()            {}
func (*InvalidTradingSymbolError) positionDomainError()           {}
func (*UpdateBeforeOpenError) positionDomainError()               {}
func (*SuspiciousEmptyPositionHistoryError) positionDomainError() {}
