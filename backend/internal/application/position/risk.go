package position

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// negativeCacheTTLMs: So lange wird eine erfolglose Historien-Suche nicht wiederholt (10 Minuten).
const negativeCacheTTLMs = int64(10 * 60 * 1000)

// errorBackoffMs: So lange wird nach einem Quellenfehler (z.B. Rate-Limit) dieselbe Position nicht erneut
// abgefragt, damit der Poll der Liste die BingX-Sperre nicht verlängert (1 Minute).
const errorBackoffMs = int64(60 * 1000)

// rateLimitCooldownMs: Nach einem Rate-Limit der Historie werden alle Positionen so lange nicht abgefragt (5 Minuten).
const rateLimitCooldownMs = int64(5 * 60 * 1000)

// maxHistoryQueriesPerRun: Höchstens so viele neue Historien-Abfragen pro Resolve-Durchlauf (Poll).
const maxHistoryQueriesPerRun = 2

// historyCall ist eine laufende Abfrage; Wartende bekommen dasselbe Ergebnis.
type historyCall struct {
	done   chan struct{}
	result *string
}

// RiskResolver liefert das ursprüngliche Risiko einer Position. Fehlt es und ist der aktuelle SL unbrauchbar
// (fehlt oder liegt nicht auf der Verlustseite), wird optional die allOrders-Historie befragt.
type RiskResolver struct {
	risks   interfaces.PositionRiskRepository
	now     func() int64
	history interfaces.StopOrderHistorySource // optional

	mutex         sync.Mutex
	retryAfter    map[string]int64 // positionId -> frühester nächster Versuch (Epoch-ms)
	cooldownUntil int64            // globale Sperre nach Rate-Limit (Epoch-ms)
	queriesInRun  int
	inFlight      map[string]*historyCall
}

// NewRiskResolver baut den Resolver ohne Historie.
func NewRiskResolver(risks interfaces.PositionRiskRepository, now func() int64) *RiskResolver {
	return &RiskResolver{risks: risks, now: now, retryAfter: make(map[string]int64), inFlight: make(map[string]*historyCall)}
}

// WithHistory aktiviert die Suche des ursprünglichen SL in der Order-Historie.
func (resolver *RiskResolver) WithHistory(history interfaces.StopOrderHistorySource) *RiskResolver {
	resolver.history = history
	return resolver
}

// BeginRun startet einen Durchlauf (Poll): Das Abfrage-Budget wird zurückgesetzt, Einträge für nicht mehr
// offene Positionen werden entfernt.
func (resolver *RiskResolver) BeginRun(openPositionIDs []string) {
	open := make(map[string]struct{}, len(openPositionIDs))
	for _, positionID := range openPositionIDs {
		open[positionID] = struct{}{}
	}
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()
	resolver.queriesInRun = 0
	for positionID := range resolver.retryAfter {
		if _, stillOpen := open[positionID]; !stillOpen {
			delete(resolver.retryAfter, positionID)
		}
	}
}

// ResolveRisk ist die Kurzform ohne Historie und ohne Negativ-Cache.
func ResolveRisk(
	ctx context.Context,
	risks interfaces.PositionRiskRepository,
	now func() int64,
	openPosition domainposition.OpenPosition,
	stopTriggers []string,
) (risk domainposition.PositionRisk, known bool) {
	return NewRiskResolver(risks, now).Resolve(ctx, openPosition, stopTriggers)
}

