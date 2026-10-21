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

func TestWebUIIntegration_UnbekannteApiAnfragenBleibenNotFound(t *testing.T) {
	handler := newHandlerMitWebUI()
	tests := []struct{ name, method, target, accept string }{
		{"GET unbekannter Pfad ohne Accept html", http.MethodGet, "/unbekannt", "*/*"},
		{"POST auf unbekannten Pfad", http.MethodPost, "/unbekannt", "text/html"},
		{"POST auf Datei-Pfad", http.MethodPost, "/index.html", "*/*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newTestRequest(tt.method, tt.target, allowedTestHost)
			request.Header.Set("Accept", tt.accept)
			recorder := serve(handler, request)
			if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "not-found") {
				t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestWebUIIntegration_BrowserNavigationBekommtDasUI(t *testing.T) {
	handler := newHandlerMitWebUI()
	request := newTestRequest(http.MethodGet, "/journal", allowedTestHost)
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	if recorder := serve(handler, request); recorder.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 (SPA-Fallback)", recorder.Code)
	}
}

// Vite legt die gehashten Bundles unter /assets/ ab; nur /assets/icons gehört der API.
func TestWebUIIntegration_ViteAssetsKommenVomUI(t *testing.T) {
	settings := testSettings
	settings.WebUI = webui.NewHandler(fstest.MapFS{
		"index.html":           {Data: []byte("<html>UI</html>")},
		"assets/index-abc.js":  {Data: []byte("console.log('bundle')")},
		"assets/index-abc.css": {Data: []byte("body{}")},
	})
	handler := NewHandler(settings, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for target, want := range map[string]string{
		"/assets/index-abc.js":  "bundle",
		"/assets/index-abc.css": "body",
	} {
		t.Run(target, func(t *testing.T) {
			recorder := serve(handler, newTestRequest(http.MethodGet, target, allowedTestHost))
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), want) {
				t.Fatalf("Status %d, Body %q", recorder.Code, recorder.Body.String())
			}
		})
	}
	t.Run("/assets/icons bleibt API", func(t *testing.T) {
		recorder := serve(handler, newTestRequest(http.MethodGet, "/assets/icons", allowedTestHost))
		if strings.Contains(recorder.Body.String(), "UI") || strings.Contains(recorder.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("API-Route lieferte Web-UI: %d %q", recorder.Code, recorder.Body.String())
		}
	})
}
