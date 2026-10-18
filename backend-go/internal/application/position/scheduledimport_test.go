package position_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
)

const (
	fullStart = int64(1_000)
	nowMillis = int64(1_800_000_000_000)
	oneHourMs = int64(time.Hour / time.Millisecond)
)

var errBoom = errors.New("boom")

type callLog struct{ entries []string }

type fakeIncomeImporter struct {
	log       *callLog
	err       error
	gotStart  int64
	resultSym []string
}

func (fake *fakeIncomeImporter) Execute(_ context.Context, start int64) (appposition.ImportIncomeResult, error) {
	fake.log.entries = append(fake.log.entries, "income")
	fake.gotStart = start
	return appposition.ImportIncomeResult{Symbols: fake.resultSym}, fake.err
}

type fakeSymbolLister struct {
	log     *callLog
	symbols []string
	err     error
}

func (fake *fakeSymbolLister) ListIncomeSymbols(context.Context) ([]string, error) {
	fake.log.entries = append(fake.log.entries, "symbols")
	return fake.symbols, fake.err
}

type fakeContracts struct {
	symbols []string
	err     error
}

func (fake *fakeContracts) FetchContracts(context.Context) ([]string, error) {
	return fake.symbols, fake.err
}

type positionCall struct {
	symbol   string
	from, to int64
}

type fakePositionImporter struct {
	log        *callLog
	calls      []positionCall
	errs       map[string]error
	suspicious map[string]bool
	onCall     func(symbol string)
}

func (fake *fakePositionImporter) Execute(_ context.Context, symbol string, from, to int64) (appposition.ImportClosedPositionsResult, error) {
	fake.log.entries = append(fake.log.entries, "positions:"+symbol)
	fake.calls = append(fake.calls, positionCall{symbol, from, to})
	if fake.onCall != nil {
		fake.onCall(symbol)
	}
	if err := fake.errs[symbol]; err != nil {
		return appposition.ImportClosedPositionsResult{}, err
	}
	result := appposition.ImportClosedPositionsResult{Imported: 1}
	if fake.suspicious[symbol] {
		result.Imported = 0
		result.Suspicious = []appposition.ImportWindowResult{{Symbol: symbol, From: from, To: to, Suspicious: true}}
	}
	return result, nil
}

type memoryState struct {
	values map[string]int64
	setErr error
}

func (state *memoryState) Get(_ context.Context, key string) (int64, bool, error) {
	value, found := state.values[key]
	return value, found, nil
}
func (state *memoryState) Set(_ context.Context, key string, value int64) error {
	if state.setErr != nil {
		return state.setErr
	}
	state.values[key] = value
	return nil
}
func (state *memoryState) Delete(_ context.Context, key string) error {
	delete(state.values, key)
	return nil
}
func (state *memoryState) ListKeys(context.Context, string) ([]string, error) { return nil, nil }

type scheduledFixture struct {
	log       *callLog
	income    *fakeIncomeImporter
	lister    *fakeSymbolLister
	contracts *fakeContracts
	positions *fakePositionImporter
	state     *memoryState
	useCase   *appposition.ScheduledImportUseCase
}

func newScheduledFixture(dbSymbols, contractSymbols []string) *scheduledFixture {
	log := &callLog{}
	fixture := &scheduledFixture{
		log:       log,
		income:    &fakeIncomeImporter{log: log},
		lister:    &fakeSymbolLister{log: log, symbols: dbSymbols},
		contracts: &fakeContracts{symbols: contractSymbols},
		positions: &fakePositionImporter{log: log},
		state:     &memoryState{values: map[string]int64{}},
	}
	fixture.useCase = appposition.NewScheduledImportUseCase(appposition.ScheduledImportDeps{
		Income: fixture.income, Positions: fixture.positions, Symbols: fixture.lister, Contracts: fixture.contracts,
		State: fixture.state, Now: func() time.Time { return time.UnixMilli(nowMillis) }, FullImportStart: fullStart,
	})
	return fixture
}

