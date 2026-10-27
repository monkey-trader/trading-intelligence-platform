package position_test

import (
	"context"
	"errors"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Erwartet: (*GetOpenPositionsUseCase).WithCandleSource(chartinterfaces.CandleSource). Pro Position:
// FetchCandles(symbol, 15m, limit) -> UpsertCandles -> ListCandles(symbol, 15m, limit). limit =
// ceil((now-openTime)/15min) + Puffer (1..10); braucht das mehr als 1440, bleibt 2R unbekannt, ohne Abruf.

const (
	freshOpenTime  = int64(1_791_500_000_000)
	fifteenMinutes = int64(15 * 60 * 1000)
	maxFetchLimit  = 1440
)

type fetchCall struct {
	symbol   string
	interval chart.CandleInterval
	limit    int
}

// freshSource liefert feste Kerzen und merkt sich die Aufrufe.
type freshSource struct {
	candles []chart.Candle
	err     error
	calls   []fetchCall
}

func (source *freshSource) FetchCandles(_ context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	source.calls = append(source.calls, fetchCall{symbol, interval, limit})
	return source.candles, source.err
}

// cacheRepo ist ein Cache, der Upserts übernimmt (ListCandles liefert, was gespeichert wurde, sonst den Startbestand).
type cacheRepo struct {
	stored    []chart.Candle
	upserted  [][]chart.Candle
	upsertErr error
	listCalls []fetchCall
}

func (repo *cacheRepo) UpsertCandles(_ context.Context, candles []chart.Candle) error {
	repo.upserted = append(repo.upserted, candles)
	if repo.upsertErr != nil {
		return repo.upsertErr
	}
	byTime := map[int64]chart.Candle{}
	for _, c := range repo.stored {
		byTime[c.OpenTime] = c
	}
	for _, c := range candles {
		byTime[c.OpenTime] = c
	}
	merged := make([]chart.Candle, 0, len(byTime))
	for _, c := range byTime {
		merged = append(merged, c)
	}
	for i := range merged { // aufsteigend sortieren
		for j := i + 1; j < len(merged); j++ {
			if merged[j].OpenTime < merged[i].OpenTime {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}
	repo.stored = merged
	return nil
}

func (repo *cacheRepo) ListCandles(_ context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	repo.listCalls = append(repo.listCalls, fetchCall{symbol, interval, limit})
	return repo.stored, nil
}

func freshRisks() *fakeRiskRepo {
	return &fakeRiskRepo{stored: map[string]position.PositionRisk{riskPositionID: {PositionID: riskPositionID, EntryPrice: "0.9600", InitialStopLoss: "0.9000"}}}
}

func executeFresh(t *testing.T, now int64, source *freshSource, cache *cacheRepo) appposition.OpenPositionEntry {
	t.Helper()
	useCase := appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(openPosition(nil)), &fakeOrders{}).
		WithRisk(freshRisks(), cache, func() int64 { return now }).
		WithCandleSource(source)
	overview, err := useCase.Execute(context.Background())
	if err != nil || len(overview.Positions) != 1 {
		t.Fatalf("err=%v positions=%d", err, len(overview.Positions))
	}
	return overview.Positions[0]
}

// held15m liefert ein "jetzt", das genau n volle 15-Minuten-Intervalle nach der Eröffnung liegt.
func held15m(n int64) int64 { return freshOpenTime + n*fifteenMinutes }

func TestFresh_SourceWirdMitSymbol15mUndLimitAufgerufenUndCacheBefuellt(t *testing.T) {
	fetched := []chart.Candle{candle(freshOpenTime, "1.0"), candle(1_791_600_000_000, "1.0")}
	source := &freshSource{candles: fetched}
	cache := &cacheRepo{}
	executeFresh(t, riskNow, source, cache)

	if len(source.calls) != 1 || source.calls[0].symbol != "SUI-USDT" || source.calls[0].interval != chart.Interval15m {
		t.Fatalf("FetchCandles-Aufrufe: %+v", source.calls)
	}
	// Haltedauer: ceil(100_900_000 / 900_000) = 113 Intervalle.
	if limit := source.calls[0].limit; limit < 114 || limit > 123 {
		t.Errorf("limit = %d, erwartet 113 + Puffer (1..10)", limit)
	}
	if len(cache.upserted) != 1 || len(cache.upserted[0]) != len(fetched) {
		t.Errorf("UpsertCandles: %+v", cache.upserted)
	}
	if len(cache.listCalls) != 1 || cache.listCalls[0].symbol != "SUI-USDT" || cache.listCalls[0].interval != chart.Interval15m ||
		cache.listCalls[0].limit != source.calls[0].limit {
		t.Errorf("ListCandles: %+v", cache.listCalls)
	}
}

func TestFresh_FrischeKerzenMachenAusVeraltetemCacheEinEchtesErgebnis(t *testing.T) {
	staleCache := func() *cacheRepo {
		return &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799")}}
	}
	// Cache endet 15 Minuten nach der Eröffnung, "jetzt" liegt 112 Minuten später: ohne Source unbekannt.
	now := riskNow
	cases := []struct {
		name        string
		newCandles  []chart.Candle
		wantReached bool
		wantAt      int64
	}{
		{"2R erreicht", []chart.Candle{candle(1_791_600_000_000, "1.08")}, true, 1_791_600_000_000},
		{"2R nicht erreicht", []chart.Candle{candle(1_791_600_000_000, "1.0799")}, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Lücke schließen: Zwischenkerzen + neue Kerze, damit die Reihe bis "jetzt" reicht.
			entry := executeFresh(t, now, &freshSource{candles: tc.newCandles}, staleCache())
			if entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached != tc.wantReached {
				t.Fatalf("BreakEvenTriggerReached = %v", entry.BreakEvenTriggerReached)
			}
			if tc.wantReached && (entry.Reached2RAt == nil || *entry.Reached2RAt != tc.wantAt) {
				t.Errorf("Reached2RAt = %v", entry.Reached2RAt)
			}
			if !tc.wantReached && entry.Reached2RAt != nil {
				t.Errorf("Reached2RAt = %v", entry.Reached2RAt)
			}
		})
	}
}

func TestFresh_FetchFehlerRechnetMitCacheWeiter(t *testing.T) {
	t.Run("aktueller Cache: weiterhin false", func(t *testing.T) {
		cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799"), candle(1_791_600_000_000, "1.0799")}}
		entry := executeFresh(t, riskNow, &freshSource{err: errors.New("429")}, cache)
		if entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached || entry.Reached2RAt != nil {
			t.Errorf("%v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
		}
	})
	t.Run("veralteter Cache: unbekannt", func(t *testing.T) {
		cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799"), candle(1_791_600_000_000, "1.0799")}}
		entry := executeFresh(t, riskNow+16*60*1000, &freshSource{err: errors.New("429")}, cache)
		if entry.BreakEvenTriggerReached != nil || entry.Reached2RAt != nil {
			t.Errorf("%v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
		}
		if entry.CurrentRMultiple == nil {
			t.Error("R-Werte müssen trotz Fetch-Fehler bleiben")
		}
	})
	t.Run("erreichtes 2R im Cache bleibt auch bei veraltetem Cache", func(t *testing.T) {
		cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.08")}}
		entry := executeFresh(t, riskNow+24*60*60*1000, &freshSource{err: errors.New("429")}, cache)
		if entry.BreakEvenTriggerReached == nil || !*entry.BreakEvenTriggerReached {
			t.Errorf("%v", entry.BreakEvenTriggerReached)
		}
	})
}

func TestFresh_UpsertFehlerStuerztNichtAb(t *testing.T) {
	cache := &cacheRepo{
		stored:    []chart.Candle{candle(freshOpenTime, "1.0799"), candle(1_791_600_000_000, "1.0799")},
		upsertErr: errors.New("disk full"),
	}
	entry := executeFresh(t, riskNow, &freshSource{candles: []chart.Candle{candle(1_791_600_000_000, "1.0799")}}, cache)
	if entry.CurrentRMultiple == nil || entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached {
		t.Errorf("weiter mit Cache erwartet: %+v", entry)
	}
}

func TestFresh_LimitWaechstMitHaltedauerUndUeber1440IstUnbekannt(t *testing.T) {
	limitFor := func(t *testing.T, intervals int64) (int, appposition.OpenPositionEntry, *freshSource) {
		t.Helper()
		source := &freshSource{}
		// Cache deckt alles ab und erreicht 2R nicht; Aussage hängt nur am limit-Verhalten.
		cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799"), candle(held15m(intervals)-fifteenMinutes, "1.0799")}}
		entry := executeFresh(t, held15m(intervals), source, cache)
		if len(source.calls) == 0 {
			return 0, entry, source
		}
		return source.calls[0].limit, entry, source
	}

	t.Run("limit deckt die Haltedauer und wächst mit ihr", func(t *testing.T) {
		previous := 0
		for _, intervals := range []int64{4, 100, 1000} {
			limit, _, _ := limitFor(t, intervals)
			if limit < int(intervals)+1 || limit > int(intervals)+10 || limit > maxFetchLimit {
				t.Errorf("%d Intervalle -> limit %d", intervals, limit)
			}
			if limit <= previous {
				t.Errorf("limit %d wächst nicht (vorher %d)", limit, previous)
			}
			previous = limit
		}
	})
	t.Run("angebrochenes Intervall wird aufgerundet", func(t *testing.T) {
		source := &freshSource{}
		cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799")}}
		executeFresh(t, held15m(10)+1, source, cache)
		if len(source.calls) != 1 || source.calls[0].limit < 11 {
			t.Errorf("calls: %+v", source.calls)
		}
	})
	t.Run("über 1440 Intervalle: kein Abruf, 2R unbekannt", func(t *testing.T) {
		limit, entry, source := limitFor(t, 1500)
		if limit != 0 || len(source.calls) != 0 {
			t.Errorf("es darf nichts abgerufen werden: %+v", source.calls)
		}
		if entry.BreakEvenTriggerReached != nil || entry.Reached2RAt != nil {
			t.Errorf("unbekannt erwartet: %v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
		}
		if entry.CurrentRMultiple == nil {
			t.Error("R-Werte müssen bleiben")
		}
	})
}

func TestFresh_ZweiRInErsterUndLetzterKerze(t *testing.T) {
	lastStart := 1_791_600_000_000
	cases := []struct {
		name    string
		candles []chart.Candle
		wantAt  int64
	}{
		{"erste Kerze", []chart.Candle{candle(freshOpenTime, "1.08"), candle(int64(lastStart), "1.0")}, freshOpenTime},
		{"letzte Kerze", []chart.Candle{candle(freshOpenTime, "1.0"), candle(int64(lastStart), "1.08")}, int64(lastStart)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := executeFresh(t, riskNow, &freshSource{candles: tc.candles}, &cacheRepo{})
			if entry.BreakEvenTriggerReached == nil || !*entry.BreakEvenTriggerReached ||
				entry.Reached2RAt == nil || *entry.Reached2RAt != tc.wantAt {
				t.Errorf("%v %v", entry.BreakEvenTriggerReached, entry.Reached2RAt)
			}
		})
	}
}

func TestFresh_OhneCandleSourceNurCache(t *testing.T) {
	cache := &cacheRepo{stored: []chart.Candle{candle(freshOpenTime, "1.0799"), candle(1_791_600_000_000, "1.0799")}}
	useCase := appposition.NewGetOpenPositionsUseCaseWithProtection(sourceWith(openPosition(nil)), &fakeOrders{}).
		WithRisk(freshRisks(), cache, func() int64 { return riskNow })
	overview, err := useCase.Execute(context.Background())
	if err != nil || len(overview.Positions) != 1 {
		t.Fatalf("err=%v", err)
	}
	entry := overview.Positions[0]
	if entry.BreakEvenTriggerReached == nil || *entry.BreakEvenTriggerReached || len(cache.upserted) != 0 {
		t.Errorf("reached=%v upserts=%d", entry.BreakEvenTriggerReached, len(cache.upserted))
	}
}
