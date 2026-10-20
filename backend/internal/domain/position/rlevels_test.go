package position_test

import (
	"errors"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Erwartete Signaturen (Ergebnisse sind minimal formatierte Dezimal-Strings, ohne Nachkomma-Nullen):
//   RiskDistance(side, entry, initialStopLoss string) (string, error)
//   RLevels(side, entry, initialStopLoss string) ([]RLevel, error)  // RLevel{Multiple int; Price, Percent string}, 1R,2R,3R
//   CurrentRMultiple(side, entry, risk, markPrice string) (string, error) // 4 Nachkommastellen, getrimmt
//   ReachedAt(side, levelPrice string, openTime int64, candles []CandleRange) (*int64, error)
// Fehler: *StopLossAtEntryError, *StopLossWrongSideError, *decimal.InvalidDecimalError, *decimal.DivisionByZeroError.

func TestRiskDistance(t *testing.T) {
	tests := []struct {
		name         string
		side         position.PositionSide
		entry, stop  string
		want         string
		wantErrAtEnt bool
		wantWrong    bool
		wantInvalid  bool
	}{
		{name: "LONG", side: position.Long, entry: "100", stop: "98", want: "2"},
		{name: "SHORT", side: position.Short, entry: "100", stop: "102.5", want: "2.5"},
		{name: "Nachkommastellen exakt", side: position.Long, entry: "0.9600", stop: "0.9000", want: "0.06"},
		{name: "SL gleich Entry", side: position.Long, entry: "100", stop: "100.00", wantErrAtEnt: true},
		{name: "LONG SL über Entry", side: position.Long, entry: "100", stop: "101", wantWrong: true},
		{name: "SHORT SL unter Entry", side: position.Short, entry: "100", stop: "99", wantWrong: true},
		{name: "SL ungültig", side: position.Long, entry: "100", stop: "abc", wantInvalid: true},
		{name: "Entry ungültig", side: position.Long, entry: "", stop: "98", wantInvalid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := position.RiskDistance(test.side, test.entry, test.stop)
			var atEntry *position.StopLossAtEntryError
			var wrongSide *position.StopLossWrongSideError
			var invalid *decimal.InvalidDecimalError
			switch {
			case test.wantErrAtEnt:
				if !errors.As(err, &atEntry) {
					t.Fatalf("err = %v", err)
				}
			case test.wantWrong:
				if !errors.As(err, &wrongSide) {
					t.Fatalf("err = %v", err)
				}
			case test.wantInvalid:
				if !errors.As(err, &invalid) {
					t.Fatalf("err = %v", err)
				}
			default:
				if err != nil || got != test.want {
					t.Fatalf("got %q err %v, want %q", got, err, test.want)
				}
			}
		})
	}
}

func TestRLevels(t *testing.T) {
	tests := []struct {
		name        string
		side        position.PositionSide
		entry, stop string
		want        []position.RLevel
	}{
		{"LONG", position.Long, "100", "98", []position.RLevel{
			{Multiple: 1, Price: "102", Percent: "0.02"},
			{Multiple: 2, Price: "104", Percent: "0.04"},
			{Multiple: 3, Price: "106", Percent: "0.06"},
		}},
		{"SHORT", position.Short, "100", "102", []position.RLevel{
			{Multiple: 1, Price: "98", Percent: "-0.02"},
			{Multiple: 2, Price: "96", Percent: "-0.04"},
			{Multiple: 3, Price: "94", Percent: "-0.06"},
		}},
		{"Nachkommastellen", position.Long, "0.9600", "0.9000", []position.RLevel{
			{Multiple: 1, Price: "1.02", Percent: "0.0625"},
			{Multiple: 2, Price: "1.08", Percent: "0.125"},
			{Multiple: 3, Price: "1.14", Percent: "0.1875"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := position.RLevels(test.side, test.entry, test.stop)
			if err != nil || len(got) != 3 {
				t.Fatalf("got %+v err %v", got, err)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Errorf("Level %d = %+v, want %+v", index+1, got[index], test.want[index])
				}
			}
		})
	}
}