func TestScheduledIncremental_ReihenfolgeIncomeDannSymboleAusDbDannPositionen(t *testing.T) {
	fixture := newScheduledFixture([]string{"BTC-USDT", "ONDO-USDT"}, nil)
	fixture.income.resultSym = []string{"NUR-IM-LAUF-USDT"} // darf NICHT verwendet werden

	result, err := fixture.useCase.RunIncremental(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	want := []string{"income", "symbols", "positions:BTC-USDT", "positions:ONDO-USDT"}
	if !reflect.DeepEqual(fixture.log.entries, want) {
		t.Errorf("Reihenfolge = %v, erwartet %v", fixture.log.entries, want)
	}
	if fixture.income.gotStart != fullStart {
		t.Errorf("Income-Start = %d, erwartet %d", fixture.income.gotStart, fullStart)
	}
	if !reflect.DeepEqual(result.Succeeded, []string{"BTC-USDT", "ONDO-USDT"}) {
		t.Errorf("Succeeded = %v", result.Succeeded)
	}
}

func TestScheduledIncremental_ErsterLaufVonFullStartBisJetztUndStateNachErfolg(t *testing.T) {
	fixture := newScheduledFixture([]string{"BTC-USDT"}, nil)

	if _, err := fixture.useCase.RunIncremental(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fixture.positions.calls) == 0 {
		t.Fatal("kein Positions-Aufruf")
	}
	if want := (positionCall{"BTC-USDT", fullStart, nowMillis}); fixture.positions.calls[0] != want {
		t.Errorf("Aufruf = %+v, erwartet %+v", fixture.positions.calls[0], want)
	}
	if got := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"BTC-USDT"]; got != nowMillis {
		t.Errorf("State = %d, erwartet %d", got, nowMillis)
	}
}

func TestScheduledIncremental_FolgelaufStartetMitEinerStundeUeberlappung(t *testing.T) {
	fixture := newScheduledFixture([]string{"BTC-USDT"}, nil)
	fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"BTC-USDT"] = nowMillis - 5*oneHourMs

	if _, err := fixture.useCase.RunIncremental(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fixture.positions.calls) == 0 {
		t.Fatal("kein Positions-Aufruf")
	}
	if want := (positionCall{"BTC-USDT", nowMillis - 6*oneHourMs, nowMillis}); fixture.positions.calls[0] != want {
		t.Errorf("Aufruf = %+v, erwartet %+v", fixture.positions.calls[0], want)
	}
}

func TestScheduledIncremental_StateGiltProSymbol(t *testing.T) {
	fixture := newScheduledFixture([]string{"A-USDT", "B-USDT"}, nil)
	fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"A-USDT"] = nowMillis - oneHourMs

	if _, err := fixture.useCase.RunIncremental(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(fixture.positions.calls) != 2 {
		t.Fatalf("Aufrufe = %+v", fixture.positions.calls)
	}
	if fixture.positions.calls[0].from != nowMillis-2*oneHourMs || fixture.positions.calls[1].from != fullStart {
		t.Errorf("Aufrufe = %+v", fixture.positions.calls)
	}
}

func TestScheduledIncremental_SuspiciousFensterWerdenNichtAlsErledigtMarkiert(t *testing.T) {
	fixture := newScheduledFixture([]string{"GOOD-USDT", "SUS-USDT"}, nil)
	fixture.positions.suspicious = map[string]bool{"SUS-USDT": true}

	result, err := fixture.useCase.RunIncremental(context.Background())

	if err != nil {
		t.Fatalf("Verdacht ist kein Fehler: %v", err)
	}
	if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"SUS-USDT"]; set {
		t.Error("State für verdächtiges Symbol darf nicht gesetzt werden")
	}
	if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"GOOD-USDT"]; !set {
		t.Error("State für erfolgreiches Symbol fehlt")
	}
	if !reflect.DeepEqual(result.Suspicious, []string{"SUS-USDT"}) || !reflect.DeepEqual(result.Succeeded, []string{"GOOD-USDT"}) {
		t.Errorf("Ergebnis = %+v", result)
	}
}

