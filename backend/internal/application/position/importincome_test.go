package position_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

const (
	dayMillis  = int64(86_400_000)
	hourMillis = int64(3_600_000)
	weekMillis = 7 * dayMillis
)

type incomeCall struct{ from, to int64 }

type incomeSourceFake struct {
	calls     []incomeCall
	byFrom    map[int64][]domainposition.Income
	failOnNth int // 1-basiert, 0 = nie
	err       error
}

func (fake *incomeSourceFake) FetchIncome(_ context.Context, from, to int64) ([]domainposition.Income, error) {
	fake.calls = append(fake.calls, incomeCall{from, to})
	if fake.failOnNth > 0 && len(fake.calls) == fake.failOnNth {
		return nil, fake.err
	}
	return fake.byFrom[from], nil
}

type incomeRepoFake struct {
	batches [][]domainposition.Income
	err     error
}

func (fake *incomeRepoFake) UpsertIncomeBatch(_ context.Context, entries []domainposition.Income) error {
	if fake.err != nil {
		return fake.err
	}
	fake.batches = append(fake.batches, entries)
	return nil
}

type incomeStateFake struct {
	values         map[string]int64
	sets           []string
	getErr, setErr error
}

func newIncomeState() *incomeStateFake { return &incomeStateFake{values: map[string]int64{}} }

func (fake *incomeStateFake) Get(_ context.Context, key string) (int64, bool, error) {
	value, found := fake.values[key]
	return value, found, fake.getErr
}
func (fake *incomeStateFake) Set(_ context.Context, key string, value int64) error {
	if fake.setErr != nil {
		return fake.setErr
	}
	fake.values[key] = value
	fake.sets = append(fake.sets, key)
	return nil
}
func (fake *incomeStateFake) Delete(context.Context, string) error               { return nil }
func (fake *incomeStateFake) ListKeys(context.Context, string) ([]string, error) { return nil, nil }

func income(tranID, symbol, incomeType string, at int64) domainposition.Income {
	return domainposition.Income{TranID: tranID, Symbol: symbol, IncomeType: incomeType, Amount: "1", Time: at, Asset: "USDT"}
}

func runIncome(t *testing.T, source *incomeSourceFake, repo *incomeRepoFake, state *incomeStateFake, fullStart, nowMs int64) (appposition.ImportIncomeResult, error) {
	t.Helper()
	useCase := appposition.NewImportIncomeUseCase(source, repo, state, func() time.Time { return time.UnixMilli(nowMs) })
	return useCase.Execute(context.Background(), fullStart)
}

func TestImportIncome_Fenster(t *testing.T) {
	const start = int64(1_700_000_000_000)
	tests := []struct {
		name string
		now  int64
		want []incomeCall
	}{
		{"genau 7 Tage: ein Fenster", start + weekMillis, []incomeCall{{start, start + weekMillis}}},
		{"7 Tage + 1 ms: zwei Fenster", start + weekMillis + 1, []incomeCall{{start, start + weekMillis}, {start + weekMillis, start + weekMillis + 1}}},
		{"10 Tage", start + 10*dayMillis, []incomeCall{{start, start + weekMillis}, {start + weekMillis, start + 10*dayMillis}}},
		{"14 Tage: genau zwei", start + 2*weekMillis, []incomeCall{{start, start + weekMillis}, {start + weekMillis, start + 2*weekMillis}}},
		{"1 Stunde", start + hourMillis, []incomeCall{{start, start + hourMillis}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, state := &incomeSourceFake{}, newIncomeState()
			if _, err := runIncome(t, source, &incomeRepoFake{}, state, start, tt.now); err != nil {
				t.Fatalf("Fehler: %v", err)
			}
			if !reflect.DeepEqual(source.calls, tt.want) {
				t.Errorf("Fenster %v, erwartet %v", source.calls, tt.want)
			}
			if state.values[appposition.IncomeScanUntilKey] != tt.now {
				t.Errorf("State = %d, erwartet jetzt %d", state.values[appposition.IncomeScanUntilKey], tt.now)
			}
		})
	}
}

func TestImportIncome_StateKeyName(t *testing.T) {
	if appposition.IncomeScanUntilKey != "income_scan_until" {
		t.Errorf("Schlüssel %q, laut schema.md income_scan_until", appposition.IncomeScanUntilKey)
	}
}

func TestImportIncome_StartMitUeberlappungAusState(t *testing.T) {
	const fullStart, saved = int64(1_000_000_000_000), int64(1_700_000_000_000)
	source, state := &incomeSourceFake{}, newIncomeState()
	state.values[appposition.IncomeScanUntilKey] = saved
	now := saved + 2*hourMillis
	if _, err := runIncome(t, source, &incomeRepoFake{}, state, fullStart, now); err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	want := []incomeCall{{saved - hourMillis, now}}
	if !reflect.DeepEqual(source.calls, want) {
		t.Errorf("Fenster %v, erwartet %v (State minus 1 h)", source.calls, want)
	}
	if state.values[appposition.IncomeScanUntilKey] != now {
		t.Errorf("State nicht auf jetzt vorgerückt: %d", state.values[appposition.IncomeScanUntilKey])
	}
}

