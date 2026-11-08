package position_test

import (
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Anzeigename und Basis-Asset stammen aus domain/asset (ohne externen Aufruf, wie in der App).
var assetLabelCases = []struct{ symbol, baseAsset, displayName string }{
	{"NCCOGOLD2USD-USDT", "XAU", "Gold"},
	{"NCCOXAG2USD-USDT", "XAG", "Silber"},
	{"NCCO1OILWTI2USD-USDT", "1OILWTI", "1oilwti"},
	{"BTC-USDT", "BTC", "BTC"},
	{"sui-usdt", "SUI", "SUI"},
}

func TestExecute_OffenePositionTraegtBaseAssetUndDisplayName(t *testing.T) {
	for _, tc := range assetLabelCases {
		overview := execute(t, sourceWith(openPosition(func(p *position.OpenPosition) { p.Symbol = tc.symbol })))
		entry := overview.Positions[0]
		if entry.BaseAsset != tc.baseAsset || entry.DisplayName != tc.displayName {
			t.Errorf("%s: baseAsset=%q displayName=%q, erwartet %q/%q", tc.symbol, entry.BaseAsset, entry.DisplayName, tc.baseAsset, tc.displayName)
		}
	}
}

func TestExecute_UngueltigesSymbolFaelltAufDasSymbolZurueck(t *testing.T) {
	overview := execute(t, sourceWith(openPosition(func(p *position.OpenPosition) { p.Symbol = "kaputt symbol!" })))
	entry := overview.Positions[0]
	if entry.BaseAsset != "kaputt symbol!" || entry.DisplayName != "kaputt symbol!" {
		t.Errorf("baseAsset=%q displayName=%q", entry.BaseAsset, entry.DisplayName)
	}
}

func TestGetClosedPositions_TraegtBaseAssetUndDisplayName(t *testing.T) {
	for _, tc := range assetLabelCases {
		closed := closedOf("1", position.Long, "5")
		closed.Symbol = tc.symbol
		views := runClosed(t, []position.ClosedPosition{closed}, nil)
		if views[0].BaseAsset != tc.baseAsset || views[0].DisplayName != tc.displayName {
			t.Errorf("%s: baseAsset=%q displayName=%q", tc.symbol, views[0].BaseAsset, views[0].DisplayName)
		}
	}
}
