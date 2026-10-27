package mcp_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/mcp"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/tools"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../../../fixtures/api/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// session startet den Server auf io.Pipe, send schreibt eine Zeile, recv liest eine Antwortzeile.
type session struct {
	t      *testing.T
	in     *io.PipeWriter
	lines  chan string
	done   chan error
	cancel context.CancelFunc
}

func newSession(t *testing.T, backend http.Handler) *session {
	t.Helper()
	server := httptest.NewServer(backend)
	t.Cleanup(server.Close)
	return newSessionFor(t, server.URL)
}

func newSessionFor(t *testing.T, baseURL string) *session {
	t.Helper()
	registry := tools.NewRegistry(apiclient.New(baseURL, time.Second))
	srv := mcp.NewServer(registry, "trading-intelligence-mcp", "test")
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{t: t, in: inW, lines: make(chan string, 16), done: make(chan error, 1), cancel: cancel}
	go func() {
		s.done <- srv.Serve(ctx, inR, outW)
		_ = outW.Close()
	}()
	go func() {
		scanner := bufio.NewScanner(outR)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		for scanner.Scan() {
			s.lines <- scanner.Text()
		}
		close(s.lines)
	}()
	t.Cleanup(func() { cancel(); _ = inW.Close() })
	return s
}

func (s *session) send(line string) {
	s.t.Helper()
	go func() { _, _ = io.WriteString(s.in, line+"\n") }()
}

func (s *session) recv() map[string]any {
	s.t.Helper()
	select {
	case line, ok := <-s.lines:
		if !ok {
			s.t.Fatal("Ausgabe geschlossen")
		}
		if strings.Contains(line, "\n") {
			s.t.Fatal("mehr als eine Zeile")
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			s.t.Fatalf("Antwort ist kein JSON: %q", line)
		}
		if msg["jsonrpc"] != "2.0" {
			s.t.Errorf("jsonrpc fehlt: %q", line)
		}
		return msg
	case <-time.After(2 * time.Second):
		s.t.Fatal("keine Antwort")
		return nil
	}
}

func (s *session) expectSilence() {
	s.t.Helper()
	select {
	case line := <-s.lines:
		s.t.Fatalf("keine Antwort erwartet, bekam %q", line)
	case <-time.After(150 * time.Millisecond):
	}
}

func okBackend(t *testing.T) http.Handler {
	routes := map[string]string{
		"/positions/open":                     "open.json",
		"/positions/closed":                   "closed.json",
		"/account/summary":                    "summary.json",
		"/positions/1234567890123456789/note": "note.json",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := routes[r.URL.Path]
		if !ok {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(404)
			_, _ = io.WriteString(w, fixture(t, "problem-404.json"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fixture(t, name))
	})
}

func errorCode(t *testing.T, msg map[string]any) float64 {
	t.Helper()
	e, ok := msg["error"].(map[string]any)
	if !ok {
		t.Fatalf("error-Objekt erwartet: %v", msg)
	}
	return e["code"].(float64)
}

func TestInitializeAntwortetMitProtokollversionCapabilitiesUndServerInfo(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`)
	msg := s.recv()
	if msg["id"].(float64) != 1 {
		t.Errorf("id: %v", msg["id"])
	}
	result := msg["result"].(map[string]any)
	if result["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion: %v", result["protocolVersion"])
	}
	caps := result["capabilities"].(map[string]any)
	if _, ok := caps["tools"].(map[string]any); !ok {
		t.Errorf("capabilities.tools fehlt: %v", caps)
	}
	info := result["serverInfo"].(map[string]any)
	if info["name"] == "" || info["version"] == "" {
		t.Errorf("serverInfo: %v", info)
	}
}

func TestInitializeMitAndererVersionAntwortetMitEigenerVersion(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":"a","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`)
	msg := s.recv()
	if msg["id"] != "a" {
		t.Errorf("String-id muss erhalten bleiben: %v", msg["id"])
	}
	if v := msg["result"].(map[string]any)["protocolVersion"]; v != "2025-06-18" {
		t.Errorf("protocolVersion: %v", v)
	}
}

func TestPingAntwortetMitLeeremResult(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":7,"method":"ping"}`)
	msg := s.recv()
	if r, ok := msg["result"].(map[string]any); !ok || len(r) != 0 {
		t.Errorf("leeres result erwartet: %v", msg)
	}
}

