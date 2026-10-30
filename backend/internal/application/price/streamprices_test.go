package price_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	domainprice "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/price"
)

type fakeSource struct {
	mutex  sync.Mutex
	quotes []domainprice.Quote
	err    error
	calls  int
}

func (source *fakeSource) Fetch(context.Context) ([]domainprice.Quote, error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	source.calls++
	return source.quotes, source.err
}

func (source *fakeSource) set(quotes []domainprice.Quote, err error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	source.quotes, source.err = quotes, err
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.UnixMilli(1_791_800_000_000) }

func levels(breakEven, takeProfit string) []domainposition.RLevel {
	return []domainposition.RLevel{{Multiple: 2, Price: breakEven}, {Multiple: 3, Price: takeProfit}}
}

func entry(id, symbol string, side domainposition.PositionSide, entryLevels []domainposition.RLevel) appposition.OpenPositionEntry {
	return appposition.OpenPositionEntry{
		OpenPosition: domainposition.OpenPosition{PositionID: id, Symbol: symbol, Side: side},
		Levels:       entryLevels,
	}
}

type harness struct {
	source   *fakeSource
	broker   *price.Broker[price.PriceSnapshot]
	useCase  *price.StreamPricesUseCase
	snapshot <-chan price.PriceSnapshot
	sleeps   []time.Duration
	jitter   func(time.Duration) time.Duration
	entries  []appposition.OpenPositionEntry
}

func newHarness(entries ...appposition.OpenPositionEntry) *harness {
	harness := &harness{source: &fakeSource{}, broker: price.NewBroker[price.PriceSnapshot](16), entries: entries}
	harness.jitter = func(time.Duration) time.Duration { return 0 }
	harness.useCase = price.NewStreamPricesUseCase(price.StreamPricesConfig{
		Source: harness.source,
		OpenPositions: func(context.Context) (appposition.OpenPositionsOverview, error) {
			return appposition.OpenPositionsOverview{Positions: harness.entries}, nil
		},
		Broker: harness.broker,
		Clock:  fixedClock{},
		Sleep: func(_ context.Context, wait time.Duration) error {
			harness.sleeps = append(harness.sleeps, wait)
			return nil
		},
		Jitter: func(base time.Duration) time.Duration { return harness.jitter(base) },
	})
	harness.snapshot, _ = harness.broker.Subscribe()
	return harness
}

func (harness *harness) next(t *testing.T) price.PriceSnapshot {
	t.Helper()
	select {
	case snapshot := <-harness.snapshot:
		return snapshot
	case <-time.After(time.Second):
		t.Fatal("kein Snapshot publiziert")
	}
	return price.PriceSnapshot{}
}

func TestPollWaitsTwoSecondsAfterSuccessAndMakesOneRequest(t *testing.T) {
	harness := newHarness(entry("1", "SUI-USDT", domainposition.Long, nil))
	harness.source.set([]domainprice.Quote{{Symbol: "SUI-USDT", Price: "3", Change24hPercent: "1.3"}}, nil)

	wait := harness.useCase.Poll(context.Background())

	if wait != 2*time.Second {
		t.Errorf("Wartezeit = %v, erwartet 2s", wait)
	}
	if harness.source.calls != 1 {
		t.Errorf("Source-Aufrufe = %d, erwartet 1 je Takt", harness.source.calls)
	}
	snapshot := harness.next(t)
	if snapshot.Stale || snapshot.FetchedAt != 1_791_800_000_000 {
		t.Errorf("Snapshot = %+v, erwartet stale=false und FetchedAt aus der Uhr", snapshot)
	}
}

