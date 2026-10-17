package position_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/decimal"
)

// Fälle portiert aus backend/src/application/position/GetOpenPositionsUseCase.test.ts.

const fetchedAt int64 = 1_791_800_000_000

func openPosition(modify func(*position.OpenPosition)) position.OpenPosition {
	created := position.OpenPosition{
		PositionID:       "9100000000000000001",
		Symbol:           "SUI-USDT",
		Side:             position.Long,
		Isolated:         true,
		PositionAmount:   "500",
		AvgOpenPrice:     "0.9600",
		MarkPrice:        "1.0100",
		Leverage:         10,
		InitialMargin:    "48.0000",
		Margin:           "60.5000",
		PositionValue:    "505",
		UnrealizedProfit: "25.0000",
		RealisedProfit:   "-0.2400",
		PnlRatio:         "0.5208",
		LiquidationPrice: "0.865",
		OpenTime:         1791500000000,
		UpdateTime:       1791600000000,
	}
	if modify != nil {
		modify(&created)
	}
	return created
}

// fakeSource liefert einen vorbereiteten Snapshot oder Fehler und zählt die Aufrufe.
type fakeSource struct {
	snapshot        position.OpenPositionsSnapshot
	err             error
	callCount       int
	receivedContext context.Context
}

func (source *fakeSource) FetchOpenPositions(ctx context.Context) (position.OpenPositionsSnapshot, error) {
	source.callCount++
	source.receivedContext = ctx
	if source.err != nil {
		return position.OpenPositionsSnapshot{}, source.err
	}
	return position.OpenPositionsSnapshot{
		Positions: slices.Clone(source.snapshot.Positions),
		FetchedAt: source.snapshot.FetchedAt,
	}, nil
}

var _ interfaces.OpenPositionSource = (*fakeSource)(nil)

func sourceWith(positions ...position.OpenPosition) *fakeSource {
	return &fakeSource{snapshot: position.OpenPositionsSnapshot{Positions: positions, FetchedAt: fetchedAt}}
}

func execute(t *testing.T, source interfaces.OpenPositionSource) appposition.OpenPositionsOverview {
	t.Helper()
	overview, err := appposition.NewGetOpenPositionsUseCase(source).Execute(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	return overview
}

func positionIDsOf(overview appposition.OpenPositionsOverview) []string {
	positionIDs := make([]string, 0, len(overview.Positions))
	for _, entry := range overview.Positions {
		positionIDs = append(positionIDs, entry.PositionID)
	}
	return positionIDs
}

func TestExecute_Sortierung(t *testing.T) {
	const sameTime int64 = 1_791_500_000_000
	testCases := []struct {
		name        string
		positions   []position.OpenPosition
		expectedIDs []string
	}{
		{
			name: "openTime absteigend",
			positions: []position.OpenPosition{
				openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.Symbol = "AAA-USDT"; p.OpenTime = 1791000000000 }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.Symbol = "BBB-USDT"; p.OpenTime = 1791900000000 }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "3"; p.Symbol = "CCC-USDT"; p.OpenTime = 1791500000000 }),
			},
			expectedIDs: []string{"2", "3", "1"},
		},
		{
			name: "gleiche openTime nach Symbol aufsteigend",
			positions: []position.OpenPosition{
				openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.Symbol = "SUI-USDT"; p.OpenTime = sameTime }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.Symbol = "BTC-USDT"; p.OpenTime = sameTime }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "4"; p.Symbol = "ZEC-USDT"; p.OpenTime = sameTime + 1 }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "3"; p.Symbol = "ETH-USDT"; p.OpenTime = sameTime }),
			},
			expectedIDs: []string{"4", "2", "3", "1"},
		},
		{
			name: "gleiche openTime und Symbol: LONG vor SHORT, dann positionId",
			positions: []position.OpenPosition{
				openPosition(func(p *position.OpenPosition) {
					p.PositionID = "3"
					p.Symbol = "BTC-USDT"
					p.Side = position.Short
					p.OpenTime = sameTime
				}),
				openPosition(func(p *position.OpenPosition) {
					p.PositionID = "2"
					p.Symbol = "BTC-USDT"
					p.Side = position.Long
					p.OpenTime = sameTime
				}),
				openPosition(func(p *position.OpenPosition) {
					p.PositionID = "1"
					p.Symbol = "BTC-USDT"
					p.Side = position.Long
					p.OpenTime = sameTime
				}),
			},
			expectedIDs: []string{"1", "2", "3"},
		},
		{
			name: "positionId als Text verglichen (strings.Compare), nicht numerisch",
			positions: []position.OpenPosition{
				openPosition(func(p *position.OpenPosition) { p.PositionID = "9"; p.OpenTime = sameTime }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "10"; p.OpenTime = sameTime }),
			},
			expectedIDs: []string{"10", "9"},
		},
		{
			name: "Großschreibung zählt nach Byte-Reihenfolge",
			positions: []position.OpenPosition{
				openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.Symbol = "B-USDT"; p.OpenTime = sameTime }),
				openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.Symbol = "1000PEPE-USDT"; p.OpenTime = sameTime }),
			},
			expectedIDs: []string{"2", "1"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := sourceWith(testCase.positions...)

			overview := execute(t, source)

			if got := positionIDsOf(overview); !slices.Equal(got, testCase.expectedIDs) {
				t.Errorf("Reihenfolge = %v, erwartet %v", got, testCase.expectedIDs)
			}
			if source.callCount != 1 {
				t.Errorf("Source %d-mal aufgerufen, erwartet 1", source.callCount)
			}
		})
	}
}

