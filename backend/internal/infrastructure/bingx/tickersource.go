package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/price"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
	sharedinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
)

// tickerPath ist der öffentliche BingX-Endpunkt der 24h-Ticker (ohne symbol: alle Symbole).
const tickerPath = "/openApi/swap/v2/quote/ticker"

// TickerSource holt alle Kurse mit einem Request.
type TickerSource struct {
	client GetClient
	clock  sharedinterfaces.Clock
}

// NewTickerSource baut die Source; die Uhr liefert Quote.At.
func NewTickerSource(client GetClient, clock sharedinterfaces.Clock) *TickerSource {
	return &TickerSource{client: client, clock: clock}
}

// tickerEntry ist ein Ticker; Zahlen kommen als Strings, andere Typen machen den Eintrag ungültig.
type tickerEntry struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	PriceChangePercent string `json:"priceChangePercent"`
}

// Fetch holt alle Ticker mit einem Request. Ungültige Einträge werden übersprungen,
// Clientfehler und Nicht-Arrays ergeben price.ErrPriceUnavailable.
func (source *TickerSource) Fetch(ctx context.Context) ([]price.Quote, error) {
	data, err := source.client.Get(ctx, tickerPath, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", price.ErrPriceUnavailable, err)
	}
	if len(data) == 0 || string(data) == "null" {
		return []price.Quote{}, nil
	}
	var rawEntries []json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		return nil, fmt.Errorf("%w: Ticker-Antwort ist kein Array: %w", price.ErrPriceUnavailable, err)
	}
	now := source.clock.Now().UnixMilli()
	quotes := make([]price.Quote, 0, len(rawEntries))
	for _, raw := range rawEntries {
		var entry tickerEntry
		if json.Unmarshal(raw, &entry) != nil || entry.Symbol == "" || !isPositiveDecimal(entry.LastPrice) {
			continue
		}
		if _, err := decimal.Subtract(entry.PriceChangePercent, "0"); err != nil {
			continue
		}
		quotes = append(quotes, price.Quote{
			Symbol: entry.Symbol, Price: entry.LastPrice, Change24hPercent: entry.PriceChangePercent, At: now,
		})
	}
	return quotes, nil
}

func isPositiveDecimal(value string) bool {
	normalized, err := decimal.Subtract(value, "0")
	return err == nil && normalized != "0" && !strings.HasPrefix(normalized, "-")
}
