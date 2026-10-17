package position

import (
	"context"
	"errors"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
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
	var (
		rateLimitError  *bingx.RateLimitError
		networkError    *bingx.NetworkError
		apiError        *bingx.APIError
		httpError       *bingx.HTTPError
		unexpectedError *bingx.UnexpectedResponseError
		domainError     domainposition.PositionDomainError
	)
	var reason domainposition.UnavailableReason
	switch {
	case errors.As(err, &rateLimitError):
		reason = domainposition.ReasonRateLimited
	case errors.As(err, &networkError):
		reason = domainposition.ReasonUnreachable
	case errors.As(err, &apiError), errors.As(err, &httpError):
		reason = domainposition.ReasonUpstreamError
	case errors.As(err, &unexpectedError), errors.As(err, &domainError):
		reason = domainposition.ReasonInvalidResponse
	default:
		return err
	}
	return &domainposition.OpenPositionsUnavailableError{Reason: reason, Cause: err}
}
