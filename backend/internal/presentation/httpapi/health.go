package httpapi

import "net/http"

const healthBody = `{"status":"ok"}`

// handleHealth beantwortet GET /health mit 200 und {"status":"ok"}.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(healthBody))
}
