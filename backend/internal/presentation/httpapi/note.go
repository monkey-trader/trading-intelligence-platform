package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
)

const notePathSuffix = "/note"

// notePathSegment liefert das Segment zwischen /positions/ und /note, wenn der Pfad dazu passt.
func notePathSegment(urlPath string) (string, bool) {
	rest, hasPrefix := strings.CutPrefix(urlPath, chartPathPrefix)
	if !hasPrefix {
		return "", false
	}
	segment, hasSuffix := strings.CutSuffix(rest, notePathSuffix)
	if !hasSuffix || segment == "" || strings.Contains(segment, "/") {
		return "", false
	}
	return segment, true
}

// newNoteHandler beantwortet GET /positions/{positionId}/note; Fehlerursachen landen nur im Log.
func newNoteHandler(
	getNote func(ctx context.Context, positionID string) (journal.JournalNote, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		positionID, _ := notePathSegment(r.URL.Path)
		if !positionIDPattern.MatchString(positionID) {
			writeProblem(w, ProblemBadRequest)
			return
		}
		note, err := getNote(r.Context(), positionID)
		if err != nil {
			var notFound *journal.NoteNotFoundError
			if errors.As(err, &notFound) {
				writeProblem(w, ProblemNotFound)
				return
			}
			problemType := problemTypeFor(err)
			logger.Error("Notiz nicht lesbar", slog.String("problem", string(problemType)), slog.String("cause", truncateForLog(err.Error())))
			writeProblem(w, problemType)
			return
		}
		body, err := json.Marshal(toJournalNoteDto(note))
		if err != nil {
			logger.Error("Notiz nicht serialisierbar", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
