package chart_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domainchart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	chartinfra "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/chart"
)

type countingOrders struct {
	calls int
	err   error
}

func (source *countingOrders) FetchProtectiveOrders(context.Context) (domainchart.ProtectiveOrdersSnapshot, error) {
	source.calls++
	if source.err != nil {
		return domainchart.ProtectiveOrdersSnapshot{}, source.err
	}
	return domainchart.ProtectiveOrdersSnapshot{FetchedAt: int64(source.calls)}, nil
}

func TestCachingProtectiveOrderSource(t *testing.T) {
	const ttl = 5 * time.Second
	clock := &fakeClock{current: time.UnixMilli(1_791_000_000_000)}
	inner := &countingOrders{}
	cache := chartinfra.NewCachingProtectiveOrderSource(inner, clock, ttl)

	for range 2 {
		if _, err := cache.FetchProtectiveOrders(context.Background()); err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
	}
	if inner.calls != 1 {
		t.Fatalf("%d Abfragen innerhalb der TTL, erwartet 1", inner.calls)
	}

	clock.current = clock.current.Add(ttl)
	snapshot, err := cache.FetchProtectiveOrders(context.Background())
	if err != nil || inner.calls != 2 || snapshot.FetchedAt != 2 {
		t.Fatalf("nach TTL: calls=%d fetchedAt=%d err=%v, erwartet neue Abfrage", inner.calls, snapshot.FetchedAt, err)
	}

	t.Run("Fehler werden nicht gecacht", func(t *testing.T) {
		failing := &countingOrders{err: errors.New("boom")}
		failingCache := chartinfra.NewCachingProtectiveOrderSource(failing, clock, ttl)
		for range 2 {
			if _, err := failingCache.FetchProtectiveOrders(context.Background()); !errors.Is(err, failing.err) {
				t.Fatalf("Fehler = %v, erwartet boom", err)
			}
		}
		if failing.calls != 2 {
			t.Errorf("%d Abfragen, erwartet 2", failing.calls)
		}
	})
}