func TestScheduledIncremental_FehlerBeiEinemSymbolBrichtAndereNichtAb(t *testing.T) {
	fixture := newScheduledFixture([]string{"A-USDT", "BAD-USDT", "C-USDT"}, nil)
	fixture.positions.errs = map[string]error{"BAD-USDT": errBoom}

	result, err := fixture.useCase.RunIncremental(context.Background())

	var failures *appposition.ImportFailuresError
	if !errors.As(err, &failures) {
		t.Fatalf("err = %v, erwartet *ImportFailuresError", err)
	}
	if len(failures.Failures) != 1 || failures.Failures[0].Symbol != "BAD-USDT" || !errors.Is(failures.Failures[0].Err, errBoom) {
		t.Errorf("Failures = %+v", failures.Failures)
	}
	if !errors.Is(err, errBoom) {
		t.Error("errors.Is auf die Ursache muss über ImportFailuresError funktionieren")
	}
	if len(fixture.positions.calls) != 3 || fixture.positions.calls[2].symbol != "C-USDT" {
		t.Errorf("C-USDT wurde nicht mehr verarbeitet: %+v", fixture.positions.calls)
	}
	if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"BAD-USDT"]; set {
		t.Error("State des fehlgeschlagenen Symbols darf nicht gesetzt werden")
	}
	for _, symbol := range []string{"A-USDT", "C-USDT"} {
		if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+symbol]; !set {
			t.Errorf("State für %s fehlt", symbol)
		}
	}
	if len(result.Failures) != 1 {
		t.Errorf("result.Failures = %+v", result.Failures)
	}
}

func TestScheduledIncremental_IncomeFehlerStopptVorDenPositionen(t *testing.T) {
	fixture := newScheduledFixture([]string{"BTC-USDT"}, nil)
	fixture.income.err = errBoom

	_, err := fixture.useCase.RunIncremental(context.Background())

	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v", err)
	}
	if len(fixture.positions.calls) != 0 || len(fixture.state.values) != 0 {
		t.Errorf("keine Positionen/State erwartet: %v %v", fixture.positions.calls, fixture.state.values)
	}
}

func TestScheduledIncremental_SymbolListenFehler(t *testing.T) {
	fixture := newScheduledFixture(nil, nil)
	fixture.lister.err = errBoom

	_, err := fixture.useCase.RunIncremental(context.Background())

	if !errors.Is(err, errBoom) || len(fixture.positions.calls) != 0 {
		t.Errorf("err = %v, Aufrufe = %v", err, fixture.positions.calls)
	}
}

func TestScheduledIncremental_LeereSymbolListeIstOk(t *testing.T) {
	fixture := newScheduledFixture(nil, nil)

	result, err := fixture.useCase.RunIncremental(context.Background())

	if err != nil || len(result.Symbols) != 0 || len(fixture.positions.calls) != 0 {
		t.Errorf("err = %v, result = %+v", err, result)
	}
}

func TestScheduledIncremental_StateSetFehlerWirdGemeldet(t *testing.T) {
	fixture := newScheduledFixture([]string{"BTC-USDT"}, nil)
	fixture.state.setErr = errBoom

	result, err := fixture.useCase.RunIncremental(context.Background())

	var failures *appposition.ImportFailuresError
	if !errors.As(err, &failures) || !errors.Is(err, errBoom) {
		t.Fatalf("err = %v", err)
	}
	if len(result.Succeeded) != 0 {
		t.Errorf("Succeeded = %v, erwartet leer", result.Succeeded)
	}
}

func TestScheduledIncremental_AbgebrochenerKontextStopptSofort(t *testing.T) {
	fixture := newScheduledFixture([]string{"A-USDT", "B-USDT"}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	fixture.positions.onCall = func(string) { cancel() }

	_, err := fixture.useCase.RunIncremental(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, erwartet context.Canceled", err)
	}
	if len(fixture.positions.calls) != 1 {
		t.Errorf("Aufrufe = %d, erwartet 1", len(fixture.positions.calls))
	}
	if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+"A-USDT"]; !set {
		t.Error("bereits erfolgreiches Symbol behält seinen State")
	}
}