func TestExecute_PositionUnveraendertMitAbstand(t *testing.T) {
	original := openPosition(nil)

	overview := execute(t, sourceWith(original))

	if len(overview.Positions) != 1 {
		t.Fatalf("%d Positionen, erwartet 1", len(overview.Positions))
	}
	entry := overview.Positions[0]
	if entry.OpenPosition != original {
		t.Errorf("Position verändert:\n%+v\nerwartet\n%+v", entry.OpenPosition, original)
	}
	// LONG: (1.0100 − 0.865) / 1.0100 = 0.1435643564… → 0.14356436
	if entry.LiquidationDistanceRatio == nil || *entry.LiquidationDistanceRatio != "0.14356436" {
		t.Errorf("LiquidationDistanceRatio = %v, erwartet 0.14356436", describeRatio(entry.LiquidationDistanceRatio))
	}
}

func TestExecute_AbstandJeSeiteUndNilOhneLiquidation(t *testing.T) {
	source := sourceWith(
		openPosition(func(p *position.OpenPosition) { p.PositionID = "3"; p.LiquidationPrice = "0"; p.OpenTime = 1 }),
		openPosition(func(p *position.OpenPosition) {
			p.PositionID = "2"
			p.Side = position.Short
			p.MarkPrice = "4175.00"
			p.LiquidationPrice = "4410.5"
			p.OpenTime = 2
		}),
		openPosition(func(p *position.OpenPosition) {
			p.PositionID = "1"
			p.MarkPrice = "81365.0"
			p.LiquidationPrice = "76507"
			p.OpenTime = 3
		}),
	)

	overview := execute(t, source)

	expected := []struct {
		positionID string
		ratio      *string
	}{
		{"1", ptr("0.05970626")},
		{"2", ptr("0.05640719")},
		{"3", nil},
	}
	if len(overview.Positions) != len(expected) {
		t.Fatalf("%d Positionen, erwartet %d", len(overview.Positions), len(expected))
	}
	for index, want := range expected {
		entry := overview.Positions[index]
		if entry.PositionID != want.positionID {
			t.Errorf("Index %d: positionId = %q, erwartet %q", index, entry.PositionID, want.positionID)
		}
		if describeRatio(entry.LiquidationDistanceRatio) != describeRatio(want.ratio) {
			t.Errorf("Index %d: Abstand = %s, erwartet %s", index,
				describeRatio(entry.LiquidationDistanceRatio), describeRatio(want.ratio))
		}
	}
}

func TestExecute_SummeUndFetchedAt(t *testing.T) {
	source := sourceWith(
		openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.UnrealizedProfit = "25.0000" }),
		openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.UnrealizedProfit = "-6.2500" }),
		openPosition(func(p *position.OpenPosition) { p.PositionID = "3"; p.UnrealizedProfit = "0.00000001" }),
	)
	source.snapshot.FetchedAt = 1_791_123_456_789

	overview := execute(t, source)

	if overview.TotalUnrealizedProfit != "18.75000001" {
		t.Errorf("Summe = %q, erwartet 18.75000001", overview.TotalUnrealizedProfit)
	}
	if overview.FetchedAt != 1_791_123_456_789 {
		t.Errorf("FetchedAt = %d, erwartet 1791123456789", overview.FetchedAt)
	}
}

