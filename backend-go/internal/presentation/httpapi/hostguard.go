package httpapi

import (
	"net/http"
	"strings"
)

// withHostGuard schützt vor DNS-Rebinding: Jede Anfrage mit fremdem oder leerem Host-Header
// wird mit 403 forbidden-host abgewiesen, bevor Routing oder Handler laufen.
// Ein absolutes Request-Ziel wird ebenfalls abgewiesen, weil Go r.Host dann aus der URL setzt.
func withHostGuard(allowedHosts []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.IsAbs() || !isAllowedHost(r.Host, allowedHosts) {
			writeProblem(w, ProblemForbiddenHost)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isAllowedHost vergleicht den rohen Host-Header ohne Groß-/Kleinschreibung mit den erlaubten Einträgen.
func isAllowedHost(hostHeader string, allowedHosts []string) bool {
	if hostHeader == "" {
		return false
	}
	for _, allowedHost := range allowedHosts {
		if strings.EqualFold(allowedHost, hostHeader) {
			return true
		}
	}
	return false
}
