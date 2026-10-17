package position_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/position"
)

// Fälle portiert aus CachingOpenPositionSource.test.ts, ergänzt um ctx-Abbruch und FetchTimeout.
// Nebenläufige Fälle laufen in einer synctest-Blase: synctest.Wait() wartet, bis alle Goroutinen
// dauerhaft blockiert sind. So braucht es weder echte Timer noch Test-Hooks im Produktivcode.

const (
	cacheTTL           = 5 * time.Second
	cacheRateLimitHold = 30 * time.Second
	cacheFetchTimeout  = 8 * time.Second
	cacheStartMillis   = 1_791_000_000_000
)

var suiPosition = domainposition.OpenPosition{
	PositionID: "9100000000000000001", Symbol: "SUI-USDT", Side: domainposition.Long, Isolated: true,
	PositionAmount: "500", AvgOpenPrice: "0.9600", MarkPrice: "1.0100", Leverage: 10,
	InitialMargin: "48.0000", Margin: "60.5000", PositionValue: "505", UnrealizedProfit: "25.0000",
	RealisedProfit: "-0.2400", PnlRatio: "0.5208", LiquidationPrice: "0.865",
	OpenTime: 1791500000000, UpdateTime: 1791600000000,
}

func newCache(source *countingSource, clock *fakeClock) *position.CachingSource {
	return position.NewCachingSource(source, clock, position.CacheOptions{
		TTL: cacheTTL, RateLimitHold: cacheRateLimitHold, FetchTimeout: cacheFetchTimeout,
	})
}

// countingSource arbeitet vorbereitete Ergebnisse ab (nil = Erfolg); das letzte wiederholt sich.
// FetchedAt kommt aus der Fake-Clock zum Aufrufzeitpunkt.
type countingSource struct {
	mutex     sync.Mutex
	clock     *fakeClock
	outcomes  []error
	callCount int
}

func (source *countingSource) FetchOpenPositions(context.Context) (domainposition.OpenPositionsSnapshot, error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	outcome := source.outcomes[min(source.callCount, len(source.outcomes)-1)]
	source.callCount++
	if outcome != nil {
		return domainposition.OpenPositionsSnapshot{}, outcome
	}
	return domainposition.OpenPositionsSnapshot{
		Positions: []domainposition.OpenPosition{suiPosition},
		FetchedAt: source.clock.Now().UnixMilli(),
	}, nil
}

func (source *countingSource) calls() int {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	return source.callCount
}

type sourceOutcome struct {
	snapshot domainposition.OpenPositionsSnapshot
	err      error
}

// deferredSource antwortet erst, wenn der Test ein Ergebnis schickt, oder bricht über ctx ab.
type deferredSource struct {
	mutex     sync.Mutex
	callCount int
	waiting   int
	contexts  []context.Context
	outcomes  chan sourceOutcome
}

func newDeferredSource() *deferredSource {
	return &deferredSource{outcomes: make(chan sourceOutcome)}
}

func (source *deferredSource) FetchOpenPositions(ctx context.Context) (domainposition.OpenPositionsSnapshot, error) {
	source.mutex.Lock()
	source.callCount++
	source.contexts = append(source.contexts, ctx)
	source.waiting++
	source.mutex.Unlock()
	defer func() {
		source.mutex.Lock()
		source.waiting--
		source.mutex.Unlock()
	}()
	select {
	case outcome := <-source.outcomes:
		return outcome.snapshot, outcome.err
	case <-ctx.Done():
		return domainposition.OpenPositionsSnapshot{}, ctx.Err()
	}
}

// settle schickt das Ergebnis an die wartende Abfrage. Nur nach synctest.Wait() aufrufen;
// wartet keine Abfrage, schlägt der Test fehl, statt die Blase zu verklemmen.
func (source *deferredSource) settle(t *testing.T, outcome sourceOutcome) {
	t.Helper()
	source.mutex.Lock()
	waiting := source.waiting
	source.mutex.Unlock()
	if waiting != 1 {
		t.Fatalf("%d wartende Source-Abfragen, erwartet genau 1", waiting)
	}
	source.outcomes <- outcome
}

