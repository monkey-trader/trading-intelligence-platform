package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

// Vertrag: PUT/DELETE /positions/{positionId}/initial-stop-loss in docs/openapi.yaml.
// Erwartete neue Settings-Felder (optional; beide nötig, sonst Route nicht registriert):
//   SetInitialStopLoss   func(ctx, positionID, rawStopLoss string) (appposition.SetInitialStopLossResult, error)
//   ClearInitialStopLoss func(ctx, positionID string) error
//   AllowRemote bool // true = Schreib-Routen nicht registriert (404)
// Neue ProblemType-Konstanten: ProblemInvalidStopLoss (422), ProblemUnsupportedMediaType (415),
// ProblemForbiddenOrigin (403), ProblemPayloadTooLarge (413).
// Erlaubte Origins: "http://"+jeder erlaubte Host sowie http://127.0.0.1:5173 (Vite).

const stopPath = "/positions/" + chartPositionID + "/initial-stop-loss"

const expectedStopBody = `{"positionId":"9100000000000000001","initialStopLoss":"0.9000","entryPrice":"0.9600",` +
	`"riskDistance":"0.06","source":"MANUAL","setAt":1791800000000}`

type stopCalls struct {
	setCount, clearCount int
	positionID, raw      string
}

func stopSettings(calls *stopCalls, setErr, clearErr error) Settings {
	settings := settingsWith(notConfiguredOpenPositions)
	settings.SetInitialStopLoss = func(_ context.Context, positionID, raw string) (appposition.SetInitialStopLossResult, error) {
		calls.setCount++
		calls.positionID, calls.raw = positionID, raw
		if setErr != nil {
			return appposition.SetInitialStopLossResult{}, setErr
		}
		return appposition.SetInitialStopLossResult{
			Override:     position.InitialStopLossOverride{PositionID: positionID, EntryPrice: "0.9600", InitialStopLoss: raw, SetAt: 1791800000000},
			RiskDistance: "0.06",
		}, nil
	}
	settings.ClearInitialStopLoss = func(_ context.Context, positionID string) error {
		calls.clearCount++
		calls.positionID = positionID
		return clearErr
	}
	return settings
}

type stopRequest struct {
	method, target, body, contentType, origin string
}

func sendStop(settings Settings, req stopRequest) *httptest.ResponseRecorder {
	if req.target == "" {
		req.target = stopPath
	}
	request := httptest.NewRequest(req.method, req.target, strings.NewReader(req.body))
	request.Host = allowedTestHost
	if req.contentType != "" {
		request.Header.Set("Content-Type", req.contentType)
	}
	if req.origin != "" {
		request.Header.Set("Origin", req.origin)
	}
	return serve(NewHandler(settings, discardLogger()), request)
}

func putJSON(body string) stopRequest {
	return stopRequest{method: http.MethodPut, body: body, contentType: "application/json"}
}

func problemTypeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("kein Problem-JSON: %q", recorder.Body.String())
	}
	return body.Type
}

func TestInitialStopLoss_PutErfolg(t *testing.T) {
	calls := &stopCalls{}
	recorder := sendStop(stopSettings(calls, nil, nil), putJSON(`{"initialStopLoss":"0.9000"}`))

	if recorder.Code != http.StatusOK || recorder.Body.String() != expectedStopBody {
		t.Fatalf("Status %d Body %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q", contentType)
	}
	if calls.setCount != 1 || calls.positionID != chartPositionID || calls.raw != "0.9000" {
		t.Errorf("Aufruf: %+v", calls)
	}
}

func TestInitialStopLoss_ContentTypeMitCharsetIstErlaubt(t *testing.T) {
	calls := &stopCalls{}
	req := putJSON(`{"initialStopLoss":"0.9"}`)
	req.contentType = "application/json; charset=utf-8"
	if recorder := sendStop(stopSettings(calls, nil, nil), req); recorder.Code != http.StatusOK {
		t.Fatalf("Status %d", recorder.Code)
	}
}

