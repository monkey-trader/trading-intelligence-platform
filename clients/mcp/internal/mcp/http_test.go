package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/mcp"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/tools"
)

const testToken = "test-token-nur-fuer-tests-0123456789"

func newHTTPHandler(t *testing.T, backendURL string) http.Handler {
	t.Helper()
	registry := tools.NewRegistry(apiclient.New(backendURL, time.Second))
	handler, err := mcp.NewHTTPHandler(mcp.NewServer(registry, "trading-intelligence-mcp", "test"), []byte(testToken))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

type reply struct {
	status int
	header http.Header
	body   string
}

func post(t *testing.T, handler http.Handler, body string, headers map[string]string) reply {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, mcp.HTTPPath, strings.NewReader(body))
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	data, _ := io.ReadAll(rec.Body)
	return reply{status: rec.Code, header: rec.Result().Header, body: string(data)}
}

func bearer() map[string]string { return map[string]string{"Authorization": "Bearer " + testToken} }

const pingBody = `{"jsonrpc":"2.0","id":1,"method":"ping"}`

func TestHTTPOhneTokenIst401OhneBegruendung(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	got := post(t, handler, pingBody, nil)
	if got.status != http.StatusUnauthorized || got.body != "" {
		t.Fatalf("status=%d body=%q", got.status, got.body)
	}
}

func TestHTTPMitFalschemTokenIstUnauthorizedUndGleichWieOhneToken(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	wrong := map[string]map[string]string{
		"falscher Token":    {"Authorization": "Bearer falsch"},
		"Präfix des Tokens": {"Authorization": "Bearer " + testToken[:10]},
		"Token + Zusatz":    {"Authorization": "Bearer " + testToken + "x"},
		"anderes Schema":    {"Authorization": "Basic " + testToken},
		"nur Token":         {"Authorization": testToken},
		"leer":              {"Authorization": ""},
	}
	reference := post(t, handler, pingBody, nil)
	for name, headers := range wrong {
		t.Run(name, func(t *testing.T) {
			got := post(t, handler, pingBody, headers)
			if got.status != http.StatusUnauthorized || got.body != reference.body ||
				got.header.Get("WWW-Authenticate") != reference.header.Get("WWW-Authenticate") {
				t.Fatalf("antwort weicht ab: %+v", got)
			}
		})
	}
}

func TestHTTPPruftTokenVorPfadMethodeUndOrigin(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	req := httptest.NewRequest(http.MethodGet, "/irgendwas", nil)
	req.Header.Set("Origin", "https://boese.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, 401 erwartet (kein Hinweis auf Pfad oder Origin)", rec.Code)
	}
}

func TestHTTPMitTokenBeantwortetPingAlsJSON(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	got := post(t, handler, pingBody, bearer())
	if got.status != http.StatusOK || got.header.Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d type=%q", got.status, got.header.Get("Content-Type"))
	}
	if got.body != `{"jsonrpc":"2.0","id":1,"result":{}}` {
		t.Fatalf("body = %s", got.body)
	}
}

func TestHTTPInitializeMeldetProtokoll20250618(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	got := post(t, handler, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`, bearer())
	if !strings.Contains(got.body, `"protocolVersion":"2025-06-18"`) || got.header.Get("Mcp-Session-Id") != "" {
		t.Fatalf("antwort: %s, header %v", got.body, got.header)
	}
}

func TestHTTPBenachrichtigungUndClientAntwortGebenLeere202(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	for _, body := range []string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
	} {
		got := post(t, handler, body, bearer())
		if got.status != http.StatusAccepted || got.body != "" {
			t.Fatalf("status=%d body=%q", got.status, got.body)
		}
	}
}

func TestHTTPEinWerkzeugAufrufLiefertBackendBodyUnveraendert(t *testing.T) {
	want := fixture(t, "open.json")
	var path string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(want))
	}))
	t.Cleanup(backend.Close)
	got := post(t, newHTTPHandler(t, backend.URL),
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_open_positions","arguments":{}}}`, bearer())
	if got.status != http.StatusOK || path != "/positions/open" {
		t.Fatalf("status=%d backendPfad=%q", got.status, path)
	}
	var rpc struct {
		ID     int `json:"id"`
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(got.body), &rpc); err != nil {
		t.Fatal(err)
	}
	if rpc.ID != 7 || rpc.Result.IsError || len(rpc.Result.Content) != 1 || rpc.Result.Content[0].Text != want {
		t.Fatalf("antwort: %s", got.body)
	}
}

