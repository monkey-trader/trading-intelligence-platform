package position

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// State-Schlüssel (Schema wie IncomeScanUntilKey: snake_case, Epoch-ms, Symbol nach ':').
const (
	// PositionsScanUntilKeyPrefix + Symbol: Fortschritt des inkrementellen Positions-Imports.
	PositionsScanUntilKeyPrefix = "positions_scan_until:"
	// PositionsFullSearchKeyPrefix + Symbol: Symbol ist in der Vollsuche erledigt (Wert = Zeitpunkt).
	PositionsFullSearchKeyPrefix = "positions_full_search_done:"
)

// IncomeImporter ist der Income-Schritt (ImportIncomeUseCase).
type IncomeImporter interface {
	Execute(ctx context.Context, fullImportStart int64) (ImportIncomeResult, error)
}

// ClosedImporter ist der Positions-Schritt (ImportClosedPositionsUseCase).
type ClosedImporter interface {
	Execute(ctx context.Context, symbol string, from, to int64) (ImportClosedPositionsResult, error)
}

// SymbolFailure ist ein fehlgeschlagenes Symbol.
type SymbolFailure struct {
	Symbol string
	Err    error
}

// ImportFailuresError sammelt Symbol-Fehler eines Laufs (errors.As). Unwrap() []error liefert die Ursachen.
type ImportFailuresError struct{ Failures []SymbolFailure }

func (err *ImportFailuresError) Error() string {
	parts := make([]string, 0, len(err.Failures))
	for _, failure := range err.Failures {
		parts = append(parts, fmt.Sprintf("%s: %v", failure.Symbol, failure.Err))
	}
	return fmt.Sprintf("%d Symbol(e) fehlgeschlagen: %s", len(err.Failures), strings.Join(parts, "; "))
}

// Unwrap liefert die Ursachen für errors.Is/As.
func (err *ImportFailuresError) Unwrap() []error {
	causes := make([]error, 0, len(err.Failures))
	for _, failure := range err.Failures {
		causes = append(causes, failure.Err)
	}
	return causes
}

// ScheduledImportResult ist das Ergebnis eines Laufs.
type ScheduledImportResult struct {
	// Symbols sind die abgearbeiteten Symbole (Erfolg, Verdacht oder Fehler), in Reihenfolge.
	Symbols []string
	// Succeeded sind Symbole, deren State vorgerückt ist.
	Succeeded []string
	// Suspicious sind Symbole mit Verdacht (State nicht vorgerückt).
	Suspicious []string
	Failures   []SymbolFailure
}

// ScheduledImportDeps bündelt die Abhängigkeiten.
type ScheduledImportDeps struct {
	Income          IncomeImporter
	Positions       ClosedImporter
	Symbols         interfaces.IncomeSymbolLister
	Contracts       interfaces.ContractSource
	OpenSource      interfaces.OpenPositionSource
	OpenRecorder    interfaces.OpenPositionRecorder
	State           interfaces.ImportStateRepository
	Now             func() time.Time
	FullImportStart int64
}

// ScheduledImportUseCase orchestriert Income -> Symbole (aus DB) -> Positionen.
type ScheduledImportUseCase struct{ deps ScheduledImportDeps }

// NewScheduledImportUseCase baut den Use Case.
func NewScheduledImportUseCase(deps ScheduledImportDeps) *ScheduledImportUseCase {
	return &ScheduledImportUseCase{deps: deps}
}

// overlapMillis ist die Überlappung beim inkrementellen Lauf (1 h).
const overlapMillis = int64(time.Hour / time.Millisecond)

// RunIncremental führt aus: offene Positionen anlegen, Income, dann Positionen für alle Income-Symbole
// aus der DB sowie die Symbole der offenen Positionen (damit ihr Schließen sicher erfasst wird).
func (uc *ScheduledImportUseCase) RunIncremental(ctx context.Context) (ScheduledImportResult, error) {
	snapshot, err := uc.deps.OpenSource.FetchOpenPositions(ctx)
	if err != nil {
		return ScheduledImportResult{}, fmt.Errorf("offene positionen laden: %w", err)
	}
	if err := uc.deps.OpenRecorder.RecordOpenPositions(ctx, snapshot.Positions); err != nil {
		return ScheduledImportResult{}, fmt.Errorf("offene positionen anlegen: %w", err)
	}
	if _, err := uc.deps.Income.Execute(ctx, uc.deps.FullImportStart); err != nil {
		return ScheduledImportResult{}, fmt.Errorf("income importieren: %w", err)
	}
	symbols, err := uc.deps.Symbols.ListIncomeSymbols(ctx)
	if err != nil {
		return ScheduledImportResult{}, fmt.Errorf("income-symbole lesen: %w", err)
	}
	for _, open := range snapshot.Positions {
		symbols = append(symbols, open.Symbol)
	}
	slices.Sort(symbols)
	return uc.importSymbols(ctx, slices.Compact(symbols), false)
}

