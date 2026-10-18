package chart_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"testing"

	appchart "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

const (
	suiPositionID       = "9100000000000000001" // > 2^53, muss String bleiben
	ordersFetchedAt     = int64(1_791_800_000_000)
	suiOpenTime         = int64(1_791_500_000_000)
	suiSymbol           = "SUI-USDT"
	suiAvgOpenPrice     = "0.9600"
	candleSourceTimeOne = int64(1_791_799_100_000)
)

type fakePositionSource struct {
	positions []position.OpenPosition
	err       error
}

func (source *fakePositionSource) FetchOpenPositions(context.Context) (position.OpenPositionsSnapshot, error) {
	if source.err != nil {
		return position.OpenPositionsSnapshot{}, source.err
	}
	return position.OpenPositionsSnapshot{Positions: source.positions, FetchedAt: ordersFetchedAt}, nil
}

type fakeOrderSource struct {
	orders    []chart.ProtectiveOrder
	err       error
	callCount int
}

func (source *fakeOrderSource) FetchProtectiveOrders(context.Context) (chart.ProtectiveOrdersSnapshot, error) {
	source.callCount++
	if source.err != nil {
		return chart.ProtectiveOrdersSnapshot{}, source.err
	}
	return chart.ProtectiveOrdersSnapshot{Orders: source.orders, FetchedAt: ordersFetchedAt}, nil
}

type candleRequest struct {
	symbol   string
	interval chart.CandleInterval
	limit    int
}

type fakeCandleSource struct {
	candles   []chart.Candle
	err       error
	requests  []candleRequest
	callCount int
}

func (source *fakeCandleSource) FetchCandles(_ context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	source.callCount++
	source.requests = append(source.requests, candleRequest{symbol, interval, limit})
	return source.candles, source.err
}

// fakeCandleRepository speichert per Schlüssel (symbol, interval, openTime) wie die SQLite-Tabelle.
type fakeCandleRepository struct {
	stored       map[string]chart.Candle
	upsertErr    error
	listErr      error
	listRequests []candleRequest
}

func candleKey(candle chart.Candle) string {
	return candle.Symbol + "|" + string(candle.Interval) + "|" + strconv.FormatInt(candle.OpenTime, 10)
}

func (repository *fakeCandleRepository) UpsertCandles(_ context.Context, candles []chart.Candle) error {
	if repository.upsertErr != nil {
		return repository.upsertErr
	}
	if repository.stored == nil {
		repository.stored = map[string]chart.Candle{}
	}
	for _, candle := range candles {
		repository.stored[candleKey(candle)] = candle
	}
	return nil
}

func (repository *fakeCandleRepository) ListCandles(_ context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	repository.listRequests = append(repository.listRequests, candleRequest{symbol, interval, limit})
	if repository.listErr != nil {
		return nil, repository.listErr
	}
	var matching []chart.Candle
	for _, candle := range repository.stored {
		if candle.Symbol == symbol && candle.Interval == interval {
			matching = append(matching, candle)
		}
	}
	slices.SortFunc(matching, func(left, right chart.Candle) int { return int(left.OpenTime - right.OpenTime) })
	if len(matching) > limit {
		matching = matching[len(matching)-limit:]
	}
	return matching, nil
}

var (
	_ chartinterfaces.CandleSource          = (*fakeCandleSource)(nil)
	_ chartinterfaces.CandleRepository      = (*fakeCandleRepository)(nil)
	_ chartinterfaces.ProtectiveOrderSource = (*fakeOrderSource)(nil)
)

func suiPosition() position.OpenPosition {
	return position.OpenPosition{
		PositionID: suiPositionID, Symbol: suiSymbol, Side: position.Long, AvgOpenPrice: suiAvgOpenPrice,
		MarkPrice: "1.0100", PositionAmount: "500", LiquidationPrice: "0.865", Leverage: 10, OpenTime: suiOpenTime,
		UpdateTime: suiOpenTime + 1,
	}
}

