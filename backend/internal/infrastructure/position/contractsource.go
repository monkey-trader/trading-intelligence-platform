package position

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

const (
	contractsPath     = "/openApi/swap/v2/quote/contracts"
	contractsEndpoint = "quote/contracts"
	usdtCurrency      = "USDT"
)

// BingXContractSource liest die USDT-Kontrakte über GET /openApi/swap/v2/quote/contracts.
type BingXContractSource struct{ client bingx.GetClient }

// NewBingXContractSource baut die Source.
func NewBingXContractSource(client bingx.GetClient) *BingXContractSource {
	return &BingXContractSource{client: client}
}

type rawContract struct {
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
}

// FetchContracts liefert alle USDT-Symbole, dedupliziert und sortiert.
func (source *BingXContractSource) FetchContracts(ctx context.Context) ([]string, error) {
	data, err := source.client.Get(ctx, contractsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("kontrakte abrufen: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	if string(trimmed) == "null" {
		return []string{}, nil
	}
	var contracts []rawContract
	if len(trimmed) == 0 || trimmed[0] != '[' || json.Unmarshal(trimmed, &contracts) != nil {
		return nil, unexpected("Antwort von " + contractsEndpoint + " ist kein Array.")
	}
	symbols := []string{}
	for _, contract := range contracts {
		if contract.Symbol != "" && contract.Currency == usdtCurrency {
			symbols = append(symbols, contract.Symbol)
		}
	}
	slices.Sort(symbols)
	return slices.Compact(symbols), nil
}
