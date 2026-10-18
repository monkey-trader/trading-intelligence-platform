package position

import (
	"context"
	"errors"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

// openPositionsPath ist der BingX-Endpunkt der offenen Positionen.
const openPositionsPath = "/openApi/swap/v2/user/positions"

// BingXOpenPositionSource fragt die offenen Positionen live bei BingX ab.
type BingXOpenPositionSource struct {
	client bingx.GetClient
	clock  interfaces.Clock
}

// NewBingXOpenPositionSource baut die Source.
func NewBingXOpenPositionSource(client bingx.GetClient, clock interfaces.Clock) *BingXOpenPositionSource {
	return &BingXOpenPositionSource{client: client, clock: clock}
}

// FetchOpenPositions ruft GET /openApi/swap/v2/user/positions auf und übersetzt Fehler.
func (source *BingXOpenPositionSource) FetchOpenPositions(ctx context.Context) (domainposition.OpenPositionsSnapshot, error) {
	data, err := source.client.Get(ctx, openPositionsPath, nil)
	if err != nil {
		return domainposition.OpenPositionsSnapshot{}, toUnavailable(err)
	}
	positions, err := mapOpenPositionsResponse(data)
	if err != nil {
		return domainposition.OpenPositionsSnapshot{}, toUnavailable(err)
	}
	// fetchedAt erst nach der erfolgreichen Antwort lesen.
	return domainposition.OpenPositionsSnapshot{Positions: positions, FetchedAt: source.clock.Now().UnixMilli()}, nil
}

// toUnavailable übersetzt bekannte Fehler in *OpenPositionsUnavailableError; unbekannte gehen unverändert durch.
func toUnavailable(err error) error {
	reason, known := classifyUnavailable(err)
	if !known {
		return err
	}
	return &domainposition.OpenPositionsUnavailableError{Reason: reason, Cause: err}
}

// classifyUnavailable ordnet einen Quellfehler einem Grund zu; known ist falsch bei unbekannten Fehlern.
func classifyUnavailable(err error) (reason domainposition.UnavailableReason, known bool) {
	var (
		rateLimitError  *bingx.RateLimitError
		networkError    *bingx.NetworkError
		apiError        *bingx.APIError
		httpError       *bingx.HTTPError
		unexpectedError *bingx.UnexpectedResponseError
		domainError     domainposition.PositionDomainError
	)
	switch {
	case errors.As(err, &rateLimitError):
		return domainposition.ReasonRateLimited, true
	case errors.As(err, &networkError):
		return domainposition.ReasonUnreachable, true
	case errors.As(err, &apiError), errors.As(err, &httpError):
		return domainposition.ReasonUpstreamError, true
	case errors.As(err, &unexpectedError), errors.As(err, &domainError):
		return domainposition.ReasonInvalidResponse, true
	}
	return "", false
}
