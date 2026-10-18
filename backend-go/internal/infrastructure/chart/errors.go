// Package chart enthält die BingX-Adapter für Kerzen und Schutzorders.
package chart

import (
	"errors"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// BingX-Codes der Kerzen-Abfrage (gemessen am 2026-10-18, siehe fixtures/bingx/README.md).
const (
	klineParameterCode      = 109400
	klineSymbolNotFoundCode = 109425
)

// invalidResponse meldet eine Antwort, die nicht die erwartete Form hat.
func invalidResponse(message string) error {
	return &chart.DataUnavailableError{
		Reason: position.ReasonInvalidResponse,
		Cause:  &bingx.UnexpectedResponseError{Message: message},
	}
}

// toUnavailable übersetzt Client-Fehler in *DataUnavailableError; unbekannte Fehler gehen unverändert durch.
// Für Kerzen werden die Codes 109400 und 109425 zu fachlichen Ursachen.
func toUnavailable(err error, symbol string) error {
	var (
		rateLimitError  *bingx.RateLimitError
		networkError    *bingx.NetworkError
		apiError        *bingx.APIError
		httpError       *bingx.HTTPError
		unexpectedError *bingx.UnexpectedResponseError
	)
	switch {
	case errors.As(err, &rateLimitError):
		return &chart.DataUnavailableError{Reason: position.ReasonRateLimited, Cause: err}
	case errors.As(err, &networkError):
		return &chart.DataUnavailableError{Reason: position.ReasonUnreachable, Cause: err}
	case errors.As(err, &apiError):
		return &chart.DataUnavailableError{Reason: position.ReasonUpstreamError, Cause: klineCause(apiError, symbol, err)}
	case errors.As(err, &httpError):
		return &chart.DataUnavailableError{Reason: position.ReasonUpstreamError, Cause: err}
	case errors.As(err, &unexpectedError):
		return &chart.DataUnavailableError{Reason: position.ReasonInvalidResponse, Cause: err}
	}
	return err
}

// klineCause liefert bei bekannten Kerzen-Codes die fachliche Ursache, sonst den Originalfehler.
func klineCause(apiError *bingx.APIError, symbol string, original error) error {
	switch apiError.Code {
	case klineParameterCode:
		return &chart.KlineParameterError{Code: apiError.Code, Msg: apiError.Msg}
	case klineSymbolNotFoundCode:
		return &chart.SymbolNotFoundError{Symbol: symbol}
	}
	return original
}
