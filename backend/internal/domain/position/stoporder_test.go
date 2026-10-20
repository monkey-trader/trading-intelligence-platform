package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Erwartet: EarliestInitialStopLoss(side PositionSide, entry string, records []StopOrderRecord) (*string, error)
// mit StopOrderRecord{PositionID, TriggerPrice string; Time int64}.
// Der zeitlich früheste Record auf der Verlustseite (LONG < Entry, SHORT > Entry; gleich Entry zählt nicht).

func TestEarliestInitialStopLoss(t *testing.T) {
	tests := []struct {
		name    string
		side    position.PositionSide
		entry   string
		records []position.StopOrderRecord
		want    string // "" = nil
	}{
		{"LONG frühester gewinnt, unabhängig von der Reihenfolge", position.Long, "100",
			[]position.StopOrderRecord{{TriggerPrice: "99", Time: 300}, {TriggerPrice: "95", Time: 100}, {TriggerPrice: "97", Time: 200}}, "95"},
		{"LONG früheste Order auf Gewinnseite wird übersprungen", position.Long, "100",
			[]position.StopOrderRecord{{TriggerPrice: "101", Time: 50}, {TriggerPrice: "96", Time: 100}}, "96"},
		{"SHORT frühester über Entry", position.Short, "100",
			[]position.StopOrderRecord{{TriggerPrice: "104", Time: 200}, {TriggerPrice: "103", Time: 100}}, "103"},
		{"SHORT Trigger unter Entry zählt nicht", position.Short, "100",
			[]position.StopOrderRecord{{TriggerPrice: "99", Time: 10}}, ""},
		{"Trigger gleich Entry zählt nicht (Dezimalvergleich)", position.Long, "0.9600",
			[]position.StopOrderRecord{{TriggerPrice: "0.96", Time: 10}}, ""},
		{"keine Records", position.Long, "100", nil, ""},
		{"Preis unverändert zurückgegeben", position.Long, "80000",
			[]position.StopOrderRecord{{TriggerPrice: "79000.0", Time: 1}}, "79000.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := position.EarliestInitialStopLoss(test.side, test.entry, test.records)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if test.want == "" {
				if got != nil {
					t.Fatalf("got %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != test.want {
				t.Fatalf("got %v, want %q", got, test.want)
			}
		})
	}
}

func TestEarliestInitialStopLoss_UngueltigeDezimalzahl(t *testing.T) {
	var invalid *decimal.InvalidDecimalError
	_, err := position.EarliestInitialStopLoss(position.Long, "100", []position.StopOrderRecord{{TriggerPrice: "abc", Time: 1}})
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v", err)
	}
	_, err = position.EarliestInitialStopLoss(position.Long, "", []position.StopOrderRecord{{TriggerPrice: "9", Time: 1}})
	if !errors.As(err, &invalid) {
		t.Fatalf("Entry ungültig: err = %v", err)
	}
}
