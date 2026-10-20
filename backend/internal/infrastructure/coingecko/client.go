// Package coingecko löst Coin-Icons über die öffentliche CoinGecko-API auf.
package coingecko

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

const (
	requestTimeout = 5 * time.Second
	maxBodyBytes   = 1 << 20
	userAgent      = "trading-intelligence-platform/1.0 (+coin-icons)"
)

// RateLimitedError meldet: CoinGecko antwortet mit HTTP 429.
type RateLimitedError struct{}

func (*RateLimitedError) Error() string { return "coingecko: rate limit erreicht (HTTP 429)" }

// Client ist der CoinGecko-Adapter für den IconSource-Port.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient baut einen Client für die angegebene Basis-URL.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: &http.Client{Timeout: requestTimeout}}
}

type marketEntry struct {
	Symbol string  `json:"symbol"`
	Image  *string `json:"image"`
	Rank   *int    `json:"market_cap_rank"`
}

// ResolveIcons fragt alle Basis-Assets in einem Aufruf ab. Je Symbol gewinnt der beste
// market_cap_rank; Treffer ohne Rang zählen schlechter, aber vor keinem Treffer.
func (client *Client) ResolveIcons(ctx context.Context, bases []asset.BaseAsset) (map[asset.BaseAsset]asset.IconURL, error) {
	icons := map[asset.BaseAsset]asset.IconURL{}
	if len(bases) == 0 {
		return icons, nil
	}
	symbols := make([]string, len(bases))
	wanted := make(map[string]asset.BaseAsset, len(bases))
	for index, base := range bases {
		lower := strings.ToLower(base.String())
		symbols[index] = lower
		wanted[lower] = base
	}
	query := url.Values{
		"vs_currency": {"usd"},
		"symbols":     {strings.Join(symbols, ",")},
		"order":       {"market_cap_desc"},
		"per_page":    {"250"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/v3/coins/markets?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("coingecko: request bauen: %w", err)
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("coingecko: request fehlgeschlagen: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, &RateLimitedError{}
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko: unerwarteter HTTP-Status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("coingecko: antwort lesen: %w", err)
	}
	if len(body) > maxBodyBytes {
		return nil, errors.New("coingecko: antwort größer als 1 MiB")
	}
	var entries []marketEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("coingecko: ungültiges JSON: %w", err)
	}

	bestRank := map[asset.BaseAsset]int{}
	const noRank = int(^uint(0) >> 1)
	for _, entry := range entries {
		base, isWanted := wanted[strings.ToLower(entry.Symbol)]
		if !isWanted || entry.Image == nil {
			continue
		}
		icon, err := asset.NewIconURL(*entry.Image)
		if err != nil {
			continue
		}
		rank := noRank
		if entry.Rank != nil {
			rank = *entry.Rank
		}
		if previous, exists := bestRank[base]; exists && previous <= rank {
			continue
		}
		bestRank[base] = rank
		icons[base] = icon
	}
	return icons, nil
}
