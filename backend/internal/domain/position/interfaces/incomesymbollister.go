package interfaces

import "context"

// IncomeSymbolLister liefert die Symbole mit REALIZED_PNL aus der Income-Tabelle (eindeutig, sortiert).
type IncomeSymbolLister interface {
	ListIncomeSymbols(ctx context.Context) ([]string, error)
}
