package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const allowedTestHost = "127.0.0.1:3000"

var testSettings = Settings{AllowedHosts: []string{"127.0.0.1:3000", "localhost:3000", "[::1]:3000"}}

// trackingBody meldet, ob jemand den Request-Body gelesen hat.
type trackingBody struct {
	wasRead atomic.Bool
}

func (body *trackingBody) Read([]byte) (int, error) {
	body.wasRead.Store(true)
	return 0, io.ErrUnexpectedEOF
}

func (body *trackingBody) Close() error { return nil }

func newTestRequest(method, target, host string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.Host = host
	return request
}

func serve(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestHealth(t *testing.T) {
	for _, target := range []string{"/health", "/health?x=1"} {
		t.Run(target, func(t *testing.T) {
			recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, target, allowedTestHost))

			if recorder.Code != http.StatusOK {
				t.Errorf("Status = %d, erwartet 200", recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", contentType)
			}
			if body := recorder.Body.String(); body != `{"status":"ok"}` {
				t.Errorf("Body = %q, erwartet {\"status\":\"ok\"}", body)
			}
		})
	}
}

func TestOpenPositions_StubLiefertNotConfigured(t *testing.T) {
	recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))

	assertProblem(t, recorder, http.StatusServiceUnavailable, notConfiguredBody)
	if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "" {
		t.Errorf("Retry-After = %q, bei not-configured nicht erlaubt", retryAfter)
	}
}

func TestHostGuard(t *testing.T) {
	testCases := []struct {
		name           string
		host           string
		expectedStatus int
	}{
		{name: "127.0.0.1 erlaubt", host: "127.0.0.1:3000", expectedStatus: http.StatusOK},
		{name: "localhost erlaubt", host: "localhost:3000", expectedStatus: http.StatusOK},
		{name: "IPv6-Loopback erlaubt", host: "[::1]:3000", expectedStatus: http.StatusOK},
		{name: "Großschreibung erlaubt", host: "LOCALHOST:3000", expectedStatus: http.StatusOK},
		{name: "gemischte Schreibung erlaubt", host: "LocalHost:3000", expectedStatus: http.StatusOK},
		{name: "fremder Host", host: "evil.example:3000", expectedStatus: http.StatusForbidden},
		{name: "fremder Port", host: "127.0.0.1:3001", expectedStatus: http.StatusForbidden},
		{name: "ohne Port", host: "127.0.0.1", expectedStatus: http.StatusForbidden},
		{name: "localhost ohne Port", host: "localhost", expectedStatus: http.StatusForbidden},
		{name: "leerer Host", host: "", expectedStatus: http.StatusForbidden},
		{name: "Präfix eines erlaubten Hosts", host: "127.0.0.1:30001", expectedStatus: http.StatusForbidden},
		{name: "erlaubter Host als Suffix", host: "evil.127.0.0.1:3000", expectedStatus: http.StatusForbidden},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, "/health", testCase.host))

			if testCase.expectedStatus == http.StatusForbidden {
				assertProblem(t, recorder, http.StatusForbidden, forbiddenHostBody)
				return
			}
			if recorder.Code != testCase.expectedStatus {
				t.Errorf("Status = %d, erwartet %d", recorder.Code, testCase.expectedStatus)
			}
		})
	}
}

func TestHostGuard_AbsoluteFormErgibt403(t *testing.T) {
	// Go setzt r.Host bei absolute-form aus der URL; deshalb lehnt der Guard sie grundsätzlich ab.
	for _, target := range []string{"http://127.0.0.1:3000/health", "http://evil.example/health"} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request.Host = allowedTestHost
			if !request.URL.IsAbs() {
				t.Fatalf("Testaufbau: URL %q ist nicht absolut", target)
			}

			recorder := serve(NewHandler(testSettings, discardLogger()), request)

			assertProblem(t, recorder, http.StatusForbidden, forbiddenHostBody)
		})
	}
}

func TestHostGuard_LaeuftVorRoutingUndHandlern(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		target string
	}{
		{name: "unbekannter Pfad", method: http.MethodGet, target: "/gibt-es-nicht"},
		{name: "POST", method: http.MethodPost, target: "/health"},
		{name: "HEAD", method: http.MethodHead, target: "/health"},
		{name: "OPTIONS", method: http.MethodOptions, target: "/health"},
		{name: "DELETE", method: http.MethodDelete, target: "/positions/open"},
		{name: "openPositions", method: http.MethodGet, target: "/positions/open"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(testCase.method, testCase.target, "evil.example:3000"))

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("Status = %d, erwartet 403 vor Routing", recorder.Code)
			}
			if testCase.method != http.MethodHead {
				assertProblem(t, recorder, http.StatusForbidden, forbiddenHostBody)
			}
		})
	}
}

