package position_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

// Helpers für Test-Fakes.

type fakeClosedPositionSource struct {
	calls []struct {
		symbol string
		from   int64
		to     int64
	}
	responses map[string]map[string][]domainposition.ClosedPosition
	errors    map[string]map[string]error
}

// Stelle sicher, dass fakeClosedPositionSource interfaces.ClosedPositionSource implementiert.
var _ interfaces.ClosedPositionSource = (*fakeClosedPositionSource)(nil)

func (f *fakeClosedPositionSource) FetchClosedPositions(ctx context.Context, symbol string, from, to int64) ([]domainposition.ClosedPosition, error) {
	f.calls = append(f.calls, struct {
		symbol string
		from   int64
		to     int64
	}{symbol, from, to})

	key := keyForRange(from, to)
	if f.errors != nil && f.errors[symbol] != nil && f.errors[symbol][key] != nil {
		return nil, f.errors[symbol][key]
	}
	if f.responses != nil && f.responses[symbol] != nil {
		return f.responses[symbol][key], nil
	}
	return nil, nil
}

type fakePositionRepository struct {
	upserted []domainposition.ClosedPosition
	errors   map[string]error
}

// Stelle sicher, dass fakePositionRepository interfaces.PositionRepository implementiert.
var _ interfaces.PositionRepository = (*fakePositionRepository)(nil)

func (f *fakePositionRepository) UpsertClosedPosition(ctx context.Context, p domainposition.ClosedPosition) error {
	if f.errors != nil && f.errors[p.PositionID] != nil {
		return f.errors[p.PositionID]
	}
	f.upserted = append(f.upserted, p)
	return nil
}

type fakeIncomeLookup struct {
	alwaysTrue bool                       // true für JEDEN Zeitraum
	data       map[string]map[string]bool // symbol -> (key -> hasIncome)
	errors     map[string]map[string]error
	calls      []struct {
		symbol string
		from   int64
		to     int64
	} // Track IncomeLookup calls for verification.
}

// Stelle sicher, dass fakeIncomeLookup interfaces.IncomeLookup implementiert.
var _ interfaces.IncomeLookup = (*fakeIncomeLookup)(nil)

func (f *fakeIncomeLookup) HasIncomeForSymbol(ctx context.Context, symbol string, from, to int64) (bool, error) {
	f.calls = append(f.calls, struct {
		symbol string
		from   int64
		to     int64
	}{symbol, from, to})

	key := keyForRange(from, to)
	if f.errors != nil && f.errors[symbol] != nil && f.errors[symbol][key] != nil {
		return false, f.errors[symbol][key]
	}
	if f.alwaysTrue {
		return true, nil
	}
	if f.data != nil && f.data[symbol] != nil {
		return f.data[symbol][key], nil
	}
	return false, nil
}

func keyForRange(from, to int64) string {
	return fmt.Sprintf("%d-%d", from, to)
}

// Tests

func TestImportClosedPositions_HappyPath_SingleWindow(t *testing.T) {
	ctx := context.Background()

	// Ein Zeitraum von 50 Tagen (< 89), darum ein Fenster.
	from := int64(1780000000000)
	to := int64(1780000000000 + 50*24*60*60*1000)

	position := domainposition.ClosedPosition{
		PositionID:     "9000000000000000000",
		Symbol:         "ONDO-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 1000,
		CloseTime:      from + 2000,
		AvgOpenPrice:   "0.4120",
		AvgClosePrice:  "0.4188",
		RealisedProfit: "1.7000",
		NetProfit:      "1.5123",
		PositionAmount: "250",
		ClosedAmount:   "250",
		Leverage:       7,
		Raw:            `{}`,
	}

	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"ONDO-USDT": {
				keyForRange(from, to): {position},
			},
		},
	}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{
		data: map[string]map[string]bool{
			"ONDO-USDT": {
				keyForRange(from, to): false,
			},
		},
	}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "ONDO-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported = %d, erwartet 1", result.Imported)
	}
	if len(result.Suspicious) != 0 {
		t.Errorf("Suspicious.len = %d, erwartet 0", len(result.Suspicious))
	}
	if len(repository.upserted) != 1 {
		t.Errorf("upserted.len = %d, erwartet 1", len(repository.upserted))
	}
	if repository.upserted[0].PositionID != "9000000000000000000" {
		t.Errorf("upserted[0].PositionID = %s, erwartet 9000000000000000000", repository.upserted[0].PositionID)
	}
}

