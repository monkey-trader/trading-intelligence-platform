// Package asset enthält die Use Cases für Coin-Icons.
package asset

import (
	"context"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset/interfaces"
	shared "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
)

const (
	positiveTTL = 24 * time.Hour
	negativeTTL = 10 * time.Minute
)

// GetAssetIconsUseCase liefert Icons für Symbole, mit Cache und genau einem Batch-Aufruf.
type GetAssetIconsUseCase struct {
	source interfaces.IconSource
	cache  interfaces.IconCache
}

// NewGetAssetIconsUseCase baut den Use Case. Die Uhr gehört dem Cache, hier nur der Vollständigkeit halber.
func NewGetAssetIconsUseCase(source interfaces.IconSource, cache interfaces.IconCache, _ shared.Clock) *GetAssetIconsUseCase {
	return &GetAssetIconsUseCase{source: source, cache: cache}
}

// Execute liefert je eindeutigem Symbol einen Eintrag in Anfragereihenfolge. Quellenfehler ergeben
// Einträge ohne Icon, ohne Fehler und ohne Cache-Eintrag.
func (useCase *GetAssetIconsUseCase) Execute(ctx context.Context, list asset.SymbolList) ([]asset.CoinIcon, error) {
	result := make([]asset.CoinIcon, 0, len(list.Symbols()))
	resolved := map[asset.BaseAsset]*asset.IconURL{}
	var missing []asset.BaseAsset
	queued := map[asset.BaseAsset]bool{}
	seenSymbols := map[string]bool{}

	for _, symbol := range list.Symbols() {
		base, err := asset.NewBaseAsset(symbol)
		if err != nil {
			return nil, err
		}
		if !seenSymbols[symbol] {
			seenSymbols[symbol] = true
			result = append(result, asset.CoinIcon{Symbol: symbol, BaseAsset: base, DisplayName: asset.DisplayName(base)})
		}
		if asset.IsCommodity(base) || queued[base] {
			continue
		}
		queued[base] = true
		if icon, found := useCase.cache.Get(base); found {
			resolved[base] = icon
			continue
		}
		missing = append(missing, base)
	}

	if len(missing) > 0 {
		if icons, err := useCase.source.ResolveIcons(ctx, missing); err == nil {
			for _, base := range missing {
				if icon, found := icons[base]; found {
					resolved[base] = &icon
					useCase.cache.Put(base, &icon, positiveTTL)
				} else {
					useCase.cache.Put(base, nil, negativeTTL)
				}
			}
		}
	}

	for index := range result {
		if icon := resolved[result[index].BaseAsset]; icon != nil && !asset.IsCommodity(result[index].BaseAsset) {
			result[index].IconURL = icon
		}
	}
	return result, nil
}