// RunFullSearch führt aus: Positionen für alle USDT-Kontrakte, fortsetzbar (State pro Symbol).
func (uc *ScheduledImportUseCase) RunFullSearch(ctx context.Context) (ScheduledImportResult, error) {
	contracts, err := uc.deps.Contracts.FetchContracts(ctx)
	if err != nil {
		return ScheduledImportResult{}, fmt.Errorf("kontrakte laden: %w", err)
	}
	pending := make([]string, 0, len(contracts))
	for _, symbol := range contracts {
		_, done, err := uc.deps.State.Get(ctx, PositionsFullSearchKeyPrefix+symbol)
		if err != nil {
			return ScheduledImportResult{}, fmt.Errorf("state für %s lesen: %w", symbol, err)
		}
		if !done {
			pending = append(pending, symbol)
		}
	}
	return uc.importSymbols(ctx, pending, true)
}

// RunFullSearchUntilDone wiederholt RunFullSearch nach retryDelay, bis kein Symbol mehr offen ist
// (z.B. nach "database is locked" oder Netzfehlern). Erledigte Symbole bleiben dank State außen vor.
// onFailure bekommt jeden fehlgeschlagenen Lauf. Endet mit nil bei Erfolg, sonst mit dem Kontextfehler.
func (uc *ScheduledImportUseCase) RunFullSearchUntilDone(ctx context.Context, retryDelay time.Duration, onFailure func(error)) error {
	for {
		_, err := uc.RunFullSearch(ctx)
		if err == nil {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		onFailure(err)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// importSymbols importiert Symbol für Symbol. Fehler eines Symbols brechen die anderen nicht ab;
// ein beendeter Kontext stoppt sofort.
func (uc *ScheduledImportUseCase) importSymbols(ctx context.Context, symbols []string, fullSearch bool) (ScheduledImportResult, error) {
	var result ScheduledImportResult
	now := uc.deps.Now().UnixMilli()
	for _, symbol := range symbols {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.Symbols = append(result.Symbols, symbol)
		from, err := uc.startFor(ctx, symbol, fullSearch, now)
		if err == nil {
			err = uc.importSymbol(ctx, symbol, from, now, fullSearch, &result)
		}
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return result, ctxErr
			}
			result.Failures = append(result.Failures, SymbolFailure{Symbol: symbol, Err: err})
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if len(result.Failures) > 0 {
		return result, &ImportFailuresError{Failures: result.Failures}
	}
	return result, nil
}

func (uc *ScheduledImportUseCase) startFor(ctx context.Context, symbol string, fullSearch bool, now int64) (int64, error) {
	if fullSearch {
		return uc.deps.FullImportStart, nil
	}
	saved, found, err := uc.deps.State.Get(ctx, PositionsScanUntilKeyPrefix+symbol)
	if err != nil {
		return 0, fmt.Errorf("state für %s lesen: %w", symbol, err)
	}
	if !found {
		return uc.deps.FullImportStart, nil
	}
	return min(max(saved-overlapMillis, uc.deps.FullImportStart), now), nil
}

func (uc *ScheduledImportUseCase) importSymbol(ctx context.Context, symbol string, from, now int64, fullSearch bool, result *ScheduledImportResult) error {
	imported, err := uc.deps.Positions.Execute(ctx, symbol, from, now)
	if err != nil {
		return err
	}
	// Nur die Vollsuche merkt sich "erledigt" dauerhaft; ein Abbruch mittendrin darf sie nicht
	// fälschlich abschliessen. Der inkrementelle Lauf wird ohnehin mit Überlappung wiederholt.
	if err := ctx.Err(); err != nil && fullSearch {
		return err // Vollsuche: abgebrochenes Symbol nicht als erledigt markieren
	}
	if len(imported.Suspicious) > 0 {
		result.Suspicious = append(result.Suspicious, symbol)
		return nil
	}
	if err := uc.deps.State.Set(ctx, PositionsScanUntilKeyPrefix+symbol, now); err != nil {
		return fmt.Errorf("state für %s speichern: %w", symbol, err)
	}
	if fullSearch {
		if err := uc.deps.State.Set(ctx, PositionsFullSearchKeyPrefix+symbol, now); err != nil {
			return fmt.Errorf("state für %s speichern: %w", symbol, err)
		}
	}
	result.Succeeded = append(result.Succeeded, symbol)
	return nil
}
