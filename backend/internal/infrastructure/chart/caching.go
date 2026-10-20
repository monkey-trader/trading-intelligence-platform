package chart

import (
	"context"
	"sync"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart/interfaces"
	sharedinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
)

// CachingProtectiveOrderSource cacht erfolgreiche Antworten für die TTL; Fehler werden nicht gecacht.
// Gleichzeitige Aufrufer teilen sich eine Abfrage (der Mutex bleibt während der Abfrage gehalten).
type CachingProtectiveOrderSource struct {
	source interfaces.ProtectiveOrderSource
	clock  sharedinterfaces.Clock
	ttl    time.Duration

	mutex     sync.Mutex
	snapshot  *chart.ProtectiveOrdersSnapshot
	expiresAt time.Time
}

// NewCachingProtectiveOrderSource baut den Cache um source.
func NewCachingProtectiveOrderSource(source interfaces.ProtectiveOrderSource, clock sharedinterfaces.Clock, ttl time.Duration) *CachingProtectiveOrderSource {
	return &CachingProtectiveOrderSource{source: source, clock: clock, ttl: ttl}
}

// FetchProtectiveOrders liefert den Cache-Inhalt oder fragt die Source neu ab.
func (cache *CachingProtectiveOrderSource) FetchProtectiveOrders(ctx context.Context) (chart.ProtectiveOrdersSnapshot, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if cache.snapshot != nil && cache.clock.Now().Before(cache.expiresAt) {
		return *cache.snapshot, nil
	}
	snapshot, err := cache.source.FetchProtectiveOrders(ctx)
	if err != nil {
		return chart.ProtectiveOrdersSnapshot{}, err
	}
	cache.snapshot = &snapshot
	cache.expiresAt = cache.clock.Now().Add(cache.ttl)
	return snapshot, nil
}