func TestInitialStopLoss_DeleteErgibt204OhneBody(t *testing.T) {
	calls := &stopCalls{}
	recorder := sendStop(stopSettings(calls, nil, nil), stopRequest{method: http.MethodDelete})

	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("Status %d Body %q", recorder.Code, recorder.Body.String())
	}
	if calls.clearCount != 1 || calls.positionID != chartPositionID {
		t.Errorf("Aufruf: %+v", calls)
	}
}

func TestInitialStopLoss_Eingabefehler(t *testing.T) {
	tests := []struct {
		name       string
		req        stopRequest
		status     int
		problem    string
		useCaseRun bool
	}{
		{"Body kein JSON", putJSON(`nope`), 400, "bad-request", false},
		{"leerer Body", putJSON(``), 400, "bad-request", false},
		{"Zahl statt String", putJSON(`{"initialStopLoss":0.9}`), 400, "bad-request", false},
		{"unbekanntes Feld", putJSON(`{"initialStopLoss":"0.9","extra":1}`), 400, "bad-request", false},
		{"Feld fehlt", putJSON(`{}`), 400, "bad-request", false},
		{"zweites JSON-Dokument", putJSON(`{"initialStopLoss":"0.9"}{}`), 400, "bad-request", false},
		{"positionId keine Ziffern", stopRequest{method: http.MethodPut, target: "/positions/abc/initial-stop-loss", body: `{"initialStopLoss":"0.9"}`, contentType: "application/json"}, 400, "bad-request", false},
		{"falscher Content-Type", stopRequest{method: http.MethodPut, body: `{"initialStopLoss":"0.9"}`, contentType: "text/plain"}, 415, "unsupported-media-type", false},
		{"Content-Type fehlt", stopRequest{method: http.MethodPut, body: `{"initialStopLoss":"0.9"}`}, 415, "unsupported-media-type", false},
		{"fremder Origin PUT", func() stopRequest {
			r := putJSON(`{"initialStopLoss":"0.9"}`)
			r.origin = "http://evil.example"
			return r
		}(), 403, "forbidden-origin", false},
		{"Origin null", func() stopRequest { r := putJSON(`{"initialStopLoss":"0.9"}`); r.origin = "null"; return r }(), 403, "forbidden-origin", false},
		{"fremder Origin DELETE", stopRequest{method: http.MethodDelete, origin: "http://evil.example"}, 403, "forbidden-origin", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := &stopCalls{}
			recorder := sendStop(stopSettings(calls, nil, nil), test.req)
			if recorder.Code != test.status || problemTypeOf(t, recorder) != test.problem {
				t.Fatalf("Status %d Body %s", recorder.Code, recorder.Body.String())
			}
			if calls.setCount != 0 || calls.clearCount != 0 {
				t.Error("Use Case darf nicht laufen")
			}
		})
	}
}

func TestInitialStopLoss_BodyGrenze1KiB(t *testing.T) {
	padding := func(total int) string {
		prefix, suffix := `{"initialStopLoss":"0.9","x":"`, `"}`
		return prefix + strings.Repeat("a", total-len(prefix)-len(suffix)) + suffix
	}
	// 1024 Byte: nicht wegen der Größe abgelehnt (400 wegen unbekanntem Feld), 1025 Byte: 413.
	if recorder := sendStop(stopSettings(&stopCalls{}, nil, nil), putJSON(padding(1024))); recorder.Code != http.StatusBadRequest {
		t.Errorf("1024 Byte: Status %d", recorder.Code)
	}
	recorder := sendStop(stopSettings(&stopCalls{}, nil, nil), putJSON(padding(1025)))
	if recorder.Code != http.StatusRequestEntityTooLarge || problemTypeOf(t, recorder) != "payload-too-large" {
		t.Errorf("1025 Byte: Status %d Body %s", recorder.Code, recorder.Body.String())
	}
}

