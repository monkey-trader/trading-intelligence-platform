package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	appchart "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
)

const (
	chartPathPrefix = "/positions/"
	chartPathSuffix = "/chart"
)

// positionIDPattern erlaubt 1 bis 20 Ziffern (BingX-IDs sind 19-stellig).
var positionIDPattern = regexp.MustCompile(`^\d{1,20}$`)

// chartPathSegment liefert das Segment zwischen /positions/ und /chart, wenn der Pfad dazu passt.
func chartPathSegment(urlPath string) (string, bool) {
	rest, hasPrefix := strings.CutPrefix(urlPath, chartPathPrefix)
	if !hasPrefix {
		return "", false
	}
	segment, hasSuffix := strings.CutSuffix(rest, chartPathSuffix)
	if !hasSuffix || segment == "" || strings.Contains(segment, "/") {
		return "", false
	}
	return segment, true
}

// parseChartQuery prüft Pfad und Query-Parameter; Fehler sind immer bad-request.
func parseChartQuery(r *http.Request, segment string) (appchart.Query, bool) {
	if !positionIDPattern.MatchString(segment) {
		return appchart.Query{}, false
	}
	values := r.URL.Query()
	for _, name := range []string{"interval", "limit"} {
		if len(values[name]) > 1 {
			return appchart.Query{}, false
		}
	}
	interval := chart.DefaultInterval
	if values.Has("interval") {
		parsed, err := chart.ParseCandleInterval(values.Get("interval"))
		if err != nil {
			return appchart.Query{}, false
		}
		interval = parsed
	}
	limit := chart.DefaultLimit
	if values.Has("limit") {
		number, err := strconv.Atoi(values.Get("limit"))
		if err != nil {
			return appchart.Query{}, false
		}
		limit, err = chart.ParseLimit(number)
		if err != nil {
			return appchart.Query{}, false
		}
	}
	return appchart.Query{PositionID: segment, Interval: interval, Limit: limit}, true
}

// newPositionChartHandler beantwortet GET /positions/{positionId}/chart über den Use Case.
// Fehlerursachen landen nur im Log, nie im Body.
func newPositionChartHandler(
	getPositionChart func(ctx context.Context, query appchart.Query) (appchart.PositionChart, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		segment, _ := chartPathSegment(r.URL.Path)
		query, valid := parseChartQuery(r, segment)
		if !valid {
			writeProblem(w, ProblemBadRequest)
			return
		}
		result, err := getPositionChart(r.Context(), query)
		if err != nil {
			problemType := chartProblemTypeFor(err)
			level := slog.LevelWarn
			if errors.Is(err, context.Canceled) {
				level = slog.LevelInfo
			}
			logger.Log(r.Context(), level, "Positions-Chart nicht verfügbar",
				slog.String("problem", string(problemType)),
				slog.String("error", truncateForLog(err.Error())))
			if problemType == ProblemRateLimited {
				w.Header().Set("Retry-After", rateLimitRetryAfterSeconds)
			}
			writeProblem(w, problemType)
			return
		}
		body, err := json.Marshal(toPositionChartDto(result))
		if err != nil {
			logger.Error("Chart konnte nicht serialisiert werden", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// chartProblemTypeFor übersetzt Fehler des Chart-Use-Cases in Problem-Typen; alles Unbekannte ist internal.
func chartProblemTypeFor(err error) ProblemType {
	var (
		notFound        *chart.PositionNotFoundError
		badInterval     *chart.UnsupportedIntervalError
		badLimit        *chart.LimitOutOfRangeError
		dataUnavailable *chart.DataUnavailableError
	)
	switch {
	case errors.As(err, &notFound):
		return ProblemNotFound
	case errors.As(err, &badInterval), errors.As(err, &badLimit):
		return ProblemBadRequest
	case errors.As(err, &dataUnavailable):
		return problemTypeForReason(dataUnavailable.Reason)
	}
	return problemTypeFor(err)
}
