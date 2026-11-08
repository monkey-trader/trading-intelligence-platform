package price

import (
	"context"
	"math/rand/v2"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	priceinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/price/interfaces"
	sharedinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
)

const (
	// PollInterval ist der Takt der Kursabfrage.
	PollInterval = 2 * time.Second
	// initialBackoff ist die erste Wartezeit nach einem Fehler; sie verdoppelt sich bis maxBackoff.
	initialBackoff = time.Second
	// maxBackoff begrenzt die Wartezeit nach Fehlern, auch mit Jitter.
	maxBackoff = 30 * time.Second
)

// PositionPrice ist der Kurs einer offenen Position (Abstände nil ohne bekanntes Risiko).
type PositionPrice struct {
	PositionID                 string
	Symbol                     string
	Price                      string
	Change24hPercent           string
	DistanceToBreakEvenPercent *string
	DistanceToTp1Percent       *string
}

// PriceSnapshot ist der Stand, der an die Abonnenten geht. Prices ist nie nil.
type PriceSnapshot struct { //nolint:revive // Name laut Vertrag (Tests, OpenAPI-Schema)
	Prices    []PositionPrice
	FetchedAt int64
	Stale     bool
}

// StreamPricesConfig bündelt die Abhängigkeiten.
type StreamPricesConfig struct {
	Source        priceinterfaces.PriceSource
	OpenPositions func(ctx context.Context) (appposition.OpenPositionsOverview, error)
	Broker        *Broker[PriceSnapshot]
	Clock         sharedinterfaces.Clock
	// Sleep wartet die Dauer ab; Fehler (Context beendet) beendet Run.
	Sleep func(ctx context.Context, wait time.Duration) error
	// Jitter liefert einen Zuschlag (0..base) auf die Backoff-Wartezeit.
	Jitter func(base time.Duration) time.Duration
}

// StreamPricesUseCase pollt die Kurse und publiziert PriceSnapshots.
type StreamPricesUseCase struct {
	config   StreamPricesConfig
	failures int
	last     *PriceSnapshot
}

// NewStreamPricesUseCase baut den Use Case.
func NewStreamPricesUseCase(config StreamPricesConfig) *StreamPricesUseCase {
	return &StreamPricesUseCase{config: config}
}

// Poll führt einen Takt aus (höchstens ein Source-Request), publiziert und liefert die Wartezeit bis zum nächsten.
func (useCase *StreamPricesUseCase) Poll(ctx context.Context) time.Duration {
	snapshot, err := useCase.fetchSnapshot(ctx)
	if err != nil {
		return useCase.publishStale()
	}
	useCase.failures = 0
	useCase.last = &snapshot
	useCase.config.Broker.Publish(snapshot)
	return PollInterval
}

func (useCase *StreamPricesUseCase) fetchSnapshot(ctx context.Context) (PriceSnapshot, error) {
	quotes, err := useCase.config.Source.Fetch(ctx)
	if err != nil {
		return PriceSnapshot{}, err
	}
	overview, err := useCase.config.OpenPositions(ctx)
	if err != nil {
		return PriceSnapshot{}, err
	}
	bySymbol := make(map[string]int, len(quotes))
	for index, quote := range quotes {
		bySymbol[quote.Symbol] = index
	}
	prices := make([]PositionPrice, 0, len(overview.Positions))
	for _, entry := range overview.Positions {
		index, found := bySymbol[entry.Symbol]
		if !found {
			continue
		}
		quote := quotes[index]
		prices = append(prices, PositionPrice{
			PositionID:                 entry.PositionID,
			Symbol:                     entry.Symbol,
			Price:                      quote.Price,
			Change24hPercent:           quote.Change24hPercent,
			DistanceToBreakEvenPercent: distanceToLevel(entry, quote.Price, breakEvenMultiple),
			DistanceToTp1Percent:       distanceToLevel(entry, quote.Price, takeProfit1Multiple),
		})
	}
	return PriceSnapshot{Prices: prices, FetchedAt: useCase.config.Clock.Now().UnixMilli()}, nil
}

// BE-Auslöser ist 2R, TP1 ist 3R.
const (
	breakEvenMultiple   = 2
	takeProfit1Multiple = 3
)

func distanceToLevel(entry appposition.OpenPositionEntry, price string, multiple int) *string {
	level, found := domainposition.LevelAt(entry.Levels, multiple)
	if !found {
		return nil
	}
	distance, err := domainposition.DistanceToLevelPercent(entry.Side, price, &level.Price)
	if err != nil {
		return nil
	}
	return distance
}

// publishStale wiederholt den letzten Stand als stale und liefert die Backoff-Wartezeit.
func (useCase *StreamPricesUseCase) publishStale() time.Duration {
	snapshot := PriceSnapshot{Prices: []PositionPrice{}}
	if useCase.last != nil {
		snapshot = *useCase.last
	}
	snapshot.Stale = true
	useCase.config.Broker.Publish(snapshot)

	base := initialBackoff << min(useCase.failures, 10)
	base = min(base, maxBackoff)
	useCase.failures++
	jitter := time.Duration(0)
	if useCase.config.Jitter != nil {
		jitter = useCase.config.Jitter(base)
	}
	return min(base+jitter, maxBackoff)
}

// DefaultJitter liefert einen Zufallszuschlag von 0 bis base/2.
func DefaultJitter(base time.Duration) time.Duration {
	if base <= 1 {
		return 0
	}
	return rand.N(base / 2) //nolint:gosec // Jitter braucht keine kryptografische Zufälligkeit
}

// Run pollt per Sleep, bis ctx endet.
func (useCase *StreamPricesUseCase) Run(ctx context.Context) {
	for ctx.Err() == nil {
		wait := useCase.Poll(ctx)
		if err := useCase.config.Sleep(ctx, wait); err != nil {
			return
		}
	}
}
