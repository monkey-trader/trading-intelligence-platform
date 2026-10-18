package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// rateLimitRetryAfterSeconds entspricht dem RateLimitHold des Caches.
const rateLimitRetryAfterSeconds = "30"

// newOpenPositionsHandler beantwortet GET /positions/open über den Use Case.
// Die Fehlerursache landet nur im Log, nie im Body.
func newOpenPositionsHandler(
	getOpenPositions func(ctx context.Context) (appposition.OpenPositionsOverview, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overview, err := getOpenPositions(r.Context())
		if err != nil {
			problemType := problemTypeFor(err)
			level := slog.LevelWarn
			if errors.Is(err, context.Canceled) {
				// Der Client hat selbst abgebrochen: kein Betriebsproblem.
				level = slog.LevelInfo
			}
			logger.Log(r.Context(), level, "Offene Positionen nicht verfügbar",
				slog.String("problem", string(problemType)),
				slog.String("error", err.Error()),
				slog.String("cause", causeText(err)))
			if problemType == ProblemRateLimited {
				w.Header().Set("Retry-After", rateLimitRetryAfterSeconds)
			}
			writeProblem(w, problemType)
			return
		}

		// json.Marshal hängt keinen Zeilenumbruch an.
		body, err := json.Marshal(toOpenPositionsSnapshotDto(overview))
		if err != nil {
			logger.Error("Snapshot konnte nicht serialisiert werden", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// problemTypeFor übersetzt *OpenPositionsUnavailableError in den Problem-Typ; alles andere ist internal.
func problemTypeFor(err error) ProblemType {
	var unavailable *position.OpenPositionsUnavailableError
	if !errors.As(err, &unavailable) {
		return ProblemInternal
	}
	return problemTypeForReason(unavailable.Reason)
}

// problemTypeForReason übersetzt den Nicht-Verfügbar-Grund in den Problem-Typ.
func problemTypeForReason(reason position.UnavailableReason) ProblemType {
	switch reason {
	case position.ReasonNotConfigured:
		return ProblemNotConfigured
	case position.ReasonRateLimited:
		return ProblemRateLimited
	case position.ReasonUnreachable:
		return ProblemUnreachable
	case position.ReasonUpstreamError:
		return ProblemUpstreamError
	case position.ReasonInvalidResponse:
		return ProblemInvalidResponse
	default:
		return ProblemInternal
	}
}

// causeText liefert die Ursache hinter *OpenPositionsUnavailableError fürs Log ("" ohne Ursache).
// Die Adapter-Fehler enthalten nie Key, Secret oder die signierte URL.
func causeText(err error) string {
	var unavailable *position.OpenPositionsUnavailableError
	if errors.As(err, &unavailable) && unavailable.Cause != nil {
		return truncateForLog(unavailable.Cause.Error())
	}
	return ""
}

// maxCauseRunes begrenzt den Cause-Text im Log.
const maxCauseRunes = 200

// truncateForLog kürzt text auf höchstens maxCauseRunes Zeichen plus „…“.
func truncateForLog(text string) string {
	runes := []rune(text)
	if len(runes) <= maxCauseRunes {
		return text
	}
	return string(runes[:maxCauseRunes]) + "…"
}
