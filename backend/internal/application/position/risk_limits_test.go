package position_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

type scriptedHistory struct {
	mutex   sync.Mutex
	symbols []string
	err     error
	records []position.StopOrderRecord
	gate    chan struct{}
	calls   atomic.Int32
}

func (source *scriptedHistory) FetchStopOrders(_ context.Context, symbol string, _, _ int64) ([]position.StopOrderRecord, error) {
	source.calls.Add(1)
	if source.gate != nil {
		<-source.gate
	}
	source.mutex.Lock()
	defer source.mutex.Unlock()
	source.symbols = append(source.symbols, symbol)
	return source.records, source.err
}

func positionWithID(id string) position.OpenPosition {
	return openPosition(func(created *position.OpenPosition) { created.PositionID = id })
}

// lockedRiskRepo serialisiert den Zugriff auf das (nicht threadsichere) fakeRiskRepo.
type lockedRiskRepo struct {
	mutex sync.Mutex
	inner fakeRiskRepo
}

func (repo *lockedRiskRepo) SaveIfAbsent(ctx context.Context, risk position.PositionRisk) (bool, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()
	return repo.inner.SaveIfAbsent(ctx, risk)
}

func (repo *lockedRiskRepo) Get(ctx context.Context, positionID string) (position.PositionRisk, bool, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()
	return repo.inner.Get(ctx, positionID)
}

func newLimitedResolver(history interfaces.StopOrderHistorySource, clock *int64) *appposition.RiskResolver {
	return appposition.NewRiskResolver(&lockedRiskRepo{}, func() int64 { return *clock }).WithHistory(history)
}

func TestRiskResolver_RateLimitSperrtAllePositionenFuerCooldown(t *testing.T) {
	now := historyNow
	history := &scriptedHistory{err: &interfaces.RateLimitedError{Err: fmt.Errorf("429")}}
	resolver := newLimitedResolver(history, &now)
	ctx := context.Background()

	resolver.BeginRun([]string{"A", "B"})
	resolver.Resolve(ctx, positionWithID("A"), nil)
	resolver.Resolve(ctx, positionWithID("B"), nil)
	if history.calls.Load() != 1 {
		t.Fatalf("nach Rate-Limit keine weitere Abfrage, Aufrufe=%d", history.calls.Load())
	}
	now += 4 * 60 * 1000
	resolver.BeginRun([]string{"A", "B"})
	resolver.Resolve(ctx, positionWithID("B"), nil)
	if history.calls.Load() != 1 {
		t.Fatalf("Cooldown läuft noch, Aufrufe=%d", history.calls.Load())
	}
	now += 2 * 60 * 1000
	resolver.BeginRun([]string{"A", "B"})
	resolver.Resolve(ctx, positionWithID("B"), nil)
	if history.calls.Load() != 2 {
		t.Fatalf("Cooldown vorbei, Aufrufe=%d", history.calls.Load())
	}
}

func TestRiskResolver_BudgetHoechstensZweiAbfragenProDurchlauf(t *testing.T) {
	now := historyNow
	history := &scriptedHistory{}
	resolver := newLimitedResolver(history, &now)
	ctx := context.Background()
	ids := []string{"A", "B", "C"}

	resolver.BeginRun(ids)
	for _, id := range ids {
		if _, known := resolver.Resolve(ctx, positionWithID(id), nil); known {
			t.Errorf("%s sollte unbekannt sein", id)
		}
	}
	if history.calls.Load() != 2 {
		t.Fatalf("Aufrufe=%d, erwartet 2", history.calls.Load())
	}
	resolver.BeginRun(ids)
	resolver.Resolve(ctx, positionWithID("C"), nil)
	if history.calls.Load() != 3 {
		t.Fatalf("C im nächsten Durchlauf bedient, Aufrufe=%d", history.calls.Load())
	}
}

func TestRiskResolver_GleichzeitigeAbfragenDerselbenPositionWerdenZusammengefasst(t *testing.T) {
	now := historyNow
	history := &scriptedHistory{gate: make(chan struct{}), records: []position.StopOrderRecord{record(riskPositionID, "0.9000", 1)}}
	resolver := newLimitedResolver(history, &now)

	var group sync.WaitGroup
	results := make([]bool, 5)
	for index := range results {
		group.Add(1)
		go func() {
			defer group.Done()
			_, results[index] = resolver.Resolve(context.Background(), positionWithID(riskPositionID), nil)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(history.gate)
	group.Wait()
	if history.calls.Load() != 1 {
		t.Errorf("Aufrufe=%d, erwartet 1", history.calls.Load())
	}
}

func TestRiskResolver_BeginRunEntferntEintraegeGeschlossenerPositionen(t *testing.T) {
	now := historyNow
	history := &scriptedHistory{}
	resolver := newLimitedResolver(history, &now)
	ctx := context.Background()

	resolver.BeginRun([]string{"A"})
	resolver.Resolve(ctx, positionWithID("A"), nil) // Negativ-Cache für A
	resolver.BeginRun(nil)                          // A ist geschlossen
	resolver.BeginRun([]string{"A"})                // A taucht wieder auf -> frisch abfragen
	resolver.Resolve(ctx, positionWithID("A"), nil)
	if history.calls.Load() != 2 {
		t.Errorf("Aufrufe=%d, erwartet 2", history.calls.Load())
	}
}
