package httpapi

import "net/http"

// handleOpenPositions ist bis Phase 2 ein Stub: Die BingX-Anbindung fehlt noch, deshalb
// immer 503 not-configured (ohne Retry-After).
func handleOpenPositions(w http.ResponseWriter, _ *http.Request) {
	writeProblem(w, ProblemNotConfigured)
}