func TestScheduledFullSearch_AlleKontrakteVonFullStartMitStatePerSymbol(t *testing.T) {
	fixture := newScheduledFixture(nil, []string{"A-USDT", "B-USDT"})

	result, err := fixture.useCase.RunFullSearch(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []positionCall{{"A-USDT", fullStart, nowMillis}, {"B-USDT", fullStart, nowMillis}}
	if !reflect.DeepEqual(fixture.positions.calls, wantCalls) {
		t.Errorf("Aufrufe = %+v", fixture.positions.calls)
	}
	for _, symbol := range []string{"A-USDT", "B-USDT"} {
		if fixture.state.values[appposition.PositionsFullSearchKeyPrefix+symbol] != nowMillis {
			t.Errorf("Vollsuche-State für %s fehlt", symbol)
		}
		if fixture.state.values[appposition.PositionsScanUntilKeyPrefix+symbol] != nowMillis {
			t.Errorf("inkrementeller State für %s fehlt", symbol)
		}
	}
	if len(fixture.income.gotStartCalls()) != 0 {
		t.Error("Vollsuche importiert kein Income")
	}
	_ = result
}

func (fake *fakeIncomeImporter) gotStartCalls() []string {
	return slices.DeleteFunc(slices.Clone(fake.log.entries), func(entry string) bool { return entry != "income" })
}

func TestScheduledFullSearch_ErledigteSymbolePausenUndRestWirdFortgesetzt(t *testing.T) {
	fixture := newScheduledFixture(nil, []string{"A-USDT", "B-USDT", "C-USDT"})
	fixture.state.values[appposition.PositionsFullSearchKeyPrefix+"A-USDT"] = nowMillis - 1

	if _, err := fixture.useCase.RunFullSearch(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, call := range fixture.positions.calls {
		got = append(got, call.symbol)
	}
	if !reflect.DeepEqual(got, []string{"B-USDT", "C-USDT"}) {
		t.Errorf("verarbeitet = %v, erwartet [B-USDT C-USDT]", got)
	}
}

func TestScheduledFullSearch_AbbruchSetztSpaeterFort(t *testing.T) {
	fixture := newScheduledFixture(nil, []string{"A-USDT", "B-USDT", "C-USDT"})
	ctx, cancel := context.WithCancel(context.Background())
	fixture.positions.onCall = func(symbol string) {
		if symbol == "B-USDT" {
			cancel()
		}
	}
	_, err := fixture.useCase.RunFullSearch(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	fixture.positions.calls = nil
	fixture.positions.onCall = nil

	if _, err := fixture.useCase.RunFullSearch(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, call := range fixture.positions.calls {
		got = append(got, call.symbol)
	}
	// B wurde abgebrochen (Kontext tot) -> nicht erledigt, daher erneut; A bleibt erledigt, C neu.
	if !reflect.DeepEqual(got, []string{"B-USDT", "C-USDT"}) {
		t.Errorf("Fortsetzung = %v, erwartet [B-USDT C-USDT]", got)
	}
}

func TestScheduledFullSearch_FehlerUndVerdachtMarkierenNichtAlsErledigt(t *testing.T) {
	fixture := newScheduledFixture(nil, []string{"BAD-USDT", "OK-USDT", "SUS-USDT"})
	fixture.positions.errs = map[string]error{"BAD-USDT": errBoom}
	fixture.positions.suspicious = map[string]bool{"SUS-USDT": true}

	result, err := fixture.useCase.RunFullSearch(context.Background())

	var failures *appposition.ImportFailuresError
	if !errors.As(err, &failures) || len(failures.Failures) != 1 || failures.Failures[0].Symbol != "BAD-USDT" {
		t.Fatalf("err = %v", err)
	}
	if len(fixture.positions.calls) != 3 {
		t.Errorf("alle drei Symbole müssen versucht werden: %+v", fixture.positions.calls)
	}
	for _, symbol := range []string{"BAD-USDT", "SUS-USDT"} {
		if _, done := fixture.state.values[appposition.PositionsFullSearchKeyPrefix+symbol]; done {
			t.Errorf("%s darf nicht als erledigt gelten", symbol)
		}
		if _, set := fixture.state.values[appposition.PositionsScanUntilKeyPrefix+symbol]; set {
			t.Errorf("%s: inkrementeller State darf nicht gesetzt sein", symbol)
		}
	}
	if _, done := fixture.state.values[appposition.PositionsFullSearchKeyPrefix+"OK-USDT"]; !done {
		t.Error("OK-USDT muss erledigt sein")
	}
	if !reflect.DeepEqual(result.Suspicious, []string{"SUS-USDT"}) {
		t.Errorf("Suspicious = %v", result.Suspicious)
	}
}

func TestScheduledFullSearch_KontraktFehlerOhneAufrufe(t *testing.T) {
	fixture := newScheduledFixture(nil, nil)
	fixture.contracts.err = errBoom

	_, err := fixture.useCase.RunFullSearch(context.Background())

	if !errors.Is(err, errBoom) || len(fixture.positions.calls) != 0 {
		t.Errorf("err = %v, Aufrufe = %v", err, fixture.positions.calls)
	}
}
