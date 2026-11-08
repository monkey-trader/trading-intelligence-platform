package position

import "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"

// assetLabel liefert Basis-Asset und Anzeigename eines Symbols aus domain/asset (kein externer Aufruf).
// Lässt sich kein Basis-Asset ableiten, gilt das Symbol selbst, damit die Antwort nie leere Namen enthält.
func assetLabel(symbol string) (baseAsset, displayName string) {
	base, err := asset.NewBaseAsset(symbol)
	if err != nil {
		return symbol, symbol
	}
	return base.String(), asset.DisplayName(base)
}
