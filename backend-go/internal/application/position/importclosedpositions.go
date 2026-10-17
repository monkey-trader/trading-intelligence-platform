package position

import (
	"context"
	"log/slog"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

// windowMillis ist die maximale Fenstergröße in Millisekunden (aus der Domain-Konstante).
const windowMillis = int64(position.MaxQueryRange / time.Millisecond)

// ImportClosedPositionsResult ist das Ergebnis eines Imports.
type ImportClosedPositionsResult struct {
	Imported   int
	Suspicious []ImportWindowResult
}

// ImportWindowResult ist das Ergebnis für ein Zeitfenster.
type ImportWindowResult struct {
	Symbol   string
	From     int64
	To       int64
	RowCount int
	// Suspicious ist wahr, wenn die Quelle leere Ergebnisse lieferte, obwohl Income vorhanden war.
	Suspicious bool
}

// ImportClosedPositionsUseCase importiert geschlossene Positionen in 89-Tage-Fenstern.
type ImportClosedPositionsUseCase struct {
	source       interfaces.ClosedPositionSource
	repository   interfaces.PositionRepository
	incomeLookup interfaces.IncomeLookup
}

// NewImportClosedPositionsUseCase baut den Use Case.
func NewImportClosedPositionsUseCase(
	source interfaces.ClosedPositionSource,
	repository interfaces.PositionRepository,
	incomeLookup interfaces.IncomeLookup,
) *ImportClosedPositionsUseCase {
	return &ImportClosedPositionsUseCase{
		source:       source,
		repository:   repository,
		incomeLookup: incomeLookup,
	}
}

// Execute importiert geschlossene Positionen für ein Symbol im Zeitbereich [from, to].
// Die Fenster sind höchstens 89 Tage lang, schließen den Endpunkt ein und überlappen sich
// deshalb um 1 ms; der Upsert ist idempotent. Imported zählt Upserts, nicht neue Zeilen.
// Verdacht wird über den Gesamtzeitraum geprüft: Wurden über alle Fenster 0 Positionen
// importiert, obwohl Income vorhanden ist, gibt es genau einen Suspicious-Eintrag (kein Fehler).
// Bei einem Fehler bricht der Import ab; das Ergebnis enthält die bis dahin importierte Anzahl.
func (uc *ImportClosedPositionsUseCase) Execute(ctx context.Context, symbol string, from, to int64) (ImportClosedPositionsResult, error) {
	if from > to {
		return ImportClosedPositionsResult{}, &position.InvalidQueryRangeError{From: from, To: to}
	}
	result := ImportClosedPositionsResult{}
	windowStart := from
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		windowEnd := min(windowStart+windowMillis, to)
		stored, err := uc.importWindow(ctx, symbol, windowStart, windowEnd)
		result.Imported += stored
		if err != nil {
			return result, err
		}
		if windowEnd >= to {
			break
		}
		windowStart = windowEnd
	}
	if result.Imported > 0 {
		return result, nil
	}
	hasIncome, err := uc.incomeLookup.HasIncomeForSymbol(ctx, symbol, from, to)
	if err != nil {
		return result, err
	}
	if hasIncome {
		slog.Warn("positionHistory leer, obwohl Income vorhanden ist", "symbol", symbol, "from", from, "to", to)
		result.Suspicious = append(result.Suspicious, ImportWindowResult{
			Symbol: symbol, From: from, To: to, RowCount: 0, Suspicious: true,
		})
	}
	return result, nil
}

// importWindow importiert ein Fenster und liefert die Anzahl gespeicherter Zeilen.
func (uc *ImportClosedPositionsUseCase) importWindow(ctx context.Context, symbol string, from, to int64) (int, error) {
	positions, err := uc.source.FetchClosedPositions(ctx, symbol, from, to)
	if err != nil {
		return 0, err
	}
	stored := 0
	for _, closedPosition := range positions {
		if err := uc.repository.UpsertClosedPosition(ctx, closedPosition); err != nil {
			return stored, err
		}
		stored++
	}
	return stored, nil
}
