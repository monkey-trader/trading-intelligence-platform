package position_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
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

func TestExecute_PositionUnveraendert(t *testing.T) {
	original := openPosition(nil)

	overview := execute(t, sourceWith(original))

	if len(overview.Positions) != 1 {
		t.Fatalf("%d Positionen, erwartet 1", len(overview.Positions))
	}
	if overview.Positions[0].OpenPosition != original {
		t.Errorf("Position verändert:\n%+v\nerwartet\n%+v", overview.Positions[0].OpenPosition, original)
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

// --- SL/TP/Break-even ---

type fakeOrders struct {
	orders []chart.ProtectiveOrder
	err    error
}

func (source *fakeOrders) FetchProtectiveOrders(context.Context) (chart.ProtectiveOrdersSnapshot, error) {
	if source.err != nil {
		return chart.ProtectiveOrdersSnapshot{}, source.err
	}
	return chart.ProtectiveOrdersSnapshot{Orders: source.orders, FetchedAt: fetchedAt}, nil
}

var _ chartinterfaces.ProtectiveOrderSource = (*fakeOrders)(nil)

func order(positionID string, kind chart.ProtectiveKind, trigger string) chart.ProtectiveOrder {
	return chart.ProtectiveOrder{OrderID: positionID + trigger, PositionID: positionID, Kind: kind, TriggerPrice: trigger}
}

func executeWithOrders(t *testing.T, positions []position.OpenPosition, orders *fakeOrders) appposition.OpenPositionsOverview {
	t.Helper()
	overview, err := appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(positions...), orders).Execute(context.Background())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	return overview
}

func TestExecute_Schutzorders(t *testing.T) {
	const ownID = "9100000000000000001"
	short := func(p *position.OpenPosition) {
		p.PositionID = "S1"
		p.Side = position.Short
		p.AvgOpenPrice = "1.0000"
	}
	testCases := []struct {
		name       string
		position   position.OpenPosition
		orders     []chart.ProtectiveOrder
		stopLoss   *string
		takeProfit *string
		breakEven  bool
	}{
		{"LONG SL im Gewinn, mehrere SLs: höchster", openPosition(nil),
			[]chart.ProtectiveOrder{order(ownID, chart.StopLoss, "0.90"), order(ownID, chart.StopLoss, "0.97"), order(ownID, chart.TakeProfit, "1.50"), order(ownID, chart.TakeProfit, "1.20")},
			ptr("0.97"), ptr("1.20"), true},
		{"LONG SL exakt auf Entry ist Break-even", openPosition(nil),
			[]chart.ProtectiveOrder{order(ownID, chart.StopLoss, "0.96")}, ptr("0.96"), nil, true},
		{"LONG SL im Verlust", openPosition(nil),
			[]chart.ProtectiveOrder{order(ownID, chart.StopLoss, "0.90")}, ptr("0.90"), nil, false},
		{"LONG ohne SL, nur TP", openPosition(nil),
			[]chart.ProtectiveOrder{order(ownID, chart.TakeProfit, "1.2")}, nil, ptr("1.2"), false},
		{"LONG ohne Orders", openPosition(nil), nil, nil, nil, false},
		{"SHORT mehrere SLs: niedrigster, im Gewinn", openPosition(short),
			[]chart.ProtectiveOrder{order("S1", chart.StopLoss, "1.10"), order("S1", chart.StopLoss, "0.99"), order("S1", chart.TakeProfit, "0.70"), order("S1", chart.TakeProfit, "0.80")},
			ptr("0.99"), ptr("0.80"), true},
		{"SHORT SL im Verlust", openPosition(short),
			[]chart.ProtectiveOrder{order("S1", chart.StopLoss, "1.10")}, ptr("1.10"), nil, false},
		{"SHORT SL exakt auf Entry", openPosition(short),
			[]chart.ProtectiveOrder{order("S1", chart.StopLoss, "1")}, ptr("1"), nil, true},
		{"Orders anderer Position werden ignoriert", openPosition(nil),
			[]chart.ProtectiveOrder{order("andere", chart.StopLoss, "0.99"), order("andere", chart.TakeProfit, "1.05"), order(ownID, chart.StopLoss, "0.90")},
			ptr("0.90"), nil, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			overview := executeWithOrders(t, []position.OpenPosition{testCase.position}, &fakeOrders{orders: testCase.orders})

			entry := overview.Positions[0]
			if describeRatio(entry.StopLoss) != describeRatio(testCase.stopLoss) {
				t.Errorf("StopLoss = %s, erwartet %s", describeRatio(entry.StopLoss), describeRatio(testCase.stopLoss))
			}
			if describeRatio(entry.TakeProfit) != describeRatio(testCase.takeProfit) {
				t.Errorf("TakeProfit = %s, erwartet %s", describeRatio(entry.TakeProfit), describeRatio(testCase.takeProfit))
			}
			if entry.BreakEven == nil || *entry.BreakEven != testCase.breakEven {
				t.Errorf("BreakEven = %v, erwartet %v", entry.BreakEven, testCase.breakEven)
			}
		})
	}
}

func TestExecute_OrdersFehlerLiefertListeMitNullFeldern(t *testing.T) {
	orders := &fakeOrders{err: errors.New("orders nicht ladbar")}

	overview := executeWithOrders(t, []position.OpenPosition{openPosition(nil)}, orders)

	if len(overview.Positions) != 1 {
		t.Fatalf("%d Positionen, erwartet 1", len(overview.Positions))
	}
	entry := overview.Positions[0]
	if entry.StopLoss != nil || entry.TakeProfit != nil || entry.BreakEven != nil {
		t.Errorf("SL/TP/BE = %v/%v/%v, erwartet alle nil", entry.StopLoss, entry.TakeProfit, entry.BreakEven)
	}
	if overview.TotalUnrealizedProfit != "25" {
		t.Errorf("Summe = %q, erwartet 25", overview.TotalUnrealizedProfit)
	}
}