func (source *deferredSource) calls() int {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	return source.callCount
}

func (source *deferredSource) lastContext() context.Context {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	if len(source.contexts) == 0 {
		return nil
	}
	return source.contexts[len(source.contexts)-1]
}

type fetchResult struct {
	snapshot domainposition.OpenPositionsSnapshot
	err      error
}

func fetchAsync(ctx context.Context, cache *position.CachingSource) <-chan fetchResult {
	results := make(chan fetchResult, 1)
	go func() {
		snapshot, err := cache.FetchOpenPositions(ctx)
		results <- fetchResult{snapshot: snapshot, err: err}
	}()
	return results
}

func newDeferredCache(source *deferredSource, clock *fakeClock) *position.CachingSource {
	return position.NewCachingSource(source, clock, position.CacheOptions{
		TTL: cacheTTL, RateLimitHold: cacheRateLimitHold, FetchTimeout: cacheFetchTimeout,
	})
}

func expectedSnapshot(fetchedAt int64) domainposition.OpenPositionsSnapshot {
	return domainposition.OpenPositionsSnapshot{Positions: []domainposition.OpenPosition{suiPosition}, FetchedAt: fetchedAt}
}

func assertSnapshot(t *testing.T, got domainposition.OpenPositionsSnapshot, err error, expected domainposition.OpenPositionsSnapshot) {
	t.Helper()
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.FetchedAt != expected.FetchedAt || len(got.Positions) != len(expected.Positions) {
		t.Fatalf("Snapshot = %+v, erwartet %+v", got, expected)
	}
	for index := range expected.Positions {
		if got.Positions[index] != expected.Positions[index] {
			t.Errorf("Position %d = %+v, erwartet %+v", index, got.Positions[index], expected.Positions[index])
		}
	}
}

func assertReason(t *testing.T, err error, expected domainposition.UnavailableReason) {
	t.Helper()
	var unavailable *domainposition.OpenPositionsUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("Fehler = %v (%T), erwartet *OpenPositionsUnavailableError", err, err)
	}
	if unavailable.Reason != expected {
		t.Errorf("Reason = %q, erwartet %q", unavailable.Reason, expected)
	}
}

func TestCachingSource_ErsterAufrufReichtDurch(t *testing.T) {
	clock := newFakeClock(cacheStartMillis)
	source := &countingSource{clock: clock, outcomes: []error{nil}}
	cache := newCache(source, clock)

	snapshot, err := cache.FetchOpenPositions(context.Background())

	assertSnapshot(t, snapshot, err, expectedSnapshot(cacheStartMillis))
	if source.calls() != 1 {
		t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
	}
}

func TestCachingSource_TTL(t *testing.T) {
	testCases := []struct {
		name              string
		advance           time.Duration
		expectedCalls     int
		expectedFetchedAt int64
	}{
		{"innerhalb der TTL: kein neuer Aufruf, ursprüngliches FetchedAt", cacheTTL - time.Millisecond, 1, cacheStartMillis},
		{"genau bei Ablauf der TTL: neuer Aufruf", cacheTTL, 2, cacheStartMillis + cacheTTL.Milliseconds()},
		{"nach Ablauf der TTL: neuer Aufruf mit neuem FetchedAt", cacheTTL + time.Millisecond, 2, cacheStartMillis + cacheTTL.Milliseconds() + 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clock := newFakeClock(cacheStartMillis)
			source := &countingSource{clock: clock, outcomes: []error{nil}}
			cache := newCache(source, clock)

			if _, err := cache.FetchOpenPositions(context.Background()); err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			clock.Advance(testCase.advance)
			snapshot, err := cache.FetchOpenPositions(context.Background())

			assertSnapshot(t, snapshot, err, expectedSnapshot(testCase.expectedFetchedAt))
			if source.calls() != testCase.expectedCalls {
				t.Errorf("Source %d-mal gefragt, erwartet %d", source.calls(), testCase.expectedCalls)
			}
		})
	}
}

