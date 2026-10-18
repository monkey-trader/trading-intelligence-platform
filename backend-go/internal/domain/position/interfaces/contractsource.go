package interfaces

import "context"

// ContractSource liefert alle handelbaren USDT-Kontrakt-Symbole (dedupliziert, sortiert).
type ContractSource interface {
	FetchContracts(ctx context.Context) ([]string, error)
}