func TestInitialStopLoss_ErlaubteOrigins(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:3000", "http://localhost:3000", "http://[::1]:3000", "http://127.0.0.1:5173"} {
		t.Run(origin, func(t *testing.T) {
			req := putJSON(`{"initialStopLoss":"0.9"}`)
			req.origin = origin
			if recorder := sendStop(stopSettings(&stopCalls{}, nil, nil), req); recorder.Code != http.StatusOK {
				t.Fatalf("PUT Status %d", recorder.Code)
			}
			if recorder := sendStop(stopSettings(&stopCalls{}, nil, nil), stopRequest{method: http.MethodDelete, origin: origin}); recorder.Code != http.StatusNoContent {
				t.Fatalf("DELETE Status %d", recorder.Code)
			}
		})
	}
}

func TestInitialStopLoss_FehlerDesUseCase(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		problem string
	}{
		{"Position nicht offen", &chart.PositionNotFoundError{PositionID: chartPositionID}, 404, "not-found"},
		{"gleich Entry", &position.StopLossAtEntryError{}, 422, "invalid-stop-loss"},
		{"falsche Seite", &position.StopLossWrongSideError{}, 422, "invalid-stop-loss"},
		{"verpackt", errors.Join(errors.New("x"), &position.StopLossWrongSideError{}), 422, "invalid-stop-loss"},
		{"keine Zahl", &decimal.InvalidDecimalError{}, 422, "invalid-stop-loss"},
		{"nicht konfiguriert", &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}, 503, "not-configured"},
		{"Rate-Limit", &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}, 503, "rate-limited"},
		{"nicht erreichbar", &position.OpenPositionsUnavailableError{Reason: position.ReasonUnreachable}, 504, "unreachable"},
		{"Upstream", &position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError}, 502, "upstream-error"},
		{"ungültige Antwort", &position.OpenPositionsUnavailableError{Reason: position.ReasonInvalidResponse}, 502, "invalid-response"},
		{"unbekannt", errors.New("db kaputt"), 500, "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := sendStop(stopSettings(&stopCalls{}, test.err, nil), putJSON(`{"initialStopLoss":"0.9"}`))
			if recorder.Code != test.status || problemTypeOf(t, recorder) != test.problem {
				t.Fatalf("Status %d Body %s", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "db kaputt") {
				t.Error("interne Details im Body")
			}
		})
	}
}

func TestInitialStopLoss_DeleteFehlerIst500OhneDetails(t *testing.T) {
	recorder := sendStop(stopSettings(&stopCalls{}, nil, errors.New("db kaputt")), stopRequest{method: http.MethodDelete})
	if recorder.Code != 500 || problemTypeOf(t, recorder) != "internal" {
		t.Fatalf("Status %d Body %s", recorder.Code, recorder.Body.String())
	}
}

func TestInitialStopLoss_NichtRegistriertBeiRemoteOderOhneUseCase(t *testing.T) {
	remote := stopSettings(&stopCalls{}, nil, nil)
	remote.AllowRemote = true
	withoutSet := stopSettings(&stopCalls{}, nil, nil)
	withoutSet.SetInitialStopLoss = nil
	for name, settings := range map[string]Settings{"ALLOW_REMOTE": remote, "ohne Use Case": withoutSet} {
		t.Run(name, func(t *testing.T) {
			for _, req := range []stopRequest{putJSON(`{"initialStopLoss":"0.9"}`), {method: http.MethodDelete}} {
				if recorder := sendStop(settings, req); recorder.Code != http.StatusNotFound {
					t.Errorf("%s: Status %d", req.method, recorder.Code)
				}
			}
		})
	}
}

func TestInitialStopLoss_AndereMethodenSind404(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch} {
		if recorder := sendStop(stopSettings(&stopCalls{}, nil, nil), stopRequest{method: method, contentType: "application/json", body: `{}`}); recorder.Code != http.StatusNotFound {
			t.Errorf("%s: Status %d", method, recorder.Code)
		}
	}
}
