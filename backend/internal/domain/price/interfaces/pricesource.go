// Package interfaces enthält die Ports der Live-Kurse.
package interfaces

import (
	"context"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/price"
)

// PriceSource liefert die Kurse aller Symbole mit einem Abruf.
type PriceSource interface {
	Fetch(ctx context.Context) ([]price.Quote, error)
}
