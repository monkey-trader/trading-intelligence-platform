package position

import (
	"context"
	"slices"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

// IncomeScanUntilKey ist der State-Schlüssel des Income-Fortschritts.
const IncomeScanUntilKey = "income_scan_until"

// IncomeHistoryFromKey merkt, ab wann die Income-Historie nachgeladen wurde (unabhängig vom Fortschritt).
const IncomeHistoryFromKey = "income_history_from"

const (
	incomeWindowMillis = int64(7 * 24 * time.Hour / time.Millisecond)
)

// ImportIncomeResult ist das Ergebnis eines Income-Laufs.
type ImportIncomeResult struct {
	Imported int
	// Symbols sind die Symbole mit REALIZED_PNL, eindeutig und sortiert.
	Symbols []string
}

// ImportIncomeUseCase importiert Income in 7-Tage-Fenstern.
type ImportIncomeUseCase struct {
	source     interfaces.IncomeSource
	repository interfaces.IncomeRepository
	state      interfaces.ImportStateRepository
	now        func() time.Time
}

// NewImportIncomeUseCase baut den Use Case.
func NewImportIncomeUseCase(source interfaces.IncomeSource, repository interfaces.IncomeRepository, state interfaces.ImportStateRepository, now func() time.Time) *ImportIncomeUseCase {
	return &ImportIncomeUseCase{source: source, repository: repository, state: state, now: now}
}

// incomeOverlapMillis ist die Überlappung zum gespeicherten Fortschritt.
const incomeOverlapMillis = int64(time.Hour / time.Millisecond)

// Execute importiert ab State-1h (oder fullImportStart) bis jetzt in 7-Tage-Fenstern.
// Der State rückt erst nach dem letzten erfolgreichen Fenster vor.
func (uc *ImportIncomeUseCase) Execute(ctx context.Context, fullImportStart int64) (ImportIncomeResult, error) {
	if err := ctx.Err(); err != nil {
		return ImportIncomeResult{}, err
	}
	start := fullImportStart
	saved, found, err := uc.state.Get(ctx, IncomeScanUntilKey)
	if err != nil {
		return ImportIncomeResult{}, err
	}
	if found {
		start = saved - incomeOverlapMillis
	}
	end := uc.now().UnixMilli()

	result := ImportIncomeResult{}
	pnlSymbols := map[string]struct{}{}
	for windowStart := start; windowStart < end; {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		windowEnd := min(windowStart+incomeWindowMillis, end)
		stored, err := uc.importWindow(ctx, windowStart, windowEnd, pnlSymbols)
		result.Imported += stored
		if err != nil {
			return result, err
		}
		windowStart = windowEnd
	}
	for symbol := range pnlSymbols {
		result.Symbols = append(result.Symbols, symbol)
	}
	slices.Sort(result.Symbols)
	if err := uc.state.Set(ctx, IncomeScanUntilKey, end); err != nil {
		return result, err
	}
	return result, nil
}

// importWindow lädt ein Fenster, entfernt tranId-Duplikate und speichert es als ein Batch.
func (uc *ImportIncomeUseCase) importWindow(ctx context.Context, from, to int64, pnlSymbols map[string]struct{}) (int, error) {
	fetched, err := uc.source.FetchIncome(ctx, from, to)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(fetched))
	unique := make([]position.Income, 0, len(fetched))
	for _, entry := range fetched {
		if _, duplicate := seen[entry.TranID]; duplicate {
			continue
		}
		seen[entry.TranID] = struct{}{}
		unique = append(unique, entry)
		if entry.IncomeType == position.IncomeTypeRealizedPnl {
			pnlSymbols[entry.Symbol] = struct{}{}
		}
	}
	if err := uc.repository.UpsertIncomeBatch(ctx, unique); err != nil {
		return 0, err
	}
	return len(unique), nil
}

// EnsureHistorySince lädt einmalig das Income ab since (Epoch-ms) bis jetzt nach, damit Summen über
// frühere Zeiträume (z.B. der Monats-PnL) vollständig sind. Der Lauf ist wiederholbar (Upsert über tranId),
// berührt den Fortschritt des laufenden Imports nicht und merkt sich nur nach Erfolg.
func (uc *ImportIncomeUseCase) EnsureHistorySince(ctx context.Context, since int64) error {
	loaded, found, err := uc.state.Get(ctx, IncomeHistoryFromKey)
	if err != nil {
		return err
	}
	if found && loaded <= since {
		return nil
	}
	end := uc.now().UnixMilli()
	ignoredSymbols := map[string]struct{}{}
	for windowStart := since; windowStart < end; {
		if err := ctx.Err(); err != nil {
			return err
		}
		windowEnd := min(windowStart+incomeWindowMillis, end)
		if _, err := uc.importWindow(ctx, windowStart, windowEnd, ignoredSymbols); err != nil {
			return err
		}
		windowStart = windowEnd
	}
	return uc.state.Set(ctx, IncomeHistoryFromKey, since)
}
