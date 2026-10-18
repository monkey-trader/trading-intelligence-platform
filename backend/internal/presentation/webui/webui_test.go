package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const indexHTML = "<!doctype html><title>TIP</title>"

func testFiles() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte(indexHTML)},
		"assets/app.js": {Data: []byte("console.log(1)")},
		"assets/a.css":  {Data: []byte("body{}")},
	}
}

func do(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

func TestHandler_Ausliefern(t *testing.T) {
	handler := NewHandler(testFiles())
	tests := []struct {
		name, target, wantBody, wantTypePrefix string
		wantStatus                             int
	}{
		{"Root", "/", indexHTML, "text/html", 200},
		{"index.html", "/index.html", indexHTML, "text/html", 200},
		{"JS ohne Fallback", "/assets/app.js", "console.log(1)", "text/javascript", 200},
		{"CSS", "/assets/a.css", "body{}", "text/css", 200},
		{"SPA-Fallback", "/journal", indexHTML, "text/html", 200},
		{"SPA-Fallback tief", "/journal/42", indexHTML, "text/html", 200},
		{"fehlende Datei mit Endung", "/assets/missing.js", "", "", 404},
		{"fehlende Datei im Root", "/missing.png", "", "", 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := do(handler, http.MethodGet, tc.target)
			if recorder.Code != tc.wantStatus {
				t.Fatalf("Status = %d, erwartet %d", recorder.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && recorder.Body.String() != tc.wantBody {
				t.Errorf("Body = %q, erwartet %q", recorder.Body.String(), tc.wantBody)
			}
			if tc.wantTypePrefix != "" && !strings.HasPrefix(recorder.Header().Get("Content-Type"), tc.wantTypePrefix) {
				t.Errorf("Content-Type = %q, erwartet Präfix %q", recorder.Header().Get("Content-Type"), tc.wantTypePrefix)
			}
		})
	}
}

func TestHandler_Head(t *testing.T) {
	recorder := do(NewHandler(testFiles()), http.MethodHead, "/")
	if recorder.Code != http.StatusOK {
		t.Fatalf("HEAD / = %d, erwartet 200", recorder.Code)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("HEAD darf keinen Body liefern, war %q", recorder.Body.String())
	}
}

func TestHandler_NurGetUndHead(t *testing.T) {
	handler := NewHandler(testFiles())
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			recorder := do(handler, method, "/")
			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s / = %d, erwartet 405", method, recorder.Code)
			}
			if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, "GET") || !strings.Contains(allow, "HEAD") {
				t.Errorf("Allow = %q, erwartet GET und HEAD", allow)
			}
		})
	}
}

func TestHandler_LeeresDist(t *testing.T) {
	// Nur .gitkeep, kein index.html: klare 404-Meldung, kein Panic.
	handler := NewHandler(fstest.MapFS{".gitkeep": {Data: nil}})
	for _, target := range []string{"/", "/index.html", "/journal"} {
		t.Run(target, func(t *testing.T) {
			recorder := do(handler, http.MethodGet, target)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("Status = %d, erwartet 404", recorder.Code)
			}
			if !strings.Contains(strings.ToLower(recorder.Body.String()), "web-ui") {
				t.Errorf("Meldung muss 'Web-UI' erwähnen, war %q", recorder.Body.String())
			}
		})
	}
}

func TestHandler_CacheControl(t *testing.T) {
	handler := NewHandler(testFiles())
	tests := []struct{ name, target, want string }{
		{"index.html wird immer neu validiert", "/", "no-cache"},
		{"SPA-Fallback wird immer neu validiert", "/journal", "no-cache"},
		{"gehashte Assets sind unveränderlich", "/assets/app.js", "public, max-age=31536000, immutable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := do(handler, http.MethodGet, tt.target).Header().Get("Cache-Control"); got != tt.want {
				t.Fatalf("Cache-Control = %q, erwartet %q", got, tt.want)
			}
		})
	}
}
