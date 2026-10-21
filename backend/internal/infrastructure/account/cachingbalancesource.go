package account

import (
	"context"
	"sync"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account"
	accountinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/account/interfaces"
)

// CachingBalanceSource hält eine erfolgreiche Antwort für ttl; Fehler werden nicht gecacht.
type CachingBalanceSource struct {
	inner accountinterfaces.BalanceSource
	now   func() time.Time
	ttl   time.Duration

	mutex     sync.Mutex
	balance   account.Balance
	fetchedAt time.Time
	valid     bool
}

// NewCachingBalanceSource umhüllt inner mit einem TTL-Cache.
func NewCachingBalanceSource(inner accountinterfaces.BalanceSource, now func() time.Time, ttl time.Duration) *CachingBalanceSource {
	return &CachingBalanceSource{inner: inner, now: now, ttl: ttl}
}

// FetchBalance liefert den Cache-Wert oder fragt die innere Source ab.
func (source *CachingBalanceSource) FetchBalance(ctx context.Context) (account.Balance, error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	if source.valid && source.now().Sub(source.fetchedAt) < source.ttl {
		return source.balance, nil
	}
	balance, err := source.inner.FetchBalance(ctx)
	if err != nil {
		return account.Balance{}, err
	}
	source.balance, source.fetchedAt, source.valid = balance, source.now(), true
	return balance, nil
}