func TestImportIncome_ErgebnisBatchesDedupSymbole(t *testing.T) {
	const start = int64(1_700_000_000_000)
	source := &incomeSourceFake{byFrom: map[int64][]domainposition.Income{
		start: {
			income("1", "XRP-USDT", "REALIZED_PNL", start+1),
			income("2", "ONDO-USDT", "REALIZED_PNL", start+2),
			income("2", "ONDO-USDT", "REALIZED_PNL", start+2), // gleiche tranId doppelt im Fenster
			income("3", "SUI-USDT", "FUNDING_FEE", start+3),   // kein REALIZED_PNL
			income("4", "XRP-USDT", "REALIZED_PNL", start+4),
		},
		start + weekMillis: {income("5", "ONDO-USDT", "REALIZED_PNL", start+weekMillis+1)},
	}}
	repo := &incomeRepoFake{}
	result, err := runIncome(t, source, repo, newIncomeState(), start, start+8*dayMillis)
	if err != nil {
		t.Fatalf("Fehler: %v", err)
	}
	if len(repo.batches) != 2 {
		t.Fatalf("Batches %d, erwartet 1 pro Fenster = 2", len(repo.batches))
	}
	if len(repo.batches[0]) != 4 {
		t.Errorf("Batch 1 hat %d Einträge, erwartet 4 (tranId-Duplikat entfernt)", len(repo.batches[0]))
	}
	if result.Imported != 5 {
		t.Errorf("Imported %d, erwartet 5 Upserts", result.Imported)
	}
	wantSymbols := []string{"ONDO-USDT", "XRP-USDT"}
	if !reflect.DeepEqual(result.Symbols, wantSymbols) {
		t.Errorf("Symbole %v, erwartet %v (nur REALIZED_PNL, eindeutig, sortiert)", result.Symbols, wantSymbols)
	}
}

func TestImportIncome_LeereHistorieIstKeinFehler(t *testing.T) {
	const start = int64(1_700_000_000_000)
	state := newIncomeState()
	result, err := runIncome(t, &incomeSourceFake{}, &incomeRepoFake{}, state, start, start+hourMillis)
	if err != nil || result.Imported != 0 || len(result.Symbols) != 0 {
		t.Errorf("result=%+v err=%v", result, err)
	}
	if _, found := state.values[appposition.IncomeScanUntilKey]; !found {
		t.Error("State muss auch bei leerer Historie vorrücken")
	}
}

func TestImportIncome_FehlerLassenStateUnveraendert(t *testing.T) {
	const start = int64(1_700_000_000_000)
	boom := errors.New("boom")
	page := map[int64][]domainposition.Income{start: {income("1", "XRP-USDT", "REALIZED_PNL", start+1)}}
	tests := []struct {
		name   string
		source *incomeSourceFake
		repo   *incomeRepoFake
		state  func() *incomeStateFake
	}{
		{"Quelle scheitert im zweiten Fenster", &incomeSourceFake{byFrom: page, failOnNth: 2, err: boom}, &incomeRepoFake{}, newIncomeState},
		{"Quelle scheitert im ersten Fenster", &incomeSourceFake{failOnNth: 1, err: boom}, &incomeRepoFake{}, newIncomeState},
		{"Repository scheitert", &incomeSourceFake{byFrom: page}, &incomeRepoFake{err: boom}, newIncomeState},
		{"State.Get scheitert", &incomeSourceFake{}, &incomeRepoFake{}, func() *incomeStateFake { s := newIncomeState(); s.getErr = boom; return s }},
		{"State.Set scheitert", &incomeSourceFake{}, &incomeRepoFake{}, func() *incomeStateFake { s := newIncomeState(); s.setErr = boom; return s }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.state()
			state.values[appposition.IncomeScanUntilKey] = start + hourMillis
			before := start + hourMillis
			_, err := runIncome(t, tt.source, tt.repo, state, start, start+10*dayMillis)
			if !errors.Is(err, boom) {
				t.Fatalf("err=%v, erwartet boom", err)
			}
			if state.values[appposition.IncomeScanUntilKey] != before {
				t.Errorf("State rückte auf %d vor, muss %d bleiben", state.values[appposition.IncomeScanUntilKey], before)
			}
		})
	}
}

func TestImportIncome_AbgebrochenerKontext(t *testing.T) {
	const start = int64(1_700_000_000_000)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state := newIncomeState()
	useCase := appposition.NewImportIncomeUseCase(&incomeSourceFake{}, &incomeRepoFake{}, state, func() time.Time { return time.UnixMilli(start + dayMillis) })
	_, err := useCase.Execute(ctx, start)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err=%v", err)
	}
	if len(state.sets) != 0 {
		t.Error("State darf nicht geschrieben werden")
	}
}