// Resolve liefert das gespeicherte Risiko oder legt es an. known=false heißt: Risiko unbekannt
// (Fehler werden geloggt, nie weitergereicht).
func (resolver *RiskResolver) Resolve(ctx context.Context, openPosition domainposition.OpenPosition, stopTriggers []string) (domainposition.PositionRisk, bool) {
	stored, found, err := resolver.risks.Get(ctx, openPosition.PositionID)
	if err != nil {
		slog.WarnContext(ctx, "Risiko nicht lesbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		return domainposition.PositionRisk{}, false
	}
	if found {
		return stored, true
	}
	stopLoss := currentValidStopLoss(openPosition, stopTriggers)
	if stopLoss == nil && resolver.history != nil {
		stopLoss = resolver.stopLossFromHistory(ctx, openPosition)
	}
	if stopLoss == nil {
		return domainposition.PositionRisk{}, false
	}
	created := domainposition.PositionRisk{
		PositionID:      openPosition.PositionID,
		EntryPrice:      openPosition.AvgOpenPrice,
		InitialStopLoss: *stopLoss,
		FirstSeenAt:     resolver.now(),
	}
	if _, err := resolver.risks.SaveIfAbsent(ctx, created); err != nil {
		slog.WarnContext(ctx, "Risiko nicht speicherbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		return domainposition.PositionRisk{}, false
	}
	return created, true
}

// currentValidStopLoss liefert den engsten aktuellen SL, wenn er auf der Verlustseite des Entry liegt.
func currentValidStopLoss(openPosition domainposition.OpenPosition, stopTriggers []string) *string {
	stopLoss, err := domainposition.TightestStopLoss(openPosition.Side, stopTriggers)
	if err != nil || stopLoss == nil {
		return nil
	}
	if _, err := domainposition.RiskDistance(openPosition.Side, openPosition.AvgOpenPrice, *stopLoss); err != nil {
		return nil
	}
	return stopLoss
}

// stopLossFromHistory sucht den frühesten Verlust-SL der Position. Pro Position läuft höchstens eine Abfrage
// gleichzeitig; Wartende bekommen deren Ergebnis.
func (resolver *RiskResolver) stopLossFromHistory(ctx context.Context, openPosition domainposition.OpenPosition) *string {
	positionID := openPosition.PositionID
	current := resolver.now()
	resolver.mutex.Lock()
	if running, busy := resolver.inFlight[positionID]; busy {
		resolver.mutex.Unlock()
		<-running.done
		return running.result
	}
	retryAfter, cached := resolver.retryAfter[positionID]
	if (cached && current <= retryAfter) || current <= resolver.cooldownUntil || resolver.queriesInRun >= maxHistoryQueriesPerRun {
		resolver.mutex.Unlock()
		return nil
	}
	resolver.queriesInRun++
	call := &historyCall{done: make(chan struct{})}
	resolver.inFlight[positionID] = call
	resolver.mutex.Unlock()

	call.result = resolver.queryHistory(ctx, openPosition, current)

	resolver.mutex.Lock()
	delete(resolver.inFlight, positionID)
	resolver.mutex.Unlock()
	close(call.done)
	return call.result
}

// queryHistory fragt die Historie ab; ohne Treffer greift der Negativ-Cache, bei Quellenfehlern ein kurzer
// Backoff, bei Rate-Limit der globale Cooldown.
func (resolver *RiskResolver) queryHistory(ctx context.Context, openPosition domainposition.OpenPosition, current int64) *string {
	records, err := resolver.history.FetchStopOrders(ctx, openPosition.Symbol, openPosition.OpenTime, current)
	if err != nil {
		slog.WarnContext(ctx, "Order-Historie nicht ladbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		resolver.mutex.Lock()
		var rateLimited *interfaces.RateLimitedError
		if errors.As(err, &rateLimited) {
			resolver.cooldownUntil = current + rateLimitCooldownMs
		} else {
			resolver.retryAfter[openPosition.PositionID] = current + errorBackoffMs
		}
		resolver.mutex.Unlock()
		return nil
	}
	own := make([]domainposition.StopOrderRecord, 0, len(records))
	for _, record := range records {
		if record.PositionID == openPosition.PositionID {
			own = append(own, record)
		}
	}
	stopLoss, err := domainposition.EarliestInitialStopLoss(openPosition.Side, openPosition.AvgOpenPrice, own)
	if err != nil {
		slog.WarnContext(ctx, "Ursprünglicher SL nicht berechenbar", slog.String("positionId", openPosition.PositionID), slog.String("error", err.Error()))
		return nil
	}
	if stopLoss == nil {
		resolver.mutex.Lock()
		resolver.retryAfter[openPosition.PositionID] = current + negativeCacheTTLMs
		resolver.mutex.Unlock()
	}
	return stopLoss
}