func protective(orderID string, kind chart.ProtectiveKind, trigger, quantity, positionID string) chart.ProtectiveOrder {
	return chart.ProtectiveOrder{
		OrderID: orderID, PositionID: positionID, Symbol: suiSymbol, Kind: kind,
		TriggerPrice: trigger, Quantity: quantity, Status: "NEW",
	}
}

func fixtureOrders() []chart.ProtectiveOrder {
	return []chart.ProtectiveOrder{
		protective("9200000000000000001", chart.StopLoss, "0.8875", "500", suiPositionID),
		protective("9200000000000000002", chart.TakeProfit, "1.2000", "250", suiPositionID),
		protective("9200000000000000003", chart.TakeProfit, "1.3500", "250", suiPositionID),
	}
}

func candleOf(openTime int64) chart.Candle {
	return chart.Candle{Symbol: suiSymbol, Interval: chart.Interval15m, OpenTime: openTime,
		Open: "1", High: "2", Low: "0.5", Close: "1.5", Volume: "10"}
}

type fixture struct {
	positions *fakePositionSource
	orders    *fakeOrderSource
	source    *fakeCandleSource
	repo      *fakeCandleRepository
	logs      *bytes.Buffer
}

func newFixture() *fixture {
	return &fixture{
		positions: &fakePositionSource{positions: []position.OpenPosition{suiPosition()}},
		orders:    &fakeOrderSource{orders: fixtureOrders()},
		source:    &fakeCandleSource{candles: []chart.Candle{candleOf(candleSourceTimeOne), candleOf(candleSourceTimeOne - 900_000)}},
		repo:      &fakeCandleRepository{},
		logs:      &bytes.Buffer{},
	}
}

func (f *fixture) execute(query appchart.Query) (appchart.PositionChart, error) {
	logger := slog.New(slog.NewTextHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	useCase := appchart.NewGetPositionChartUseCase(f.positions, f.orders, f.source, f.repo, logger)
	return useCase.Execute(context.Background(), query)
}

func defaultQuery() appchart.Query {
	return appchart.Query{PositionID: suiPositionID, Interval: chart.DefaultInterval, Limit: chart.DefaultLimit}
}

func TestExecute_HappyPath(t *testing.T) {
	f := newFixture()

	result, err := f.execute(defaultQuery())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if result.PositionID != suiPositionID || result.Symbol != suiSymbol || result.Side != position.Long ||
		result.Interval != chart.Interval15m || result.FetchedAt != ordersFetchedAt {
		t.Errorf("Kopfdaten falsch: %+v", result)
	}
	if result.Entry.Price != "0.9600" || result.Entry.Time != suiOpenTime {
		t.Errorf("Entry = %+v", result.Entry)
	}
	if len(result.StopLoss) != 1 || result.StopLoss[0].OrderID != "9200000000000000001" ||
		result.StopLoss[0].TriggerPrice != "0.8875" || result.StopLoss[0].Quantity != "500" ||
		result.StopLoss[0].DistanceRatio != "-0.07552083" {
		t.Errorf("StopLoss = %+v", result.StopLoss)
	}
	expectedTakeProfits := []struct{ orderID, trigger, ratio string }{
		{"9200000000000000002", "1.2000", "0.25000000"},
		{"9200000000000000003", "1.3500", "0.40625000"},
	}
	if len(result.TakeProfit) != len(expectedTakeProfits) {
		t.Fatalf("TakeProfit = %+v, erwartet beide Teil-TPs", result.TakeProfit)
	}
	for index, expected := range expectedTakeProfits {
		got := result.TakeProfit[index]
		if got.OrderID != expected.orderID || got.TriggerPrice != expected.trigger || got.DistanceRatio != expected.ratio {
			t.Errorf("TakeProfit[%d] = %+v, erwartet %+v", index, got, expected)
		}
	}
	// Stufe 1: genau ein ENTRY-Marker.
	if len(result.Markers) != 1 || result.Markers[0].Kind != appchart.MarkerEntry ||
		result.Markers[0].Time != suiOpenTime || result.Markers[0].Price != "0.9600" {
		t.Errorf("Markers = %+v", result.Markers)
	}
	// Kerzen kommen aufsteigend aus dem Cache, nicht roh aus der Quelle.
	if len(result.Candles) != 2 || result.Candles[0].OpenTime >= result.Candles[1].OpenTime {
		t.Errorf("Candles nicht aufsteigend: %+v", result.Candles)
	}
}

func TestExecute_KerzenQuelleUndCacheBekommenSymbolIntervallLimit(t *testing.T) {
	f := newFixture()

	if _, err := f.execute(appchart.Query{PositionID: suiPositionID, Interval: chart.Interval4h, Limit: 1000}); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	expected := candleRequest{suiSymbol, chart.Interval4h, 1000}
	if len(f.source.requests) != 1 || f.source.requests[0] != expected {
		t.Errorf("Quelle: %+v, erwartet %+v", f.source.requests, expected)
	}
	if len(f.repo.listRequests) != 1 || f.repo.listRequests[0] != expected {
		t.Errorf("Cache-Lesen: %+v, erwartet %+v", f.repo.listRequests, expected)
	}
}

func TestExecute_LaufendeKerzeWirdAktualisiertNichtDupliziert(t *testing.T) {
	f := newFixture()
	if _, err := f.execute(defaultQuery()); err != nil {
		t.Fatal(err)
	}
	updated := candleOf(candleSourceTimeOne)
	updated.Close = "1.9"
	f.source.candles = []chart.Candle{updated}

	result, err := f.execute(defaultQuery())
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Candles) != 2 {
		t.Fatalf("erwartet 2 Kerzen ohne Duplikat, erhalten %d", len(result.Candles))
	}
	if last := result.Candles[len(result.Candles)-1]; last.Close != "1.9" {
		t.Errorf("laufende Kerze nicht aktualisiert: %+v", last)
	}
}

