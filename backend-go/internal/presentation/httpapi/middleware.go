package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
)

// malformedTargetKey ist der Context-Schlüssel für das Flag des Listener-Wrappers.
type malformedTargetKey struct{}

// withMalformedTargetFlag legt das Flag des Listener-Wrappers in den Context
// (genutzt von ConnContext und der malformedTarget-Middleware).
func withMalformedTargetFlag(ctx context.Context, malformedTarget *atomic.Bool) context.Context {
	return context.WithValue(ctx, malformedTargetKey{}, malformedTarget)
}

// isMalformedTarget meldet, ob der Listener-Wrapper das Request-Ziel ersetzt hat.
func isMalformedTarget(ctx context.Context) bool {
	malformedTarget, found := ctx.Value(malformedTargetKey{}).(*atomic.Bool)
	return found && malformedTarget != nil && malformedTarget.Load()
}

// withRecover beantwortet einen Panic mit 500 internal. Die Ursache steht nur im Log.
func withRecover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if err, isError := recovered.(error); isError && errors.Is(err, http.ErrAbortHandler) {
				// Bewusster Abbruch: net/http behandelt ihn selbst.
				panic(recovered)
			}
			logger.Error("Panic in einem HTTP-Handler",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("cause", fmt.Sprint(recovered)),
			)
			writeProblem(w, ProblemInternal)
		}()
		next.ServeHTTP(w, r)
	})
}

// withMalformedTarget beantwortet Anfragen mit 400 bad-request, deren Ziel der
// Listener-Wrapper als ungültig markiert hat.
func withMalformedTarget(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isMalformedTarget(r.Context()) {
			writeProblem(w, ProblemBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