func TestImportClosedPositions_MultipleWindows_180Days(t *testing.T) {
	ctx := context.Background()

	// 180 Tage = 3 Fenster: 89 + 89 + 2 Tage
	from := int64(1780000000000)
	to := from + 180*24*60*60*1000

	pos1 := domainposition.ClosedPosition{
		PositionID:     "9000000000000000001",
		Symbol:         "BTC-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 1000,
		CloseTime:      from + 2000,
		AvgOpenPrice:   "50000",
		AvgClosePrice:  "51000",
		RealisedProfit: "100",
		NetProfit:      "95",
		PositionAmount: "1",
		ClosedAmount:   "1",
		Leverage:       10,
		Raw:            `{}`,
	}

	pos2 := domainposition.ClosedPosition{
		PositionID:     "9000000000000000002",
		Symbol:         "BTC-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 90*24*60*60*1000 + 1000,
		CloseTime:      from + 90*24*60*60*1000 + 2000,
		AvgOpenPrice:   "51000",
		AvgClosePrice:  "52000",
		RealisedProfit: "150",
		NetProfit:      "145",
		PositionAmount: "1",
		ClosedAmount:   "1",
		Leverage:       10,
		Raw:            `{}`,
	}

	pos3 := domainposition.ClosedPosition{
		PositionID:     "9000000000000000003",
		Symbol:         "BTC-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 178*24*60*60*1000 + 1000,
		CloseTime:      from + 178*24*60*60*1000 + 2000,
		AvgOpenPrice:   "52000",
		AvgClosePrice:  "52500",
		RealisedProfit: "50",
		NetProfit:      "45",
		PositionAmount: "1",
		ClosedAmount:   "1",
		Leverage:       10,
		Raw:            `{}`,
	}

	window1End := from + 89*24*60*60*1000
	window2End := from + 178*24*60*60*1000
	window3End := to

	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"BTC-USDT": {
				keyForRange(from, window1End):       {pos1},
				keyForRange(window1End, window2End): {pos2},
				keyForRange(window2End, window3End): {pos3},
			},
		},
	}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{
		data: map[string]map[string]bool{
			"BTC-USDT": {
				keyForRange(from, window1End):       false,
				keyForRange(window1End, window2End): false,
				keyForRange(window2End, window3End): false,
			},
		},
	}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "BTC-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if result.Imported != 3 {
		t.Errorf("Imported = %d, erwartet 3 (1 pro Fenster)", result.Imported)
	}
	if len(repository.upserted) != 3 {
		t.Errorf("upserted.len = %d, erwartet 3", len(repository.upserted))
	}
}

func TestImportClosedPositions_SuspiciousEmptyImport(t *testing.T) {
	// Semantik: Der gesamte Lauf ist verdächtig wenn:
	// - 0 Positionen über ALLE Fenster importiert wurden UND
	// - Income für das Symbol im GESAMTZEITRAUM (from-to) vorhanden ist.
	// IncomeLookup wird EINMAL mit Gesamtzeitraum aufgerufen, nicht pro Fenster.
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 180*24*60*60*1000 // 3 Fenster, alle leer

	// Source liefert für alle Fenster nichts (leere Antworten).
	source := &fakeClosedPositionSource{}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{
		data: map[string]map[string]bool{
			"XRP-USDT": {
				keyForRange(from, to): true, // Income nur im Gesamtzeitraum-Key
			},
		},
	}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "XRP-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if result.Imported != 0 {
		t.Errorf("Imported = %d, erwartet 0", result.Imported)
	}

	// Lauf sollte als verdächtig markiert werden.
	if len(result.Suspicious) != 1 {
		t.Fatalf("Suspicious.len = %d, erwartet 1 (Gesamtzeitraum, nicht pro Fenster)", len(result.Suspicious))
	}
	if result.Suspicious[0].From != from || result.Suspicious[0].To != to {
		t.Errorf("Suspicious[0] = (%d, %d), erwartet Gesamtzeitraum (%d, %d)",
			result.Suspicious[0].From, result.Suspicious[0].To, from, to)
	}
	if len(source.calls) != 3 {
		t.Errorf("source.calls.len = %d, erwartet 3 Fenster", len(source.calls))
	}

	// Verifiziere: IncomeLookup wurde genau einmal mit Gesamtzeitraum aufgerufen.
	if len(incomeLookup.calls) != 1 {
		t.Errorf("incomeLookup.calls.len = %d, erwartet 1 (einmaliger Aufruf mit Gesamtzeitraum)", len(incomeLookup.calls))
	}
	if len(incomeLookup.calls) > 0 {
		call := incomeLookup.calls[0]
		if call.symbol != "XRP-USDT" || call.from != from || call.to != to {
			t.Errorf("incomeLookup.calls[0] = (%s, %d, %d), erwartet (XRP-USDT, %d, %d)", call.symbol, call.from, call.to, from, to)
		}
	}
}

