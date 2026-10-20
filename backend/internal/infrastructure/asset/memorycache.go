// Package asset enthält den In-Memory-Cache für Coin-Icons.
package asset

import (
	"sync"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
	shared "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
)

type cacheEntry struct {
	icon      *asset.IconURL
	expiresAt time.Time
}

// MemoryIconCache ist ein race-sicherer Cache mit TTL pro Eintrag.
type MemoryIconCache struct {
	mutex   sync.Mutex
	clock   shared.Clock
	entries map[asset.BaseAsset]cacheEntry
}

// NewMemoryIconCache baut einen leeren Cache.
func NewMemoryIconCache(clock shared.Clock) *MemoryIconCache {
	return &MemoryIconCache{clock: clock, entries: map[asset.BaseAsset]cacheEntry{}}
}

// Get liefert den Eintrag, solange er nicht abgelaufen ist (genau zum Ablaufzeitpunkt ist er abgelaufen).
func (cache *MemoryIconCache) Get(base asset.BaseAsset) (*asset.IconURL, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	entry, found := cache.entries[base]
	if !found {
		return nil, false
	}
	if !cache.clock.Now().Before(entry.expiresAt) {
		delete(cache.entries, base)
		return nil, false
	}
	return entry.icon, true
}

// Put speichert den Eintrag (nil = bekannt ohne Icon) und überschreibt Wert und TTL.
func (cache *MemoryIconCache) Put(base asset.BaseAsset, icon *asset.IconURL, ttl time.Duration) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.entries[base] = cacheEntry{icon: icon, expiresAt: cache.clock.Now().Add(ttl)}
}
