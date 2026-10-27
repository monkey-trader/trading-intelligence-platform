package position

import "math"

const (
	defaultClosedPositionLimit = 100
	maxClosedPositionLimit     = 500
)

// ClosedPositionQueryInput ist die rohe Eingabe; nil bedeutet "nicht angegeben".
type ClosedPositionQueryInput struct {
	From, To *int64
	Symbol   string
	Limit    *int
}

// ClosedPositionQuery ist die geprüfte Abfrage geschlossener Positionen.
type ClosedPositionQuery struct {
	from, to int64
	symbol   string
	limit    int
}

// NewClosedPositionQuery prüft Limit (1..500, Default 100) und from <= to.
func NewClosedPositionQuery(input ClosedPositionQueryInput) (ClosedPositionQuery, error) {
	query := ClosedPositionQuery{from: 0, to: math.MaxInt64, symbol: input.Symbol, limit: defaultClosedPositionLimit}
	if input.Limit != nil {
		if *input.Limit < 1 || *input.Limit > maxClosedPositionLimit {
			return ClosedPositionQuery{}, &InvalidLimitError{Value: *input.Limit}
		}
		query.limit = *input.Limit
	}
	if input.From != nil {
		query.from = *input.From
	}
	if input.To != nil {
		query.to = *input.To
	}
	if query.from > query.to {
		return ClosedPositionQuery{}, &InvalidTimeRangeError{From: query.from, To: query.to}
	}
	return query, nil
}

// From ist die untere Grenze (inklusiv), ohne Angabe 0.
func (query ClosedPositionQuery) From() int64 { return query.from }

// To ist die obere Grenze (inklusiv), ohne Angabe math.MaxInt64.
func (query ClosedPositionQuery) To() int64 { return query.to }

// Symbol ist der Filter, leer = alle.
func (query ClosedPositionQuery) Symbol() string { return query.symbol }

// Limit ist die maximale Anzahl.
func (query ClosedPositionQuery) Limit() int { return query.limit }