func TestImportClosedPositions_NotSuspicious_PartialPositions(t *testing.T) {
	// Wenn in mindestens einem Fenster Positionen importiert wurden, ist der Lauf NICHT verdächtig,
	// auch wenn andere Fenster leer sind.
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 180*24*60*60*1000 // 180 Tage = 3 Fenster

	window1End := from + 89*24*60*60*1000
	window2End := from + 178*24*60*60*1000
	window3End := to

	pos1 := domainposition.ClosedPosition{
		PositionID:     "9000000000040000001",
		Symbol:         "SOL-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 1000,
		CloseTime:      from + 2000,
		AvgOpenPrice:   "100",
		AvgClosePrice:  "110",
		RealisedProfit: "10",
		NetProfit:      "8",
		PositionAmount: "1",
		ClosedAmount:   "1",
		Leverage:       5,
		Raw:            `{}`,
	}

	// Fenster 1: 1 Position
	// Fenster 2: leer
	// Fenster 3: leer
	// Gesamt: 1 Position → NICHT verdächtig (trotz leerer Fenster 2 und 3)
	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"SOL-USDT": {
				keyForRange(from, window1End):       {pos1},
				keyForRange(window1End, window2End): {},
				keyForRange(window2End, window3End): {},
			},
		},
	}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{alwaysTrue: true} // Income für JEDEN Zeitraum

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "SOL-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Imported = %d, erwartet 1", result.Imported)
	}

	// NICHT verdächtig: es wurden Positionen importiert (mindestens eine).
	if len(result.Suspicious) != 0 {
		t.Errorf("Suspicious.len = %d, erwartet 0 (leere Fenster bei Positionen in anderen ist OK)", len(result.Suspicious))
	}

	// IncomeLookup nie mit einem Teilfenster aufrufen (höchstens Gesamtzeitraum).
	for _, call := range incomeLookup.calls {
		if call.from != from || call.to != to {
			t.Errorf("incomeLookup mit Teilfenster (%d, %d) aufgerufen, erlaubt nur (%d, %d)", call.from, call.to, from, to)
		}
	}
}

func TestImportClosedPositions_SourceError_BreaksImport(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 180*24*60*60*1000

	window1End := from + 89*24*60*60*1000

	source := &fakeClosedPositionSource{
		errors: map[string]map[string]error{
			"ETH-USDT": {
				keyForRange(from, window1End): errors.New("network error"),
			},
		},
	}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "ETH-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
	if result.Imported != 0 {
		t.Errorf("Nach Fehler sollte Imported = 0, bekam %d", result.Imported)
	}
	if !errors.Is(err, errors.New("network error")) && err.Error() != "network error" {
		t.Errorf("Fehler = %v, erwartet 'network error'", err)
	}
}

func TestImportClosedPositions_RepositoryError_BreaksImport(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	position := domainposition.ClosedPosition{
		PositionID:     "9000000000000000000",
		Symbol:         "SOL-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 1000,
		CloseTime:      from + 2000,
		AvgOpenPrice:   "100",
		AvgClosePrice:  "105",
		RealisedProfit: "10",
		NetProfit:      "8",
		PositionAmount: "1",
		ClosedAmount:   "1",
		Leverage:       5,
		Raw:            `{}`,
	}

	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"SOL-USDT": {
				keyForRange(from, to): {position},
			},
		},
	}
	repository := &fakePositionRepository{
		errors: map[string]error{
			"9000000000000000000": errors.New("db error"),
		},
	}
	incomeLookup := &fakeIncomeLookup{}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "SOL-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
	if result.Imported != 0 {
		t.Errorf("Nach Fehler sollte Imported = 0, bekam %d", result.Imported)
	}
}

