package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

const swingLongPathSuffix = "/swing-long"

// swingLongPathSegment liefert die positionId aus /positions/{positionId}/swing-long.
func swingLongPathSegment(urlPath string) (string, bool) {
	rest, hasPrefix := strings.CutPrefix(urlPath, chartPathPrefix)
	if !hasPrefix {
		return "", false
	}
	segment, hasSuffix := strings.CutSuffix(rest, swingLongPathSuffix)
	if !hasSuffix || segment == "" || strings.Contains(segment, "/") {
		return "", false
	}
	return segment, true
}

// newSwingLongHandler bedient PUT und DELETE (jeweils 204, idempotent); die Fachlogik liegt in den Use Cases.
func newSwingLongHandler(
	set, clearFlag func(ctx context.Context, positionID string) error,
	allowedHosts []string,
	logger *slog.Logger,
) http.HandlerFunc {
	allowedOrigins := originsFor(allowedHosts)
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedOrigin(r.Header, allowedOrigins) {
			writeProblem(w, ProblemForbiddenOrigin)
			return
		}
		positionID, _ := swingLongPathSegment(r.URL.Path)
		if !positionIDPattern.MatchString(positionID) {
			writeProblem(w, ProblemBadRequest)
			return
		}
		action := set
		if r.Method == http.MethodDelete {
			action = clearFlag
		}
		if err := action(r.Context(), positionID); err != nil {
			logger.Error("Swing-Long-Flag nicht änderbar", slog.String("error", truncateForLog(err.Error())))
			writeProblem(w, problemTypeFor(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
