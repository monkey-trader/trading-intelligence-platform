package position

import (
	"context"
	"errors"
	"sync"
	"time"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
	sharedinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/interfaces"
)

// CacheOptions konfiguriert den Cache.
type CacheOptions struct {
	// TTL: so lange wird ein Erfolg unverändert ausgeliefert.
	TTL time.Duration
	// RateLimitHold: so lange wird ein rate-limited-Fehler erneut geliefert, ohne zu fragen.
	RateLimitHold time.Duration
	// FetchTimeout begrenzt die geteilte Abfrage; 0 bedeutet ohne eigene Frist.
	FetchTimeout time.Duration
}

// sharedFetch ist eine laufende Abfrage, auf die sich mehrere Aufrufer setzen.
// snapshot und err werden vor close(done) geschrieben und danach nur gelesen.
type sharedFetch struct {
	done     chan struct{}
	snapshot domainposition.OpenPositionsSnapshot
	err      error
}

// CachingSource cacht Erfolge, hält rate-limited zurück und teilt laufende Abfragen.
type CachingSource struct {
	source  interfaces.OpenPositionSource
	clock   sharedinterfaces.Clock
	options CacheOptions

	mutex             sync.Mutex
	cachedSnapshot    *domainposition.OpenPositionsSnapshot
	cacheExpiresAt    time.Time
	heldRateLimit     error
	rateLimitReleases time.Time
	inFlight          *sharedFetch
}

// NewCachingSource baut den Cache um source.
func NewCachingSource(source interfaces.OpenPositionSource, clock sharedinterfaces.Clock, options CacheOptions) *CachingSource {
	return &CachingSource{source: source, clock: clock, options: options}
}

// FetchOpenPositions liefert den Snapshot aus dem Cache oder aus einer (geteilten) Abfrage.
// Bricht ctx ab, kehrt nur dieser Aufrufer zurück; die geteilte Abfrage läuft für die anderen weiter.
func (cache *CachingSource) FetchOpenPositions(ctx context.Context) (domainposition.OpenPositionsSnapshot, error) {
	cache.mutex.Lock()
	now := cache.clock.Now()
	if cache.cachedSnapshot != nil && now.Before(cache.cacheExpiresAt) {
		snapshot := *cache.cachedSnapshot
		cache.mutex.Unlock()
		return snapshot, nil
	}
	if cache.heldRateLimit != nil && now.Before(cache.rateLimitReleases) {
		heldError := cache.heldRateLimit
		cache.mutex.Unlock()
		return domainposition.OpenPositionsSnapshot{}, heldError
	}
	if cache.inFlight == nil {
		cache.inFlight = &sharedFetch{done: make(chan struct{})}
		// Ohne Abbruch des ersten Aufrufers, aber mit seinen Werten und einer eigenen Frist.
		go cache.runSharedFetch(context.WithoutCancel(ctx), cache.inFlight)
	}
	current := cache.inFlight
	cache.mutex.Unlock()

	select {
	case <-current.done:
		return current.snapshot, current.err
	case <-ctx.Done():
		return domainposition.OpenPositionsSnapshot{}, ctx.Err()
	}
}

// runSharedFetch fragt die Source, merkt sich Erfolg oder rate-limited und weckt alle Wartenden.
func (cache *CachingSource) runSharedFetch(baseContext context.Context, fetch *sharedFetch) {
	fetchContext, cancel := cache.fetchContext(baseContext)
	defer cancel()

	snapshot, err := cache.safeFetch(fetchContext)

	cache.mutex.Lock()
	now := cache.clock.Now()
	if err == nil {
		cache.cachedSnapshot = &snapshot
		cache.cacheExpiresAt = now.Add(cache.options.TTL)
		cache.heldRateLimit = nil
	} else if isRateLimited(err) {
		cache.heldRateLimit = err
		cache.rateLimitReleases = now.Add(cache.options.RateLimitHold)
	}
	cache.inFlight = nil
	fetch.snapshot, fetch.err = snapshot, err
	cache.mutex.Unlock()
	close(fetch.done)
}

// safeFetch fragt die Source und macht aus einem Panic einen InvalidResponse-Fehler.
// Panic-Wert und Stack landen bewusst nicht im Fehler, damit nichts nach außen gelangt;
// so wird inFlight immer aufgeräumt und done immer geschlossen.
func (cache *CachingSource) safeFetch(fetchContext context.Context) (snapshot domainposition.OpenPositionsSnapshot, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			snapshot = domainposition.OpenPositionsSnapshot{}
			err = &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonInvalidResponse}
		}
	}()
	return cache.source.FetchOpenPositions(fetchContext)
}

func (cache *CachingSource) fetchContext(baseContext context.Context) (context.Context, context.CancelFunc) {
	if cache.options.FetchTimeout <= 0 {
		return context.WithCancel(baseContext)
	}
	return context.WithTimeout(baseContext, cache.options.FetchTimeout)
}

func isRateLimited(err error) bool {
	var unavailable *domainposition.OpenPositionsUnavailableError
	return errors.As(err, &unavailable) && unavailable.Reason == domainposition.ReasonRateLimited
}