func TestImportClosedPositions_IncomeLookupError_BreaksImport(t *testing.T) {
	ctx := context.Background()

	from := int64(1780000000000)
	to := from + 50*24*60*60*1000

	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"ADA-USDT": {
				keyForRange(from, to): {},
			},
		},
	}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{
		errors: map[string]map[string]error{
			"ADA-USDT": {
				keyForRange(from, to): errors.New("lookup error"),
			},
		},
	}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	result, err := useCase.Execute(ctx, "ADA-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
	if result.Imported != 0 {
		t.Errorf("Nach Fehler sollte Imported = 0, bekam %d", result.Imported)
	}
}

func TestImportClosedPositions_ContextCancelled_BetweenWindows(t *testing.T) {
	// Context wird WÄHREND des Imports abgebrochen (bevor 2. Fenster abgerufen wird).
	// → Error mit context.Canceled oder ähnlich, result.Imported enthält Positionen bis zum Abbruch.
	from := int64(1780000000000)
	to := from + 180*24*60*60*1000 // 3 Fenster

	window1End := from + 89*24*60*60*1000
	window2End := from + 178*24*60*60*1000

	pos1 := domainposition.ClosedPosition{
		PositionID:     "9000000000050000001",
		Symbol:         "DOGE-USDT",
		Side:           domainposition.Long,
		OpenTime:       from + 1000,
		CloseTime:      from + 2000,
		AvgOpenPrice:   "0.15",
		AvgClosePrice:  "0.20",
		RealisedProfit: "5",
		NetProfit:      "4",
		PositionAmount: "100",
		ClosedAmount:   "100",
		Leverage:       2,
		Raw:            `{}`,
	}

	// Source mit Fehler bei 2. Fenster (simuliert Context-Abbruch).
	source := &fakeClosedPositionSource{
		responses: map[string]map[string][]domainposition.ClosedPosition{
			"DOGE-USDT": {
				keyForRange(from, window1End): {pos1},
			},
		},
		errors: map[string]map[string]error{
			"DOGE-USDT": {
				keyForRange(window1End, window2End): context.Canceled,
			},
		},
	}

	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{}

	// Use-Case führt aus und bricht bei 2. Fenster ab.
	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	ctx := context.Background()
	result, err := useCase.Execute(ctx, "DOGE-USDT", from, to)

	if err == nil {
		t.Fatalf("Fehler erwartet für abgebrochenen Import")
	}

	// Fehler sollte context.Canceled oder ähnlich sein.
	if !errors.Is(err, context.Canceled) {
		// Auch OK wenn Fehler durchgereicht wird.
		t.Logf("Fehler beim Abbruch: %v", err)
	}

	// Imported sollte = 1 sein (nur das 1. Fenster, dann Abbruch).
	if result.Imported != 1 {
		t.Errorf("result.Imported = %d, erwartet 1 (abgebrochen nach Fenster 1)", result.Imported)
	}
}

func TestImportClosedPositions_NoWindowingNeeded_SingleSmallRange(t *testing.T) {
	ctx := context.Background()

	// 10 Tage, kein Windowing nötig.
	from := int64(1780000000000)
	to := from + 10*24*60*60*1000

	source := &fakeClosedPositionSource{}
	repository := &fakePositionRepository{}
	incomeLookup := &fakeIncomeLookup{}

	useCase := appposition.NewImportClosedPositionsUseCase(source, repository, incomeLookup)
	_, err := useCase.Execute(ctx, "TEST-USDT", from, to)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	// Sollte genau einen Aufruf an Source machen, nicht mehrere.
	if len(source.calls) != 1 {
		t.Errorf("source.calls.len = %d, erwartet 1", len(source.calls))
	}
	if source.calls[0].from != from || source.calls[0].to != to {
		t.Errorf("source.calls[0] = (%d, %d), erwartet (%d, %d)",
			source.calls[0].from, source.calls[0].to, from, to)
	}
}