func TestRouter_NichtBeschriebeneMethodenUndPfadeErgeben404(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		target string
	}{
		{name: "HEAD /health", method: http.MethodHead, target: "/health"},
		{name: "OPTIONS /health", method: http.MethodOptions, target: "/health"},
		{name: "POST /health", method: http.MethodPost, target: "/health"},
		{name: "PUT /health", method: http.MethodPut, target: "/health"},
		{name: "DELETE /positions/open", method: http.MethodDelete, target: "/positions/open"},
		{name: "POST /positions/open", method: http.MethodPost, target: "/positions/open"},
		{name: "unbekannter Pfad", method: http.MethodGet, target: "/gibt-es-nicht"},
		{name: "Wurzel", method: http.MethodGet, target: "/"},
		{name: "Slash am Ende", method: http.MethodGet, target: "/health/"},
		{name: "Großschreibung im Pfad", method: http.MethodGet, target: "/Health"},
		{name: "Unterpfad", method: http.MethodGet, target: "/positions/open/x"},
		{name: "ungültige Prozent-Kodierung in der Query", method: http.MethodGet, target: "/x?y=%zz"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := newTestRequest(testCase.method, testCase.target, allowedTestHost)
			body := &trackingBody{}
			request.Body = body
			request.ContentLength = -1

			recorder := serve(NewHandler(testSettings, discardLogger()), request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("Status = %d, erwartet 404", recorder.Code)
			}
			if testCase.method != http.MethodHead {
				assertProblem(t, recorder, http.StatusNotFound, notFoundBody)
			}
			if body.wasRead.Load() {
				t.Errorf("der Request-Body wurde gelesen, darf er aber nicht")
			}
		})
	}
}

func TestRecover_PanicErgibt500OhneDetails(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, nil))
	handler := newHandlerWithRoutes(testSettings, logger, map[string]http.HandlerFunc{
		"GET /panic": func(http.ResponseWriter, *http.Request) {
			panic("geheime Ursache 4711")
		},
	})

	recorder := serve(handler, newTestRequest(http.MethodGet, "/panic", allowedTestHost))

	assertProblem(t, recorder, http.StatusInternalServerError, internalBody)
	if strings.Contains(recorder.Body.String(), "4711") {
		t.Errorf("die Panic-Ursache steht im Body")
	}
	if !strings.Contains(logOutput.String(), "geheime Ursache 4711") {
		t.Errorf("die Panic-Ursache fehlt im Log: %q", logOutput.String())
	}
}

func TestRecover_PanicMitErrorWert(t *testing.T) {
	handler := newHandlerWithRoutes(testSettings, discardLogger(), map[string]http.HandlerFunc{
		"GET /panic": func(http.ResponseWriter, *http.Request) {
			panic(io.ErrUnexpectedEOF)
		},
	})

	recorder := serve(handler, newTestRequest(http.MethodGet, "/panic", allowedTestHost))

	assertProblem(t, recorder, http.StatusInternalServerError, internalBody)
}

func TestNewHandlerWithRoutes_BehaeltStandardrouten(t *testing.T) {
	handler := newHandlerWithRoutes(testSettings, discardLogger(), map[string]http.HandlerFunc{
		"GET /extra": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) },
	})

	if recorder := serve(handler, newTestRequest(http.MethodGet, "/extra", allowedTestHost)); recorder.Code != http.StatusTeapot {
		t.Errorf("Extra-Route: Status = %d, erwartet 418", recorder.Code)
	}
	if recorder := serve(handler, newTestRequest(http.MethodGet, "/health", allowedTestHost)); recorder.Code != http.StatusOK {
		t.Errorf("/health: Status = %d, erwartet 200", recorder.Code)
	}
	if recorder := serve(handler, newTestRequest(http.MethodGet, "/extra", "evil.example:3000")); recorder.Code != http.StatusForbidden {
		t.Errorf("Extra-Route mit fremdem Host: Status = %d, erwartet 403", recorder.Code)
	}
}

func TestRecover_HostGuardLaeuftAuchVorPanicRoute(t *testing.T) {
	handler := newHandlerWithRoutes(testSettings, discardLogger(), map[string]http.HandlerFunc{
		"GET /panic": func(http.ResponseWriter, *http.Request) { panic("darf nicht laufen") },
	})

	recorder := serve(handler, newTestRequest(http.MethodGet, "/panic", "evil.example:3000"))

	assertProblem(t, recorder, http.StatusForbidden, forbiddenHostBody)
}

func requestWithMalformedFlag(host string, flagValue bool) *http.Request {
	malformedTarget := &atomic.Bool{}
	malformedTarget.Store(flagValue)
	request := newTestRequest(http.MethodGet, "/", host)
	return request.WithContext(withMalformedTargetFlag(context.Background(), malformedTarget))
}

func TestMalformedTarget(t *testing.T) {
	t.Run("gesetztes Flag ergibt 400 bad-request", func(t *testing.T) {
		recorder := serve(NewHandler(testSettings, discardLogger()), requestWithMalformedFlag(allowedTestHost, true))
		assertProblem(t, recorder, http.StatusBadRequest, badRequestBody)
	})

	t.Run("Host-Guard läuft vor malformedTarget", func(t *testing.T) {
		recorder := serve(NewHandler(testSettings, discardLogger()), requestWithMalformedFlag("evil.example:3000", true))
		assertProblem(t, recorder, http.StatusForbidden, forbiddenHostBody)
	})

	t.Run("nicht gesetztes Flag: normales Routing", func(t *testing.T) {
		recorder := serve(NewHandler(testSettings, discardLogger()), requestWithMalformedFlag(allowedTestHost, false))
		assertProblem(t, recorder, http.StatusNotFound, notFoundBody)
	})

	t.Run("ohne Flag im Context: normales Routing", func(t *testing.T) {
		recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, "/health", allowedTestHost))
		if recorder.Code != http.StatusOK {
			t.Errorf("Status = %d, erwartet 200", recorder.Code)
		}
	})

	t.Run("gesetztes Flag gewinnt auch gegen eine bekannte Route", func(t *testing.T) {
		request := requestWithMalformedFlag(allowedTestHost, true)
		request.URL.Path = "/health"
		recorder := serve(NewHandler(testSettings, discardLogger()), request)
		assertProblem(t, recorder, http.StatusBadRequest, badRequestBody)
	})
}
