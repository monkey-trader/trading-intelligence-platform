package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Erwartet: position.NewInitialStopLossOverride(positionID string, side PositionSide, entry, rawStopLoss string,
// setAt int64) (InitialStopLossOverride, error) mit
// InitialStopLossOverride{PositionID, EntryPrice, InitialStopLoss string; SetAt int64}.
// Validierung über RiskDistance; die Eingabe wird nur von Leerzeichen befreit, sonst unverändert gespeichert.

func TestNewInitialStopLossOverride(t *testing.T) {
	tests := []struct {
		name      string
		side      position.PositionSide
		stop      string
		wantAtEnt bool
		wantWrong bool
		wantBad   bool
	}{
		{name: "LONG unter Entry", side: position.Long, stop: "0.9000"},
		{name: "SHORT über Entry", side: position.Short, stop: "1.0200"},
		{name: "Leerzeichen werden entfernt", side: position.Long, stop: " 0.9000 "},
		{name: "LONG gleich Entry", side: position.Long, stop: "0.96", wantAtEnt: true},
		{name: "SHORT gleich Entry", side: position.Short, stop: "0.9600", wantAtEnt: true},
		{name: "LONG über Entry", side: position.Long, stop: "1.0", wantWrong: true},
		{name: "SHORT unter Entry", side: position.Short, stop: "0.9", wantWrong: true},
		{name: "keine Zahl", side: position.Long, stop: "abc", wantBad: true},
		{name: "leer", side: position.Long, stop: "", wantBad: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := position.NewInitialStopLossOverride("9100000000000000001", test.side, "0.9600", test.stop, 1_791_800_000_000)
			var atEntry *position.StopLossAtEntryError
			var wrongSide *position.StopLossWrongSideError
			var invalid *decimal.InvalidDecimalError
			switch {
			case test.wantAtEnt:
				if !errors.As(err, &atEntry) {
					t.Fatalf("err = %v", err)
				}
			case test.wantWrong:
				if !errors.As(err, &wrongSide) {
					t.Fatalf("err = %v", err)
				}
			case test.wantBad:
				if !errors.As(err, &invalid) {
					t.Fatalf("err = %v", err)
				}
			default:
				want := position.InitialStopLossOverride{PositionID: "9100000000000000001", EntryPrice: "0.9600",
					InitialStopLoss: test.stop, SetAt: 1_791_800_000_000}
				if want.InitialStopLoss == " 0.9000 " {
					want.InitialStopLoss = "0.9000"
				}
				if err != nil || got != want {
					t.Fatalf("got %+v err %v, want %+v", got, err, want)
				}
			}
		})
	}
}
