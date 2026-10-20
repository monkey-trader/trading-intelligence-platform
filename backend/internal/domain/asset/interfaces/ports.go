// Package interfaces enthält die Ports des Asset-Kerns.
package interfaces

import (
	"context"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

// IconSource löst Icons für Basis-Assets auf einmal (Batch) auf. Unbekannte fehlen in der Map.
type IconSource interface {
	ResolveIcons(ctx context.Context, bases []asset.BaseAsset) (map[asset.BaseAsset]asset.IconURL, error)
}

// IconCache merkt Icons; ein nil-Icon bedeutet „bekannt ohne Icon“.
type IconCache interface {
	Get(base asset.BaseAsset) (*asset.IconURL, bool)
	Put(base asset.BaseAsset, icon *asset.IconURL, ttl time.Duration)
}
