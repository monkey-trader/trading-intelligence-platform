package asset_test

import (
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
	infraasset "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/asset"
)

type fakeClock struct {
	mutex sync.Mutex
	now   time.Time
}

func (clock *fakeClock) Now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.now
}

func (clock *fakeClock) advance(duration time.Duration) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.now = clock.now.Add(duration)
}

func newClock() *fakeClock { return &fakeClock{now: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)} }

func base(t *testing.T, symbol string) asset.BaseAsset {
	t.Helper()
	value, err := asset.NewBaseAsset(symbol)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func url(t *testing.T, raw string) *asset.IconURL {
	t.Helper()
	value, err := asset.NewIconURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &value
}

const iconURL = "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png"

func TestMemoryCache_UnbekanntIstKeinTreffer(t *testing.T) {
	cache := infraasset.NewMemoryIconCache(newClock())
	if _, found := cache.Get(base(t, "BTC")); found {
		t.Error("leerer Cache darf nichts finden")
	}
}

func TestMemoryCache_PositivUndNegativEintrag(t *testing.T) {
	cache := infraasset.NewMemoryIconCache(newClock())
	cache.Put(base(t, "BTC"), url(t, iconURL), time.Hour)
	cache.Put(base(t, "ENA"), nil, time.Hour)

	icon, found := cache.Get(base(t, "BTC"))
	if !found || icon == nil || icon.String() != iconURL {
		t.Errorf("BTC = %v, %v", icon, found)
	}
	icon, found = cache.Get(base(t, "ENA"))
	if !found || icon != nil {
		t.Errorf("ENA: erwartet gefunden mit nil, war %v, %v", icon, found)
	}
}

func TestMemoryCache_Ablauf_AnDerGrenze(t *testing.T) {
	clock := newClock()
	cache := infraasset.NewMemoryIconCache(clock)
	cache.Put(base(t, "BTC"), url(t, iconURL), 10*time.Minute)

	clock.advance(10*time.Minute - time.Millisecond)
	if _, found := cache.Get(base(t, "BTC")); !found {
		t.Error("1 ms vor Ablauf muss noch gelten")
	}
	clock.advance(time.Millisecond)
	if _, found := cache.Get(base(t, "BTC")); found {
		t.Error("genau zum Ablaufzeitpunkt muss der Eintrag abgelaufen sein")
	}
}

func TestMemoryCache_UeberschreibenSetztWertUndTTLNeu(t *testing.T) {
	clock := newClock()
	cache := infraasset.NewMemoryIconCache(clock)
	cache.Put(base(t, "BTC"), nil, time.Minute)
	clock.advance(30 * time.Second)
	cache.Put(base(t, "BTC"), url(t, iconURL), time.Hour)

	clock.advance(2 * time.Minute)
	icon, found := cache.Get(base(t, "BTC"))
	if !found || icon == nil {
		t.Errorf("überschriebener Eintrag = %v, %v, erwartet Icon mit neuer TTL", icon, found)
	}
}

func TestMemoryCache_NebenlaeufigerZugriff(t *testing.T) {
	cache := infraasset.NewMemoryIconCache(newClock())
	bases := []asset.BaseAsset{base(t, "BTC"), base(t, "SUI"), base(t, "ENA")}
	var group sync.WaitGroup
	for worker := range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for round := range 200 {
				target := bases[(worker+round)%len(bases)]
				cache.Put(target, url(t, iconURL), time.Hour)
				cache.Get(target)
			}
		}()
	}
	group.Wait()
}