func TestCachingSource_FehlerWerdenNichtGecacht(t *testing.T) {
	testCases := []struct {
		name    string
		failure error
	}{
		{"allgemeiner Fehler", errors.New("Programmierfehler")},
		{"unreachable", &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonUnreachable}},
		{"upstream-error", &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonUpstreamError}},
		{"not-configured", &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clock := newFakeClock(cacheStartMillis)
			source := &countingSource{clock: clock, outcomes: []error{testCase.failure, nil}}
			cache := newCache(source, clock)

			_, firstErr := cache.FetchOpenPositions(context.Background())
			snapshot, secondErr := cache.FetchOpenPositions(context.Background())

			if !errors.Is(firstErr, testCase.failure) {
				t.Errorf("erster Fehler = %v, erwartet den Source-Fehler", firstErr)
			}
			assertSnapshot(t, snapshot, secondErr, expectedSnapshot(cacheStartMillis))
			if source.calls() != 2 {
				t.Errorf("Source %d-mal gefragt, erwartet 2", source.calls())
			}
		})
	}
}

func TestCachingSource_RateLimitHold(t *testing.T) {
	clock := newFakeClock(cacheStartMillis)
	rateLimited := &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonRateLimited}
	source := &countingSource{clock: clock, outcomes: []error{rateLimited, nil}}
	cache := newCache(source, clock)

	_, firstErr := cache.FetchOpenPositions(context.Background())
	clock.Advance(time.Millisecond)
	_, heldAtStart := cache.FetchOpenPositions(context.Background())
	clock.Advance(cacheRateLimitHold - 2*time.Millisecond)
	_, heldAtEnd := cache.FetchOpenPositions(context.Background())

	for _, err := range []error{firstErr, heldAtStart, heldAtEnd} {
		assertReason(t, err, domainposition.ReasonRateLimited)
		if !errors.Is(err, rateLimited) {
			t.Errorf("Fehler = %v, erwartet den ursprünglichen rate-limited-Fehler", err)
		}
	}
	if source.calls() != 1 {
		t.Errorf("Source %d-mal gefragt, erwartet 1 während des Holds", source.calls())
	}
}

func TestCachingSource_NachAblaufDesHoldsWiederFragen(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		advance time.Duration
	}{
		{"genau bei Ablauf", cacheRateLimitHold},
		{"nach Ablauf", cacheRateLimitHold + time.Millisecond},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			clock := newFakeClock(cacheStartMillis)
			rateLimited := &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonRateLimited}
			source := &countingSource{clock: clock, outcomes: []error{rateLimited, nil}}
			cache := newCache(source, clock)

			_, _ = cache.FetchOpenPositions(context.Background())
			clock.Advance(testCase.advance)
			snapshot, err := cache.FetchOpenPositions(context.Background())

			assertSnapshot(t, snapshot, err, expectedSnapshot(cacheStartMillis+testCase.advance.Milliseconds()))
			if source.calls() != 2 {
				t.Errorf("Source %d-mal gefragt, erwartet 2", source.calls())
			}
		})
	}
}

func TestCachingSource_ErfolgLoeschtDenHold(t *testing.T) {
	clock := newFakeClock(cacheStartMillis)
	rateLimited := &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonRateLimited}
	upstream := &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonUpstreamError}
	source := &countingSource{clock: clock, outcomes: []error{rateLimited, nil, upstream, nil}}
	cache := newCache(source, clock)

	_, _ = cache.FetchOpenPositions(context.Background())
	clock.Advance(cacheRateLimitHold + time.Millisecond)
	if _, err := cache.FetchOpenPositions(context.Background()); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	clock.Advance(cacheTTL + time.Millisecond)
	_, upstreamErr := cache.FetchOpenPositions(context.Background())
	if _, err := cache.FetchOpenPositions(context.Background()); err != nil {
		t.Fatalf("unerwarteter Fehler nach upstream-error: %v", err)
	}

	if !errors.Is(upstreamErr, upstream) {
		t.Errorf("Fehler = %v, erwartet upstream-error", upstreamErr)
	}
	if source.calls() != 4 {
		t.Errorf("Source %d-mal gefragt, erwartet 4", source.calls())
	}
}