func TestExecute_OhneSLundTPNurEntryMitLeerenListen(t *testing.T) {
	f := newFixture()
	f.orders.orders = nil

	result, err := f.execute(defaultQuery())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if result.StopLoss == nil || result.TakeProfit == nil || len(result.StopLoss) != 0 || len(result.TakeProfit) != 0 {
		t.Errorf("erwartet leere, nicht-nil Listen: SL=%v TP=%v", result.StopLoss, result.TakeProfit)
	}
	if result.Entry.Price != "0.9600" || len(result.Markers) != 1 {
		t.Errorf("Entry/Marker fehlen: %+v", result)
	}
}

func TestExecute_FremdePositionenWerdenIgnoriert(t *testing.T) {
	f := newFixture()
	f.orders.orders = append(f.orders.orders,
		protective("9200000000000000009", chart.StopLoss, "0.5000", "100", "9100000000000000002"),
		// Gleiche Ziffern bis auf die letzte Stelle: nur exakter String-Vergleich darf passen.
		protective("9200000000000000010", chart.TakeProfit, "2.0000", "100", "9100000000000000000"),
	)

	result, err := f.execute(defaultQuery())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StopLoss) != 1 || len(result.TakeProfit) != 2 {
		t.Errorf("SL=%+v TP=%+v", result.StopLoss, result.TakeProfit)
	}
}

func TestExecute_SHORT_VorzeichenDerAbstaende(t *testing.T) {
	f := newFixture()
	short := suiPosition()
	short.Side = position.Short
	short.AvgOpenPrice = "100"
	f.positions.positions = []position.OpenPosition{short}
	f.orders.orders = []chart.ProtectiveOrder{
		protective("1", chart.StopLoss, "107.5", "500", suiPositionID),
		protective("2", chart.TakeProfit, "90", "500", suiPositionID),
	}

	result, err := f.execute(defaultQuery())
	if err != nil {
		t.Fatal(err)
	}
	if result.StopLoss[0].DistanceRatio != "0.07500000" || result.TakeProfit[0].DistanceRatio != "-0.10000000" {
		t.Errorf("SL=%s TP=%s", result.StopLoss[0].DistanceRatio, result.TakeProfit[0].DistanceRatio)
	}
}