func TestHTTPBackendNichtErreichbarIstKlareWerkzeugFehlermeldung(t *testing.T) {
	backend := httptest.NewServer(http.NotFoundHandler())
	url := backend.URL
	backend.Close() // Port ist jetzt zu
	got := post(t, newHTTPHandler(t, url),
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_account_summary","arguments":{}}}`, bearer())
	if got.status != http.StatusOK || !strings.Contains(got.body, `"isError":true`) ||
		!strings.Contains(got.body, "nicht erreichbar") {
		t.Fatalf("status=%d body=%s", got.status, got.body)
	}
}

func TestHTTPBackendTimeoutIstWerkzeugFehlerOhneErfundeneDaten(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(func() { close(release); backend.Close() })
	registry := tools.NewRegistry(apiclient.New(backend.URL, 100*time.Millisecond))
	handler, err := mcp.NewHTTPHandler(mcp.NewServer(registry, "x", "test"), []byte(testToken))
	if err != nil {
		t.Fatal(err)
	}
	got := post(t, handler,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_open_positions","arguments":{}}}`, bearer())
	if !strings.Contains(got.body, `"isError":true`) || strings.Contains(got.body, `"positionId"`) {
		t.Fatalf("body = %s", got.body)
	}
}

func TestHTTPNurPostAufMcpPfad(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodPut} {
		req := httptest.NewRequest(method, mcp.HTTPPath, nil)
		req.Header.Set("Authorization", "Bearer "+testToken)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: status = %d", method, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/anderer-pfad", strings.NewReader(pingBody))
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHTTPLehntOriginHeaderAb(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	headers := bearer()
	headers["Origin"] = "https://boese.example"
	if got := post(t, handler, pingBody, headers); got.status != http.StatusForbidden {
		t.Fatalf("status = %d", got.status)
	}
}

func TestHTTPProtokollversionHeader(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	cases := map[string]int{"2025-06-18": 200, "2025-03-26": 200, "1999-01-01": 400, "": 200}
	for version, want := range cases {
		headers := bearer()
		if version != "" {
			headers["MCP-Protocol-Version"] = version
		}
		if got := post(t, handler, pingBody, headers); got.status != want {
			t.Errorf("version %q: status = %d, erwartet %d", version, got.status, want)
		}
	}
}

func TestHTTPUngueltigeKoerperSind400(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	for name, body := range map[string]string{
		"leer":    "",
		"Batch":   `[` + pingBody + `]`,
		"zu groß": strings.Repeat("x", 2<<20),
	} {
		if got := post(t, handler, body, bearer()); got.status != http.StatusBadRequest {
			t.Errorf("%s: status = %d", name, got.status)
		}
	}
}

func TestHTTPKaputtesJSONIstRPCFehler32700(t *testing.T) {
	handler := newHTTPHandler(t, "http://127.0.0.1:1")
	got := post(t, handler, `{kaputt`, bearer())
	if !strings.Contains(got.body, "-32700") {
		t.Fatalf("body = %s", got.body)
	}
}

func TestHTTPHandlerBrauchtToken(t *testing.T) {
	registry := tools.NewRegistry(apiclient.New("http://127.0.0.1:1", time.Second))
	for _, token := range []string{"", "  \n"} {
		if _, err := mcp.NewHTTPHandler(mcp.NewServer(registry, "x", "t"), []byte(token)); err == nil {
			t.Errorf("token %q: Fehler erwartet", token)
		}
	}
}

// Nur lesend: alle vier Werkzeuge zusammen sprechen das Backend ausschließlich per GET an.
func TestHTTPAlleWerkzeugeSindNurLesend(t *testing.T) {
	var mu sync.Mutex
	var methods []string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		methods = append(methods, r.Method)
		mu.Unlock()
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(backend.Close)
	handler := newHTTPHandler(t, backend.URL)
	calls := []string{
		`{"name":"get_open_positions","arguments":{}}`,
		`{"name":"get_closed_positions","arguments":{}}`,
		`{"name":"get_account_summary","arguments":{}}`,
		`{"name":"get_position_note","arguments":{"positionId":"123"}}`,
	}
	for _, call := range calls {
		post(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":`+call+`}`, bearer())
	}
	if len(methods) != len(calls) {
		t.Fatalf("Backend-Aufrufe = %d, erwartet %d", len(methods), len(calls))
	}
	for _, method := range methods {
		if method != http.MethodGet {
			t.Fatalf("Backend-Methode %s", method)
		}
	}
}