func TestPollMapsQuotesToOpenPositionsInPositionOrder(t *testing.T) {
	harness := newHarness(
		entry("2", "SUI-USDT", domainposition.Long, nil),
		entry("1", "BTC-USDT", domainposition.Short, nil),
		entry("3", "NOPRICE-USDT", domainposition.Long, nil),
	)
	harness.source.set([]domainprice.Quote{
		{Symbol: "BTC-USDT", Price: "100", Change24hPercent: "-0.5"},
		{Symbol: "SUI-USDT", Price: "3", Change24hPercent: "1.3"},
		{Symbol: "OTHER-USDT", Price: "9", Change24hPercent: "0"},
	}, nil)

	harness.useCase.Poll(context.Background())

	prices := harness.next(t).Prices
	if len(prices) != 2 {
		t.Fatalf("Prices = %+v, erwartet 2 Einträge (Position ohne Kurs entfällt, fremde Symbole ignoriert)", prices)
	}
	if prices[0].PositionID != "2" || prices[0].Symbol != "SUI-USDT" || prices[0].Price != "3" || prices[0].Change24hPercent != "1.3" {
		t.Errorf("Eintrag 0 = %+v", prices[0])
	}
	if prices[1].PositionID != "1" || prices[1].Price != "100" || prices[1].Change24hPercent != "-0.5" {
		t.Errorf("Eintrag 1 = %+v", prices[1])
	}
}

func TestPollWithoutOpenPositionsPublishesEmptyNonNilList(t *testing.T) {
	harness := newHarness()
	harness.source.set([]domainprice.Quote{{Symbol: "BTC-USDT", Price: "1", Change24hPercent: "0"}}, nil)
	harness.useCase.Poll(context.Background())
	if prices := harness.next(t).Prices; prices == nil || len(prices) != 0 {
		t.Errorf("Prices = %#v, erwartet leere, nicht-nil Liste", prices)
	}
}

func TestPollComputesDistancesToBreakEvenAndTp1(t *testing.T) {
	cases := []struct {
		name           string
		side           domainposition.PositionSide
		price          string
		breakEven      string
		takeProfit     string
		wantBE, wantTP string
	}{
		{"LONG Level vor dem Kurs", domainposition.Long, "100", "102", "103", "2", "3"},
		{"LONG Kurs über dem Level ist negativ", domainposition.Long, "100", "98", "99", "-2", "-1"},
		{"SHORT Level unter dem Kurs ist positiv", domainposition.Short, "100", "98", "97", "2", "3"},
		{"SHORT Kurs unter dem Level ist negativ", domainposition.Short, "100", "102", "101", "-2", "-1"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newHarness(entry("1", "SUI-USDT", testCase.side, levels(testCase.breakEven, testCase.takeProfit)))
			harness.source.set([]domainprice.Quote{{Symbol: "SUI-USDT", Price: testCase.price, Change24hPercent: "0"}}, nil)

			harness.useCase.Poll(context.Background())

			got := harness.next(t).Prices[0]
			if got.DistanceToBreakEvenPercent == nil || *got.DistanceToBreakEvenPercent != testCase.wantBE {
				t.Errorf("BE-Abstand = %v, erwartet %s", got.DistanceToBreakEvenPercent, testCase.wantBE)
			}
			if got.DistanceToTp1Percent == nil || *got.DistanceToTp1Percent != testCase.wantTP {
				t.Errorf("TP1-Abstand = %v, erwartet %s", got.DistanceToTp1Percent, testCase.wantTP)
			}
		})
	}
}

func TestPollDistancesAreNilWithoutKnownRisk(t *testing.T) {
	harness := newHarness(entry("1", "SUI-USDT", domainposition.Long, nil))
	harness.source.set([]domainprice.Quote{{Symbol: "SUI-USDT", Price: "100", Change24hPercent: "0"}}, nil)
	harness.useCase.Poll(context.Background())
	got := harness.next(t).Prices[0]
	if got.DistanceToBreakEvenPercent != nil || got.DistanceToTp1Percent != nil {
		t.Errorf("Abstände = %v / %v, erwartet nil ohne Risiko", got.DistanceToBreakEvenPercent, got.DistanceToTp1Percent)
	}
}