func TestExecute_LeereListeIstNichtNil(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		positions []position.OpenPosition
	}{
		{"Source liefert leeres Slice", []position.OpenPosition{}},
		{"Source liefert nil", nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := &fakeSource{snapshot: position.OpenPositionsSnapshot{Positions: testCase.positions, FetchedAt: fetchedAt}}

			overview := execute(t, source)

			if overview.Positions == nil {
				t.Fatalf("Positions ist nil, erwartet leeres Slice (JSON [])")
			}
			if len(overview.Positions) != 0 {
				t.Errorf("%d Positionen, erwartet 0", len(overview.Positions))
			}
			if overview.TotalUnrealizedProfit != "0" {
				t.Errorf("Summe = %q, erwartet 0", overview.TotalUnrealizedProfit)
			}
			if overview.FetchedAt != fetchedAt {
				t.Errorf("FetchedAt = %d, erwartet %d", overview.FetchedAt, fetchedAt)
			}
			encoded, err := json.Marshal(overview.Positions)
			if err != nil || string(encoded) != "[]" {
				t.Errorf("JSON = %s (%v), erwartet []", encoded, err)
			}
		})
	}
}

func TestExecute_VeraendertDenSnapshotDerSourceNicht(t *testing.T) {
	positions := []position.OpenPosition{
		openPosition(func(p *position.OpenPosition) { p.PositionID = "1"; p.OpenTime = 1 }),
		openPosition(func(p *position.OpenPosition) { p.PositionID = "2"; p.OpenTime = 2 }),
	}
	source := &sharedSliceSource{positions: positions}

	execute(t, source)

	if positions[0].PositionID != "1" || positions[1].PositionID != "2" {
		t.Errorf("Slice der Source wurde umsortiert: %v, %v", positions[0].PositionID, positions[1].PositionID)
	}
}

// sharedSliceSource gibt immer dasselbe Slice heraus (wie ein Cache), ohne Kopie.
type sharedSliceSource struct {
	positions []position.OpenPosition
}

func (source *sharedSliceSource) FetchOpenPositions(context.Context) (position.OpenPositionsSnapshot, error) {
	return position.OpenPositionsSnapshot{Positions: source.positions, FetchedAt: fetchedAt}, nil
}

func TestExecute_ReichtContextDurch(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "anfrage-42")
	source := sourceWith(openPosition(nil))

	if _, err := appposition.NewGetOpenPositionsUseCase(source).Execute(ctx); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if source.receivedContext == nil || source.receivedContext.Value(contextKey{}) != "anfrage-42" {
		t.Errorf("Source hat den Context des Aufrufers nicht bekommen")
	}
}

func TestExecute_Fehler(t *testing.T) {
	t.Run("OpenPositionsUnavailableError wird unverändert durchgereicht", func(t *testing.T) {
		unavailable := &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}
		source := &fakeSource{err: unavailable}

		_, err := appposition.NewGetOpenPositionsUseCase(source).Execute(context.Background())

		var gotUnavailable *position.OpenPositionsUnavailableError
		if !errors.As(err, &gotUnavailable) || gotUnavailable != unavailable {
			t.Fatalf("Fehler = %v, erwartet dieselbe Instanz", err)
		}
	})

	t.Run("unbekannter Fehler geht durch", func(t *testing.T) {
		unknown := errors.New("Programmierfehler")
		source := &fakeSource{err: unknown}

		_, err := appposition.NewGetOpenPositionsUseCase(source).Execute(context.Background())

		if !errors.Is(err, unknown) {
			t.Fatalf("Fehler = %v, erwartet den Source-Fehler", err)
		}
	})

	t.Run("ungültiger Betrag ergibt InvalidDecimalError", func(t *testing.T) {
		source := sourceWith(openPosition(func(p *position.OpenPosition) { p.UnrealizedProfit = "1e2" }))

		_, err := appposition.NewGetOpenPositionsUseCase(source).Execute(context.Background())

		var invalidDecimal *decimal.InvalidDecimalError
		if !errors.As(err, &invalidDecimal) {
			t.Fatalf("Fehler = %v (%T), erwartet *decimal.InvalidDecimalError", err, err)
		}
	})
}

func ptr(value string) *string { return &value }

func describeRatio(ratio *string) string {
	if ratio == nil {
		return "<nil>"
	}
	return *ratio
}