func TestCachingSource_TeiltGleichzeitigeAufrufe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)

		first := fetchAsync(context.Background(), cache)
		synctest.Wait()
		second := fetchAsync(context.Background(), cache)
		synctest.Wait()
		source.settle(t, sourceOutcome{snapshot: expectedSnapshot(cacheStartMillis)})

		for _, results := range []<-chan fetchResult{first, second} {
			result := <-results
			assertSnapshot(t, result.snapshot, result.err, expectedSnapshot(cacheStartMillis))
		}
		if source.calls() != 1 {
			t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
		}
	})
}

func TestCachingSource_VieleGleichzeitigeAufrufe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)

		const callerCount = 20
		resultChannels := make([]<-chan fetchResult, 0, callerCount)
		for range callerCount {
			resultChannels = append(resultChannels, fetchAsync(context.Background(), cache))
		}
		synctest.Wait()
		source.settle(t, sourceOutcome{snapshot: expectedSnapshot(cacheStartMillis)})

		for _, results := range resultChannels {
			result := <-results
			assertSnapshot(t, result.snapshot, result.err, expectedSnapshot(cacheStartMillis))
		}
		if source.calls() != 1 {
			t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
		}
	})
}

func TestCachingSource_TeiltFehlerUndGibtDanachFrei(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)
		failure := &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonUpstreamError}

		first := fetchAsync(context.Background(), cache)
		synctest.Wait()
		second := fetchAsync(context.Background(), cache)
		synctest.Wait()
		source.settle(t, sourceOutcome{err: failure})

		for _, results := range []<-chan fetchResult{first, second} {
			if result := <-results; !errors.Is(result.err, failure) {
				t.Errorf("Fehler = %v, erwartet den geteilten Fehler", result.err)
			}
		}
		if source.calls() != 1 {
			t.Fatalf("Source %d-mal gefragt, erwartet 1", source.calls())
		}

		third := fetchAsync(context.Background(), cache)
		synctest.Wait()
		if source.calls() != 2 {
			t.Fatalf("nach dem geteilten Fehler: Source %d-mal gefragt, erwartet 2", source.calls())
		}
		emptySnapshot := domainposition.OpenPositionsSnapshot{Positions: []domainposition.OpenPosition{}, FetchedAt: cacheStartMillis}
		source.settle(t, sourceOutcome{snapshot: emptySnapshot})
		result := <-third
		assertSnapshot(t, result.snapshot, result.err, emptySnapshot)
	})
}

func TestCachingSource_SpaeterAufrufNachTTLTeiltLaufendeAbfrage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)

		first := fetchAsync(context.Background(), cache)
		synctest.Wait()
		clock.Advance(cacheTTL + time.Millisecond)
		late := fetchAsync(context.Background(), cache)
		synctest.Wait()
		source.settle(t, sourceOutcome{snapshot: expectedSnapshot(cacheStartMillis)})

		for _, results := range []<-chan fetchResult{first, late} {
			result := <-results
			assertSnapshot(t, result.snapshot, result.err, expectedSnapshot(cacheStartMillis))
		}
		if source.calls() != 1 {
			t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
		}
	})
}

