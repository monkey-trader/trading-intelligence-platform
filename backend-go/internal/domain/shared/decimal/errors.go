package decimal

import "fmt"

// InvalidDecimalError meldet: Eingabe passt nicht auf ^-?\d+(\.\d+)?$ oder die Skala ist ungültig.
type InvalidDecimalError struct {
	Value string
	// Reason ergänzt die Meldung, z.B. bei ungültiger Skala; leer bei ungültigem Betrag.
	Reason string
}

func (err *InvalidDecimalError) Error() string {
	if err.Reason != "" {
		return "Ungültige Dezimalzahl: " + err.Reason
	}
	return fmt.Sprintf("Ungültiger Betrag: %q", err.Value)
}

// DivisionByZeroError meldet: Der Divisor ist numerisch 0.
type DivisionByZeroError struct {
	Dividend string
	Divisor  string
}

func (err *DivisionByZeroError) Error() string {
	return fmt.Sprintf("Division durch 0: %q / %q", err.Dividend, err.Divisor)
}

// InvalidScaleError meldet: Die Skala von Divide ist negativ.
type InvalidScaleError struct {
	Scale int
}

func (err *InvalidScaleError) Error() string {
	return fmt.Sprintf("Ungültige Skala: %d (muss >= 0 sein)", err.Scale)
}
