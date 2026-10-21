package position_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func backfill(t *testing.T, source *incomeSourceFake, repo *incomeRepoFake, state *incomeStateFake, since, nowMs int64) error {
	t.Helper()
	useCase := appposition.NewImportIncomeUseCase(source, repo, state, func() time.Time { return time.UnixMilli(nowMs) })
	return useCase.EnsureHistorySince(context.Background(), since)
}

func TestEnsureHistorySince_LaedtFensterBisJetztUndMerktSich(t *testing.T) {
	since := 100 * dayMillis
	nowMs := since + 10*dayMillis
	source := &incomeSourceFake{byFrom: map[int64][]domainposition.Income{
		since: {income("1", "BTC-USDT", domainposition.IncomeTypeRealizedPnl, since+hourMillis)},
	}}
	repo := &incomeRepoFake{}
	state := newIncomeState()

	if err := backfill(t, source, repo, state, since, nowMs); err != nil {
		t.Fatalf("Fehler: %v", err)
	}

	wantCalls := []incomeCall{{since, since + weekMillis}, {since + weekMillis, nowMs}}
	if len(source.calls) != 2 || source.calls[0] != wantCalls[0] || source.calls[1] != wantCalls[1] {
		t.Fatalf("Fenster = %v, erwartet %v", source.calls, wantCalls)
	}
	if len(repo.batches) != 2 || len(repo.batches[0]) != 1 {
		t.Fatalf("Batches = %v", repo.batches)
	}
	if state.values[appposition.IncomeHistoryFromKey] != since {
		t.Fatalf("State = %v, erwartet %d", state.values, since)
	}
	if _, touched := state.values[appposition.IncomeScanUntilKey]; touched {
		t.Fatal("der Fortschritt des laufenden Imports darf nicht verändert werden")
	}
}

func TestEnsureHistorySince_TutNichtsWennSchonGeladen(t *testing.T) {
	since := 100 * dayMillis
	state := newIncomeState()
	state.values[appposition.IncomeHistoryFromKey] = since - dayMillis
	source := &incomeSourceFake{}

	if err := backfill(t, source, &incomeRepoFake{}, state, since, since+dayMillis); err != nil {
		t.Fatalf("Fehler: %v", err)
	}

	if len(source.calls) != 0 {
		t.Fatalf("unerwartete Abrufe: %v", source.calls)
	}
}

func TestEnsureHistorySince_LaedtNachWennGespeicherterWertSpaeterAlsSince(t *testing.T) {
	since := 100 * dayMillis
	nowMs := since + 10*dayMillis
	state := newIncomeState()
	state.values[appposition.IncomeHistoryFromKey] = since + 5*dayMillis
	source := &incomeSourceFake{}

	if err := backfill(t, source, &incomeRepoFake{}, state, since, nowMs); err != nil {
		t.Fatalf("Fehler: %v", err)
	}

	if len(source.calls) == 0 || source.calls[0].from != since {
		t.Fatalf("Abrufe = %v, erwartet Nachladen ab %d", source.calls, since)
	}
	if state.values[appposition.IncomeHistoryFromKey] != since {
		t.Fatalf("State = %v, erwartet %d", state.values, since)
	}
}

func TestEnsureHistorySince_MerktSichNichtBeiFehler(t *testing.T) {
	since := 100 * dayMillis
	boom := errors.New("boom")
	source := &incomeSourceFake{failOnNth: 1, err: boom}
	state := newIncomeState()

	err := backfill(t, source, &incomeRepoFake{}, state, since, since+dayMillis)

	if !errors.Is(err, boom) {
		t.Fatalf("Fehler = %v", err)
	}
	if _, saved := state.values[appposition.IncomeHistoryFromKey]; saved {
		t.Fatal("State darf bei Fehler nicht gesetzt sein")
	}
}