func TestPollRepublishesLastSnapshotAsStaleOnError(t *testing.T) {
	harness := newHarness(entry("1", "SUI-USDT", domainposition.Long, nil))
	harness.source.set([]domainprice.Quote{{Symbol: "SUI-USDT", Price: "3", Change24hPercent: "1"}}, nil)
	harness.useCase.Poll(context.Background())
	good := harness.next(t)
	harness.source.set(nil, domainprice.ErrPriceUnavailable)

	harness.useCase.Poll(context.Background())

	stale := harness.next(t)
	if !stale.Stale {
		t.Fatalf("Snapshot = %+v, erwartet stale=true", stale)
	}
	if stale.FetchedAt != good.FetchedAt || len(stale.Prices) != 1 || stale.Prices[0].Price != "3" {
		t.Errorf("stale Snapshot = %+v, erwartet die letzten Kurse und den alten FetchedAt", stale)
	}
}

func TestPollErrorBeforeAnySnapshotPublishesEmptyStale(t *testing.T) {
	harness := newHarness(entry("1", "SUI-USDT", domainposition.Long, nil))
	harness.source.set(nil, errors.New("boom"))
	harness.useCase.Poll(context.Background())
	snapshot := harness.next(t)
	if !snapshot.Stale || snapshot.Prices == nil || len(snapshot.Prices) != 0 {
		t.Errorf("Snapshot = %+v, erwartet stale=true mit leerer, nicht-nil Liste", snapshot)
	}
}

func TestPollBackoffDoublesFromOneToThirtySecondsAndResetsOnSuccess(t *testing.T) {
	harness := newHarness()
	harness.source.set(nil, errors.New("down"))
	expected := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second}
	for index, want := range expected {
		if got := harness.useCase.Poll(context.Background()); got != want {
			t.Errorf("Fehler %d: Wartezeit = %v, erwartet %v", index+1, got, want)
		}
	}
	harness.source.set([]domainprice.Quote{}, nil)
	if got := harness.useCase.Poll(context.Background()); got != 2*time.Second {
		t.Errorf("nach Erfolg = %v, erwartet 2s", got)
	}
	harness.source.set(nil, errors.New("down"))
	if got := harness.useCase.Poll(context.Background()); got != time.Second {
		t.Errorf("erster Fehler nach Erfolg = %v, erwartet 1s (Backoff zurückgesetzt)", got)
	}
}

func TestPollBackoffAddsInjectedJitterButStaysWithinThirtySeconds(t *testing.T) {
	harness := newHarness()
	harness.source.set(nil, errors.New("down"))
	harness.jitter = func(base time.Duration) time.Duration { return base / 2 }
	if got := harness.useCase.Poll(context.Background()); got != 1500*time.Millisecond {
		t.Errorf("Wartezeit = %v, erwartet 1,5s (1s + Jitter 0,5s)", got)
	}
	for range 10 {
		if got := harness.useCase.Poll(context.Background()); got > 30*time.Second {
			t.Fatalf("Wartezeit = %v, darf 30s nicht überschreiten", got)
		}
	}
}

func TestRunPollsWithSleepUntilContextEnds(t *testing.T) {
	harness := newHarness()
	harness.source.set([]domainprice.Quote{}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	harness.useCase = price.NewStreamPricesUseCase(price.StreamPricesConfig{
		Source: harness.source,
		OpenPositions: func(context.Context) (appposition.OpenPositionsOverview, error) {
			return appposition.OpenPositionsOverview{}, nil
		},
		Broker: harness.broker,
		Clock:  fixedClock{},
		Sleep: func(_ context.Context, wait time.Duration) error {
			harness.sleeps = append(harness.sleeps, wait)
			calls++
			if calls == 3 {
				cancel()
				return context.Canceled
			}
			return nil
		},
		Jitter: func(time.Duration) time.Duration { return 0 },
	})

	done := make(chan struct{})
	go func() { harness.useCase.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run endet nicht nach Abbruch des Contexts")
	}
	if harness.source.calls != 3 || len(harness.sleeps) != 3 || harness.sleeps[0] != 2*time.Second {
		t.Errorf("Source-Aufrufe=%d, Sleeps=%v, erwartet 3 Takte à 2s", harness.source.calls, harness.sleeps)
	}
}
