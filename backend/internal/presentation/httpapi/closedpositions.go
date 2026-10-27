package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// parseClosedPositionQuery liest from, to, symbol und limit; jede Verletzung ist ein Fehler (400).
func parseClosedPositionQuery(values url.Values) (position.ClosedPositionQuery, error) {
	var input position.ClosedPositionQueryInput
	input.Symbol = values.Get("symbol")
	for name, target := range map[string]**int64{"from": &input.From, "to": &input.To} {
		if !values.Has(name) {
			continue
		}
		parsed, err := strconv.ParseInt(values.Get(name), 10, 64)
		if err != nil || parsed < 0 {
			return position.ClosedPositionQuery{}, errBadQueryParameter
		}
		*target = &parsed
	}
	if values.Has("limit") {
		parsed, err := strconv.Atoi(values.Get("limit"))
		if err != nil {
			return position.ClosedPositionQuery{}, errBadQueryParameter
		}
		input.Limit = &parsed
	}
	return position.NewClosedPositionQuery(input)
}

type badQueryParameterError struct{}

func (badQueryParameterError) Error() string { return "ungültiger Query-Parameter" }

var errBadQueryParameter = badQueryParameterError{}

// newClosedPositionsHandler beantwortet GET /positions/closed; die Fehlerursache landet nur im Log.
func newClosedPositionsHandler(
	getClosed func(ctx context.Context, query position.ClosedPositionQuery) ([]appposition.ClosedPositionView, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := parseClosedPositionQuery(r.URL.Query())
		if err != nil {
			writeProblem(w, ProblemBadRequest)
			return
		}
		views, err := getClosed(r.Context(), query)
		if err != nil {
			problemType := problemTypeFor(err)
			logger.Error("Geschlossene Positionen nicht lesbar", slog.String("problem", string(problemType)), slog.String("cause", err.Error()))
			writeProblem(w, problemType)
			return
		}
		body, err := json.Marshal(toClosedPositionsDto(views))
		if err != nil {
			logger.Error("Geschlossene Positionen nicht serialisierbar", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
