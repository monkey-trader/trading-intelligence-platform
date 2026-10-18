package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestWebUIIntegration_ApiPraefixeBleibenNotFound(t *testing.T) {
	handler := newHandlerMitWebUI()
	for _, target := range []string{"/health/x", "/positions", "/positions/unbekannt", "/api", "/api/positions/open"} {
		t.Run(target, func(t *testing.T) {
			recorder := serve(handler, newTestRequest(http.MethodGet, target, allowedTestHost))
			if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "not-found") {
				t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
			}
		})
	}
}