func TestCachingSource_AbbruchDesErstenAufrufersTrifftAndereNicht(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)
		firstContext, cancelFirst := context.WithCancel(context.Background())
		defer cancelFirst()

		first := fetchAsync(firstContext, cache)
		synctest.Wait()
		second := fetchAsync(context.Background(), cache)
		synctest.Wait()

		cancelFirst()
		synctest.Wait()
		firstResult := <-first
		if !errors.Is(firstResult.err, context.Canceled) {
			t.Errorf("erster Aufrufer: Fehler = %v, erwartet context.Canceled", firstResult.err)
		}
		if sourceContext := source.lastContext(); sourceContext == nil || sourceContext.Err() != nil {
			t.Fatalf("Context der geteilten Abfrage wurde mit abgebrochen")
		}

		source.settle(t, sourceOutcome{snapshot: expectedSnapshot(cacheStartMillis)})
		secondResult := <-second
		assertSnapshot(t, secondResult.snapshot, secondResult.err, expectedSnapshot(cacheStartMillis))

		// Das Ergebnis der weitergelaufenen Abfrage landet im Cache.
		snapshot, err := cache.FetchOpenPositions(context.Background())
		assertSnapshot(t, snapshot, err, expectedSnapshot(cacheStartMillis))
		if source.calls() != 1 {
			t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
		}
	})
}

func TestCachingSource_AbbruchEinesWartendenTrifftDenErstenNicht(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)
		joinerContext, cancelJoiner := context.WithCancel(context.Background())
		defer cancelJoiner()

		first := fetchAsync(context.Background(), cache)
		synctest.Wait()
		joiner := fetchAsync(joinerContext, cache)
		synctest.Wait()

		cancelJoiner()
		synctest.Wait()
		if result := <-joiner; !errors.Is(result.err, context.Canceled) {
			t.Errorf("Wartender: Fehler = %v, erwartet context.Canceled", result.err)
		}

		source.settle(t, sourceOutcome{snapshot: expectedSnapshot(cacheStartMillis)})
		firstResult := <-first
		assertSnapshot(t, firstResult.snapshot, firstResult.err, expectedSnapshot(cacheStartMillis))
		if source.calls() != 1 {
			t.Errorf("Source %d-mal gefragt, erwartet 1", source.calls())
		}
	})
}

func TestCachingSource_GeteilteAbfrageMitFetchTimeoutUndWertenDesAufrufers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type contextKey struct{}
		clock := newFakeClock(cacheStartMillis)
		source := newDeferredSource()
		cache := newDeferredCache(source, clock)
		callerContext, cancelCaller := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "anfrage-9"))
		defer cancelCaller()

		results := fetchAsync(callerContext, cache)
		synctest.Wait()

		sourceContext := source.lastContext()
		if sourceContext == nil {
			t.Fatal("Source wurde nicht gefragt")
		}
		deadline, hasDeadline := sourceContext.Deadline()
		if !hasDeadline {
			t.Fatal("geteilte Abfrage ohne Frist, erwartet FetchTimeout")
		}
		if remaining := time.Until(deadline); remaining != cacheFetchTimeout {
			t.Errorf("Restzeit = %v, erwartet FetchTimeout %v", remaining, cacheFetchTimeout)
		}
		if sourceContext.Value(contextKey{}) != "anfrage-9" {
			t.Errorf("Werte des Aufrufer-Contexts fehlen (erwartet context.WithoutCancel)")
		}

		// Ohne Antwort läuft die Frist ab; die Source bricht über ihren ctx ab.
		time.Sleep(cacheFetchTimeout)
		synctest.Wait()
		result := <-results
		if !errors.Is(result.err, context.DeadlineExceeded) {
			t.Errorf("Fehler = %v, erwartet context.DeadlineExceeded aus der Source", result.err)
		}
	})
}

// panickingSource panict bei den ersten panicCount Aufrufen, danach liefert sie Erfolg.
type panickingSource struct {
	mutex      sync.Mutex
	clock      *fakeClock
	panicCount int
	callCount  int
}

const panicSecret = "geheimer-panic-wert-4711"

func (source *panickingSource) FetchOpenPositions(context.Context) (domainposition.OpenPositionsSnapshot, error) {
	source.mutex.Lock()
	source.callCount++
	shouldPanic := source.callCount <= source.panicCount
	source.mutex.Unlock()
	if shouldPanic {
		panic(panicSecret)
	}
	return domainposition.OpenPositionsSnapshot{
		Positions: []domainposition.OpenPosition{suiPosition},
		FetchedAt: source.clock.Now().UnixMilli(),
	}, nil
}

