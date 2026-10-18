package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/presentation/webui"
)

// newHandlerMitWebUI baut die Kette mit eingehängtem Web-UI als Fallback für unbekannte Pfade.
func newHandlerMitWebUI() http.Handler {
	settings := testSettings
	settings.WebUI = webui.NewHandler(fstest.MapFS{"index.html": {Data: []byte("<html>UI</html>")}})
	return NewHandler(settings, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestWebUIIntegration(t *testing.T) {
	handler := newHandlerMitWebUI()

	t.Run("index.html mit erlaubtem Host", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodGet, "/index.html", allowedTestHost))
		if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "UI") {
			t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("fremder Host auf Web-UI ergibt 403 forbidden-host", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodGet, "/index.html", "evil.example"))
		if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "forbidden-host") {
			t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("/health bleibt API", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodGet, "/health", allowedTestHost))
		if recorder.Code != 200 || strings.Contains(recorder.Body.String(), "UI") {
			t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("/positions/open bleibt API", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))
		if strings.Contains(recorder.Body.String(), "UI") || recorder.Code == 200 && strings.Contains(recorder.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("API-Route lieferte Web-UI: %d %q", recorder.Code, recorder.Body.String())
		}
	})
	t.Run("POST auf API-Pfad wird nie index.html", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodPost, "/positions/open", allowedTestHost))
		if strings.Contains(recorder.Body.String(), "UI") {
			t.Fatalf("Body %q", recorder.Body.String())
		}
	})
}
