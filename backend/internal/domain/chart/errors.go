package chart

import (
	"fmt"
	"strconv"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// UnsupportedIntervalError meldet: Das Intervall gehört nicht zu den fünf unterstützten Werten.
type UnsupportedIntervalError struct{ Value string }

func (err *UnsupportedIntervalError) Error() string {
	return fmt.Sprintf("Kerzenintervall %q wird nicht unterstützt.", err.Value)
}

// LimitOutOfRangeError meldet: Das Limit liegt außerhalb von MinLimit bis MaxLimit.
type LimitOutOfRangeError struct{ Value int }

func (err *LimitOutOfRangeError) Error() string {
	return fmt.Sprintf("Limit %s liegt nicht zwischen %d und %d.", strconv.Itoa(err.Value), MinLimit, MaxLimit)
}

// TriggerPriceMustBePositiveError meldet: Der Auslösepreis ist nicht größer als 0.
type TriggerPriceMustBePositiveError struct{ Value string }

func (err *TriggerPriceMustBePositiveError) Error() string {
	return fmt.Sprintf("Auslösepreis %q muss größer als 0 sein.", err.Value)
}

// PositionNotFoundError meldet: Es gibt keine offene Position mit dieser ID.
type PositionNotFoundError struct{ PositionID string }

func (err *PositionNotFoundError) Error() string {
	return fmt.Sprintf("Keine offene Position mit der ID %q.", err.PositionID)
}

// KlineParameterError meldet: BingX lehnt die Kerzen-Parameter ab (z.B. Code 109400).
type KlineParameterError struct {
	Code int
	Msg  string
}

func (err *KlineParameterError) Error() string {
	return fmt.Sprintf("BingX lehnt die Kerzen-Parameter ab (%d): %s", err.Code, err.Msg)
}

// SymbolNotFoundError meldet: BingX kennt das Symbol nicht.
type SymbolNotFoundError struct{ Symbol string }

func (err *SymbolNotFoundError) Error() string {
	return fmt.Sprintf("BingX kennt das Symbol %q nicht.", err.Symbol)
}

// DataUnavailableError meldet: Kerzen oder Schutzorders sind gerade nicht lieferbar. Cause nur fürs Log.
type DataUnavailableError struct {
	Reason position.UnavailableReason
	Cause  error
}

func (err *DataUnavailableError) Error() string {
	return "Chart-Daten nicht verfügbar: " + string(err.Reason)
}

// Unwrap liefert die Ursache.
func (err *DataUnavailableError) Unwrap() error { return err.Cause }
