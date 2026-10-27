package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

const problemContentType = "application/problem+json; charset=utf-8"

// Exakte Bodies, ohne Zeilenumbruch.
const (
	badRequestBody    = `{"type":"bad-request","title":"Die Anfrage ist ungültig.","status":400}`
	forbiddenHostBody = `{"type":"forbidden-host","title":"Der Host-Header ist nicht erlaubt.","status":403}`
	notFoundBody      = `{"type":"not-found","title":"Die angefragte Ressource gibt es nicht.","status":404}`
	notConfiguredBody = `{"type":"not-configured","title":"Der Dienst ist nicht konfiguriert.","status":503}`
	internalBody      = `{"type":"internal","title":"Interner Fehler.","status":500}`
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// assertProblem prüft Status, Content-Type und den exakten Body einer Problem-Antwort.
func assertProblem(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedBody string) {
	t.Helper()
	if recorder.Code != expectedStatus {
		t.Errorf("Status = %d, erwartet %d", recorder.Code, expectedStatus)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, erwartet %q", contentType, problemContentType)
	}
	if body := recorder.Body.String(); body != expectedBody {
		t.Errorf("Body = %q, erwartet %q", body, expectedBody)
	}
}

func TestWriteProblem(t *testing.T) {
	testCases := []struct {
		problemType    ProblemType
		expectedStatus int
		expectedBody   string
	}{
		{ProblemBadRequest, http.StatusBadRequest, badRequestBody},
		{ProblemForbiddenHost, http.StatusForbidden, forbiddenHostBody},
		{ProblemNotFound, http.StatusNotFound, notFoundBody},
		{ProblemNotConfigured, http.StatusServiceUnavailable, notConfiguredBody},
		{ProblemRateLimited, http.StatusServiceUnavailable,
			`{"type":"rate-limited","title":"BingX ist ausgelastet, bitte später erneut versuchen.","status":503}`},
		{ProblemUnreachable, http.StatusGatewayTimeout,
			`{"type":"unreachable","title":"BingX ist nicht erreichbar.","status":504}`},
		{ProblemUpstreamError, http.StatusBadGateway,
			`{"type":"upstream-error","title":"BingX hat mit einem Fehler geantwortet.","status":502}`},
		{ProblemInvalidResponse, http.StatusBadGateway,
			`{"type":"invalid-response","title":"BingX hat eine ungültige Antwort geliefert.","status":502}`},
		{ProblemInternal, http.StatusInternalServerError, internalBody},
	}

	for _, testCase := range testCases {
		t.Run(string(testCase.problemType), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeProblem(recorder, testCase.problemType)
			assertProblem(t, recorder, testCase.expectedStatus, testCase.expectedBody)
		})
	}
}

func TestWriteProblem_KonstantenEntsprechenOpenAPI(t *testing.T) {
	expectedValues := map[ProblemType]string{
		ProblemBadRequest:      "bad-request",
		ProblemForbiddenHost:   "forbidden-host",
		ProblemNotFound:        "not-found",
		ProblemNotConfigured:   "not-configured",
		ProblemRateLimited:     "rate-limited",
		ProblemUnreachable:     "unreachable",
		ProblemUpstreamError:   "upstream-error",
		ProblemInvalidResponse: "invalid-response",
		ProblemInternal:        "internal",
	}
	for problemType, expectedValue := range expectedValues {
		if string(problemType) != expectedValue {
			t.Errorf("ProblemType = %q, erwartet %q", problemType, expectedValue)
		}
	}
}

func TestWriteProblem_RetryAfterNieGesetzt(t *testing.T) {
	// Retry-After setzt erst die Route für rate-limited (Phase 2), nie writeProblem selbst.
	for _, problemType := range []ProblemType{ProblemNotConfigured, ProblemRateLimited, ProblemInternal} {
		recorder := httptest.NewRecorder()
		writeProblem(recorder, problemType)
		if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "" {
			t.Errorf("%s: Retry-After = %q, erwartet keinen Header", problemType, retryAfter)
		}
	}
}

func TestWriteProblem_UnbekannterTypWirdAlsInternalAusgeliefert(t *testing.T) {
	testCases := []struct {
		name        string
		problemType ProblemType
	}{
		{name: "unbekannter Typ", problemType: ProblemType("gibt-es-nicht")},
		{name: "leerer Typ", problemType: ProblemType("")},
		{name: "Groß-/Kleinschreibung zählt", problemType: ProblemType("NOT-FOUND")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeProblem(recorder, testCase.problemType)
			assertProblem(t, recorder, http.StatusInternalServerError, internalBody)
		})
	}
}