func TestBenachrichtigungenBleibenOhneAntwort(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	s.send(`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`)
	s.expectSilence()
	s.send(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if s.recv()["id"].(float64) != 2 {
		t.Error("Server muss danach weiter antworten")
	}
}

func TestToolsListLiefertDieVierToolsMitSchema(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	list := s.recv()["result"].(map[string]any)["tools"].([]any)
	if len(list) != 4 {
		t.Fatalf("4 Tools erwartet, bekam %d", len(list))
	}
	names := map[string]bool{}
	for _, item := range list {
		tool := item.(map[string]any)
		names[tool["name"].(string)] = true
		if tool["description"] == "" {
			t.Errorf("Beschreibung fehlt: %v", tool["name"])
		}
		if schema, ok := tool["inputSchema"].(map[string]any); !ok || schema["type"] != "object" {
			t.Errorf("inputSchema: %v", tool["inputSchema"])
		}
	}
	for _, n := range []string{"get_open_positions", "get_closed_positions", "get_account_summary", "get_position_note"} {
		if !names[n] {
			t.Errorf("Tool %s fehlt", n)
		}
	}
}

func callTool(name, args string) string {
	return `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
}

func resultText(t *testing.T, msg map[string]any) (string, bool) {
	t.Helper()
	result, ok := msg["result"].(map[string]any)
	if !ok {
		t.Fatalf("result erwartet: %v", msg)
	}
	content := result["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("ein Inhaltsblock erwartet: %v", content)
	}
	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Errorf("type: %v", block["type"])
	}
	isError, _ := result["isError"].(bool)
	return block["text"].(string), isError
}

func TestToolsCallGibtBackendBodyAlsTextUnveraendertZurueck(t *testing.T) {
	cases := []struct{ tool, args, fixture string }{
		{"get_open_positions", `{}`, "open.json"},
		{"get_closed_positions", `{"limit":10}`, "closed.json"},
		{"get_account_summary", `{}`, "summary.json"},
		{"get_position_note", `{"positionId":"1234567890123456789"}`, "note.json"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			s := newSession(t, okBackend(t))
			s.send(callTool(tc.tool, tc.args))
			text, isError := resultText(t, s.recv())
			if isError {
				t.Error("isError darf nicht gesetzt sein")
			}
			if text != strings.TrimRight(fixture(t, tc.fixture), "\n") && text != fixture(t, tc.fixture) {
				t.Errorf("Body verändert:\n%s", text)
			}
		})
	}
}

func TestToolsCallFehlerWerdenAlsIsErrorMitDeutschemKlartextGemeldet(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()

	cases := []struct {
		name    string
		session func(t *testing.T) *session
		request string
	}{
		{"Backend nicht erreichbar", func(t *testing.T) *session { return newSessionFor(t, deadURL) }, callTool("get_open_positions", `{}`)},
		{"Problem 404", func(t *testing.T) *session { return newSession(t, okBackend(t)) }, callTool("get_position_note", `{"positionId":"999"}`)},
		{"Problem 503", func(t *testing.T) *session {
			return newSession(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.Header().Set("X-Secret", "geheim")
				w.WriteHeader(503)
				_, _ = io.WriteString(w, fixture(t, "problem-503.json"))
			}))
		}, callTool("get_account_summary", `{}`)},
		{"Validierung limit", func(t *testing.T) *session { return newSession(t, okBackend(t)) }, callTool("get_closed_positions", `{"limit":501}`)},
		{"Validierung positionId", func(t *testing.T) *session { return newSession(t, okBackend(t)) }, callTool("get_position_note", `{"positionId":"abc"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.session(t)
			s.send(tc.request)
			msg := s.recv()
			if _, hasErr := msg["error"]; hasErr {
				t.Fatalf("Tool-Fehler gehören in result.isError, nicht in error: %v", msg)
			}
			text, isError := resultText(t, msg)
			if !isError {
				t.Error("isError=true erwartet")
			}
			if strings.TrimSpace(text) == "" {
				t.Error("Klartext erwartet")
			}
			for _, leak := range []string{"geheim", "X-Secret", "X-BX-APIKEY", "127.0.0.1", "goroutine", "panic"} {
				if strings.Contains(text, leak) {
					t.Errorf("Antwort enthält %q: %s", leak, text)
				}
			}
			s.send(`{"jsonrpc":"2.0","id":10,"method":"ping"}`)
			if s.recv()["id"].(float64) != 10 {
				t.Error("Server muss nach Tool-Fehler weiterlaufen")
			}
		})
	}
}

func TestToolsCallMitUnbekanntemToolOderFehlendemNamenIstUngueltigerParameter(t *testing.T) {
	cases := map[string]string{
		"unbekanntes Tool":        `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gibts_nicht","arguments":{}}}`,
		"Name fehlt":              `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{}}}`,
		"params fehlen":           `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`,
		"Name keine Zeichenkette": `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":5}}`,
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			s := newSession(t, okBackend(t))
			s.send(line)
			if code := errorCode(t, s.recv()); code != -32602 {
				t.Errorf("Code %v statt -32602", code)
			}
		})
	}
}

func TestUnbekannteMethodeLiefertFehler32601(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`)
	msg := s.recv()
	if code := errorCode(t, msg); code != -32601 {
		t.Errorf("Code %v", code)
	}
	if msg["id"].(float64) != 4 {
		t.Errorf("id: %v", msg["id"])
	}
}

func TestKaputtesJSONLiefertFehler32700UndServerLaeuftWeiter(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":`)
	msg := s.recv()
	if code := errorCode(t, msg); code != -32700 {
		t.Errorf("Code %v", code)
	}
	if msg["id"] != nil {
		t.Errorf("id muss null sein: %v", msg["id"])
	}
	s.send(`{"jsonrpc":"2.0","id":5,"method":"ping"}`)
	if s.recv()["id"].(float64) != 5 {
		t.Error("Server muss weiterlaufen")
	}
}

func TestInitializeOhneParamsIstUngueltigerParameter(t *testing.T) {
	s := newSession(t, okBackend(t))
	s.send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":"kaputt"}`)
	if code := errorCode(t, s.recv()); code != -32602 {
		t.Errorf("Code %v statt -32602", code)
	}
}

func TestServeEndetOhneFehlerBeiEndeDerEingabe(t *testing.T) {
	s := newSession(t, okBackend(t))
	_ = s.in.Close()
	select {
	case err := <-s.done:
		if err != nil {
			t.Errorf("kein Fehler erwartet: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve endet nicht")
	}
}