func TestRLevels_Fehler(t *testing.T) {
	var atEntry *position.StopLossAtEntryError
	if _, err := position.RLevels(position.Long, "100", "100"); !errors.As(err, &atEntry) {
		t.Errorf("SL == Entry: %v", err)
	}
	var wrongSide *position.StopLossWrongSideError
	if _, err := position.RLevels(position.Short, "100", "99"); !errors.As(err, &wrongSide) {
		t.Errorf("falsche Seite: %v", err)
	}
	var invalid *decimal.InvalidDecimalError
	if _, err := position.RLevels(position.Long, "100", "x"); !errors.As(err, &invalid) {
		t.Errorf("ungültig: %v", err)
	}
}

func TestCurrentRMultiple(t *testing.T) {
	tests := []struct {
		name                    string
		side                    position.PositionSide
		entry, risk, mark, want string
	}{
		{"LONG im Gewinn", position.Long, "100", "2", "103", "1.5"},
		{"LONG im Verlust", position.Long, "100", "2", "99", "-0.5"},
		{"SHORT im Gewinn", position.Short, "100", "2", "96", "2"},
		{"SHORT im Verlust", position.Short, "100", "2", "101", "-0.5"},
		{"Mark gleich Entry", position.Long, "100", "2", "100", "0"},
		{"Rundung auf 4 Stellen", position.Long, "0.96", "0.06", "1.01", "0.8333"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := position.CurrentRMultiple(test.side, test.entry, test.risk, test.mark)
			if err != nil || got != test.want {
				t.Fatalf("got %q err %v, want %q", got, err, test.want)
			}
		})
	}
	var zero *decimal.DivisionByZeroError
	if _, err := position.CurrentRMultiple(position.Long, "100", "0", "101"); !errors.As(err, &zero) {
		t.Errorf("Risiko 0: %v", err)
	}
	var invalid *decimal.InvalidDecimalError
	if _, err := position.CurrentRMultiple(position.Long, "100", "2", "n/a"); !errors.As(err, &invalid) {
		t.Errorf("Mark ungültig: %v", err)
	}
}

func TestReachedAt(t *testing.T) {
	const openTime int64 = 1_000
	candles := []position.CandleRange{
		{OpenTime: 500, High: "200", Low: "1"},       // vor Eröffnung: ignorieren
		{OpenTime: 1_000, High: "103.99", Low: "99"}, // genau openTime zählt, aber unter Level
		{OpenTime: 2_000, High: "104", Low: "97"},    // genau auf Level (LONG: >=)
		{OpenTime: 3_000, High: "110", Low: "90"},
	}
	t.Run("LONG berührt genau", func(t *testing.T) {
		got, err := position.ReachedAt(position.Long, "104", openTime, candles)
		if err != nil || got == nil || *got != 2_000 {
			t.Fatalf("got %v err %v", got, err)
		}
	})
	t.Run("LONG knapp nicht erreicht", func(t *testing.T) {
		got, err := position.ReachedAt(position.Long, "104.01", openTime, candles[:3])
		if err != nil || got != nil {
			t.Fatalf("got %v err %v", got, err)
		}
	})
	t.Run("SHORT low <= Level", func(t *testing.T) {
		got, err := position.ReachedAt(position.Short, "97", openTime, candles)
		if err != nil || got == nil || *got != 2_000 {
			t.Fatalf("got %v err %v", got, err)
		}
	})
	t.Run("Kerze vor openTime wird ignoriert", func(t *testing.T) {
		got, err := position.ReachedAt(position.Short, "1", openTime, candles)
		if err != nil || got != nil {
			t.Fatalf("got %v err %v", got, err)
		}
	})
	t.Run("keine Kerzen", func(t *testing.T) {
		got, err := position.ReachedAt(position.Long, "104", openTime, nil)
		if err != nil || got != nil {
			t.Fatalf("got %v err %v", got, err)
		}
	})
	t.Run("ungültiger Preis", func(t *testing.T) {
		var invalid *decimal.InvalidDecimalError
		if _, err := position.ReachedAt(position.Long, "x", openTime, candles); !errors.As(err, &invalid) {
			t.Fatalf("err = %v", err)
		}
		bad := []position.CandleRange{{OpenTime: 2_000, High: "?", Low: "?"}}
		if _, err := position.ReachedAt(position.Long, "104", openTime, bad); !errors.As(err, &invalid) {
			t.Fatalf("Kerze: err = %v", err)
		}
	})
}