func (source *panickingSource) calls() int {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	return source.callCount
}

func assertInvalidResponseWithoutLeak(t *testing.T, err error) {
	t.Helper()
	var unavailable *domainposition.OpenPositionsUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("Fehler = %v (%T), erwartet *OpenPositionsUnavailableError", err, err)
	}
	if unavailable.Reason != domainposition.ReasonInvalidResponse {
		t.Errorf("Reason = %q, erwartet %q", unavailable.Reason, domainposition.ReasonInvalidResponse)
	}
	for _, text := range []string{err.Error(), unavailable.Error()} {
		if strings.Contains(text, panicSecret) || strings.Contains(text, "goroutine") || strings.Contains(text, ".go:") {
			t.Errorf("Fehlertext enthält Panic-Wert oder Stack: %q", text)
		}
	}
}

func TestCachingSource_PanicDerSourceWirdZuInvalidResponse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := &panickingSource{clock: clock, panicCount: 1}
		cache := position.NewCachingSource(source, clock, position.CacheOptions{
			TTL: cacheTTL, RateLimitHold: cacheRateLimitHold, FetchTimeout: cacheFetchTimeout,
		})

		_, err := cache.FetchOpenPositions(context.Background())
		assertInvalidResponseWithoutLeak(t, err)

		// inFlight ist aufgeräumt, kein Fehler wurde gecacht: der zweite Aufruf fragt neu und gelingt.
		snapshot, err := cache.FetchOpenPositions(context.Background())
		if err != nil {
			t.Fatalf("zweiter Aufruf: unerwarteter Fehler: %v", err)
		}
		if len(snapshot.Positions) != 1 {
			t.Errorf("Positionen = %d, erwartet 1", len(snapshot.Positions))
		}
		if source.calls() != 2 {
			t.Errorf("Source-Aufrufe = %d, erwartet 2", source.calls())
		}
	})
}

// gatedPanickingSource blockiert, bis release geschlossen wird, und panict dann.
// Damit hängen alle Aufrufer nachweislich am selben inFlight, bevor der Panic eintritt.
type gatedPanickingSource struct {
	release   chan struct{}
	mutex     sync.Mutex
	callCount int
}

func (source *gatedPanickingSource) FetchOpenPositions(context.Context) (domainposition.OpenPositionsSnapshot, error) {
	source.mutex.Lock()
	source.callCount++
	source.mutex.Unlock()
	<-source.release
	panic(panicSecret)
}

func (source *gatedPanickingSource) calls() int {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	return source.callCount
}

func TestCachingSource_PanicWeckAlleGleichzeitigenWartenden(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clock := newFakeClock(cacheStartMillis)
		source := &gatedPanickingSource{release: make(chan struct{})}
		cache := position.NewCachingSource(source, clock, position.CacheOptions{
			TTL: cacheTTL, RateLimitHold: cacheRateLimitHold, FetchTimeout: cacheFetchTimeout,
		})

		const callerCount = 5
		errorsByCaller := make([]error, callerCount)
		var group sync.WaitGroup
		for index := range callerCount {
			group.Add(1)
			go func() {
				defer group.Done()
				_, errorsByCaller[index] = cache.FetchOpenPositions(context.Background())
			}()
		}
		// Alle Aufrufer warten an der einen geteilten Abfrage; die Source steckt noch am Gate.
		synctest.Wait()
		if source.calls() != 1 {
			t.Fatalf("Source-Aufrufe vor dem Panic = %d, erwartet 1", source.calls())
		}

		close(source.release)
		synctest.Wait()
		group.Wait() // hängt (Deadlock in der Blase), wenn done nie geschlossen wird

		for _, err := range errorsByCaller {
			assertInvalidResponseWithoutLeak(t, err)
		}
		if source.calls() != 1 {
			t.Errorf("Source-Aufrufe = %d, erwartet 1 (geteilte Abfrage)", source.calls())
		}
	})
}