func TestExecute_UnbekanntePositionIstNotFound(t *testing.T) {
	for _, positionID := range []string{"9100000000000000099", "", "0"} {
		t.Run("id "+positionID, func(t *testing.T) {
			f := newFixture()
			query := defaultQuery()
			query.PositionID = positionID

			_, err := f.execute(query)

			var notFound *chart.PositionNotFoundError
			if !errors.As(err, &notFound) || notFound.PositionID != positionID {
				t.Fatalf("erwartet *PositionNotFoundError, erhalten %v", err)
			}
			if f.source.callCount != 0 || f.orders.callCount != 0 {
				t.Error("bei unbekannter Position dürfen weder Kerzen noch Orders abgefragt werden")
			}
		})
	}
}

func TestExecute_FehlerDerAbhaengigkeitenWerdenDurchgereicht(t *testing.T) {
	positionsUnavailable := &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}
	ordersUnavailable := &chart.DataUnavailableError{Reason: position.ReasonRateLimited}
	candlesUnavailable := &chart.DataUnavailableError{Reason: position.ReasonUnreachable}
	storageFailure := errors.New("db kaputt")

	testCases := []struct {
		name         string
		breakFixture func(*fixture)
		check        func(error) bool
	}{
		{"Positionen", func(f *fixture) { f.positions.err = positionsUnavailable },
			func(err error) bool {
				var target *position.OpenPositionsUnavailableError
				return errors.As(err, &target)
			}},
		{"Orders", func(f *fixture) { f.orders.err = ordersUnavailable },
			func(err error) bool {
				var target *chart.DataUnavailableError
				return errors.As(err, &target) && target.Reason == position.ReasonRateLimited
			}},
		{"Kerzenquelle", func(f *fixture) { f.source.err = candlesUnavailable },
			func(err error) bool {
				var target *chart.DataUnavailableError
				return errors.As(err, &target) && target.Reason == position.ReasonUnreachable
			}},
		{"Kerzen-Symbol unbekannt", func(f *fixture) { f.source.err = &chart.SymbolNotFoundError{Symbol: suiSymbol} },
			func(err error) bool { var target *chart.SymbolNotFoundError; return errors.As(err, &target) }},
		{"Cache schreiben", func(f *fixture) { f.repo.upsertErr = storageFailure }, func(err error) bool { return errors.Is(err, storageFailure) }},
		{"Cache lesen", func(f *fixture) { f.repo.listErr = storageFailure }, func(err error) bool { return errors.Is(err, storageFailure) }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			f := newFixture()
			testCase.breakFixture(f)
			if _, err := f.execute(defaultQuery()); err == nil || !testCase.check(err) {
				t.Errorf("unerwarteter Fehler: %v", err)
			}
		})
	}
}

func TestExecute_UngueltigeTriggerPreiseSindFehler(t *testing.T) {
	f := newFixture()
	f.orders.orders = []chart.ProtectiveOrder{protective("1", chart.StopLoss, "0", "500", suiPositionID)}

	_, err := f.execute(defaultQuery())

	var notPositive *chart.TriggerPriceMustBePositiveError
	if !errors.As(err, &notPositive) {
		t.Errorf("erwartet *TriggerPriceMustBePositiveError, erhalten %v", err)
	}
}

func TestExecute_LeereKerzenAntwortBeiOffenerPositionWirdGeloggtNichtIgnoriert(t *testing.T) {
	f := newFixture()
	f.source.candles = nil

	result, err := f.execute(defaultQuery())

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if result.Candles == nil || len(result.Candles) != 0 {
		t.Errorf("erwartet leere, nicht-nil Kerzenliste, erhalten %v", result.Candles)
	}
	logged := f.logs.String()
	if !strings.Contains(logged, "level=WARN") || !strings.Contains(logged, suiSymbol) {
		t.Errorf("erwartet WARN-Log mit Symbol, erhalten %q", logged)
	}
}

func TestExecute_NichtLeereAntwortLoggtKeineWarnung(t *testing.T) {
	f := newFixture()
	if _, err := f.execute(defaultQuery()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.logs.String(), "level=WARN") {
		t.Errorf("unerwartete Warnung: %s", f.logs.String())
	}
}
