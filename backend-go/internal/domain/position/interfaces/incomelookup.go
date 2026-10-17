package interfaces

import (
	"context"
)

// IncomeLookup prüft, ob es Einträge in der Income-Tabelle für ein Symbol und Zeitbereich gibt.
type IncomeLookup interface {
	// HasIncomeForSymbol prüft, ob es Income-Einträge im Bereich [from, to] für das Symbol gibt.
	HasIncomeForSymbol(ctx context.Context, symbol string, from, to int64) (bool, error)
}
