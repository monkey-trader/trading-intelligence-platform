package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/tools"
)

type backend struct {
	server *httptest.Server
	mu     sync.Mutex
	paths  []string // Pfad plus Query je Request
}

func (b *backend) requests() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.paths...)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../../fixtures/api/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// newBackend antwortet mit status und Fixture und merkt sich jeden Request.
func newBackend(t *testing.T, status int, name string) *backend {
	t.Helper()
	b := &backend{}
	body := fixture(t, name)
	b.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		b.paths = append(b.paths, r.URL.RequestURI())
		b.mu.Unlock()
		if status >= 400 {
			w.Header().Set("Content-Type", "application/problem+json")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(b.server.Close)
	return b
}

func registry(b *backend) *tools.Registry {
	return tools.NewRegistry(apiclient.New(b.server.URL, time.Second))
}

func lookup(t *testing.T, r *tools.Registry, name string) tools.Tool {
	t.Helper()
	tool, ok := r.Lookup(name)
	if !ok {
		t.Fatalf("Tool %s fehlt", name)
	}
	return tool
}

func TestRegistryListetDieVierToolsMitDeutscherBeschreibungUndSchema(t *testing.T) {
	r := registry(newBackend(t, 200, "open.json"))
	want := []string{"get_open_positions", "get_closed_positions", "get_account_summary", "get_position_note"}
	list := r.List()
	if len(list) != len(want) {
		t.Fatalf("erwartet %d Tools, bekam %d", len(want), len(list))
	}
	for i, name := range want {
		tool := list[i]
		if tool.Name != name {
			t.Errorf("Position %d: %s statt %s", i, tool.Name, name)
		}
		if strings.TrimSpace(tool.Description) == "" || tool.Call == nil {
			t.Errorf("%s: Beschreibung oder Call fehlt", name)
		}
		var schema map[string]any
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil || schema["type"] != "object" {
			t.Errorf("%s: InputSchema ist kein Objekt-Schema: %s", name, tool.InputSchema)
		}
	}
}

func TestRegistryLookupUnbekanntesToolLiefertFalse(t *testing.T) {
	r := registry(newBackend(t, 200, "open.json"))
	if _, ok := r.Lookup("delete_everything"); ok {
		t.Error("unbekanntes Tool darf nicht gefunden werden")
	}
}

func TestToolsReichenDenBackendBodyUnveraendertDurch(t *testing.T) {
	cases := []struct {
		tool, fixture, path string
		args                string
	}{
		{"get_open_positions", "open.json", "/positions/open", `{}`},
		{"get_closed_positions", "closed.json", "/positions/closed", `{}`},
		{"get_account_summary", "summary.json", "/account/summary", `{}`},
		{"get_position_note", "note.json", "/positions/1234567890123456789/note", `{"positionId":"1234567890123456789"}`},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			b := newBackend(t, 200, tc.fixture)
			got, err := lookup(t, registry(b), tc.tool).Call(context.Background(), json.RawMessage(tc.args))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(fixture(t, tc.fixture)) {
				t.Errorf("Body verändert:\n%s", got)
			}
			reqs := b.requests()
			if len(reqs) != 1 || reqs[0] != tc.path {
				t.Errorf("Requests: %v, erwartet [%s]", reqs, tc.path)
			}
		})
	}
}

func TestToolsAkzeptierenFehlendeArgumenteAlsLeeresObjekt(t *testing.T) {
	for _, args := range []string{``, `null`, `{}`} {
		b := newBackend(t, 200, "open.json")
		if _, err := lookup(t, registry(b), "get_open_positions").Call(context.Background(), json.RawMessage(args)); err != nil {
			t.Errorf("args %q: %v", args, err)
		}
	}
}

func TestGetClosedPositionsGibtFilterAlsQueryWeiter(t *testing.T) {
	b := newBackend(t, 200, "closed.json")
	args := `{"from":1759000000000,"to":1759100000000,"symbol":"SUI-USDT","limit":50}`
	if _, err := lookup(t, registry(b), "get_closed_positions").Call(context.Background(), json.RawMessage(args)); err != nil {
		t.Fatal(err)
	}
	want := "/positions/closed?from=1759000000000&limit=50&symbol=SUI-USDT&to=1759100000000"
	if reqs := b.requests(); len(reqs) != 1 || reqs[0] != want {
		t.Errorf("Requests: %v, erwartet %s", reqs, want)
	}
}

func TestGetClosedPositionsGrenzwerte(t *testing.T) {
	cases := []struct {
		name  string
		args  string
		valid bool
	}{
		{"limit genau 1", `{"limit":1}`, true},
		{"limit genau 500", `{"limit":500}`, true},
		{"limit 0", `{"limit":0}`, false},
		{"limit 501", `{"limit":501}`, false},
		{"limit negativ", `{"limit":-1}`, false},
		{"limit Dezimalzahl", `{"limit":1.5}`, false},
		{"limit als String", `{"limit":"10"}`, false},
		{"from 0", `{"from":0}`, true},
		{"from negativ", `{"from":-1}`, false},
		{"from Dezimalzahl", `{"from":1.5}`, false},
		{"from als String", `{"from":"1"}`, false},
		{"to gleich from", `{"from":100,"to":100}`, true},
		{"to genau from plus 1", `{"from":100,"to":101}`, true},
		{"to kleiner als from", `{"from":100,"to":99}`, false},
		{"to negativ", `{"to":-5}`, false},
		{"symbol als Zahl", `{"symbol":5}`, false},
		{"kein Objekt", `[1]`, false},
		{"kaputtes JSON", `{"limit":`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBackend(t, 200, "closed.json")
			_, err := lookup(t, registry(b), "get_closed_positions").Call(context.Background(), json.RawMessage(tc.args))
			if tc.valid {
				if err != nil {
					t.Fatalf("gültig erwartet: %v", err)
				}
				if len(b.requests()) != 1 {
					t.Error("ein Request erwartet")
				}
				return
			}
			if !errors.Is(err, tools.ErrInvalidArguments) {
				t.Fatalf("erwartet ErrInvalidArguments, bekam %v", err)
			}
			if n := len(b.requests()); n != 0 {
				t.Errorf("bei ungültigen Argumenten darf kein Request gehen, waren %d", n)
			}
		})
	}
}

func TestGetPositionNoteValidiertPositionIdVorDemAufruf(t *testing.T) {
	cases := []struct {
		name  string
		args  string
		valid bool
	}{
		{"19 Ziffern", `{"positionId":"1234567890123456789"}`, true},
		{"1 Ziffer", `{"positionId":"1"}`, true},
		{"genau 20 Ziffern", `{"positionId":"12345678901234567890"}`, true},
		{"21 Ziffern", `{"positionId":"123456789012345678901"}`, false},
		{"leer", `{"positionId":""}`, false},
		{"fehlt", `{}`, false},
		{"als Zahl", `{"positionId":1234567890123456789}`, false},
		{"Buchstaben", `{"positionId":"12a"}`, false},
		{"Pfad-Injektion", `{"positionId":"1/../../account/summary"}`, false},
		{"Zeilenumbruch am Ende", `{"positionId":"123\n"}`, false},
		{"Leerzeichen", `{"positionId":"12 3"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBackend(t, 200, "note.json")
			_, err := lookup(t, registry(b), "get_position_note").Call(context.Background(), json.RawMessage(tc.args))
			if tc.valid && err != nil {
				t.Fatalf("gültig erwartet: %v", err)
			}
			if !tc.valid {
				if !errors.Is(err, tools.ErrInvalidArguments) {
					t.Fatalf("erwartet ErrInvalidArguments, bekam %v", err)
				}
				if n := len(b.requests()); n != 0 {
					t.Errorf("kein Request erwartet, waren %d", n)
				}
			}
		})
	}
}

func TestToolsGebenBackendFehlerAlsProblemErrorWeiter(t *testing.T) {
	cases := []struct {
		fixture string
		status  int
	}{{"problem-404.json", 404}, {"problem-503.json", 503}}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			b := newBackend(t, tc.status, tc.fixture)
			_, err := lookup(t, registry(b), "get_position_note").Call(context.Background(), json.RawMessage(`{"positionId":"1234567890123456789"}`))
			var problem *apiclient.ProblemError
			if !errors.As(err, &problem) || problem.Status != tc.status {
				t.Fatalf("erwartet *ProblemError %d, bekam %v", tc.status, err)
			}
		})
	}
}

func TestToolsMeldenErrUnreachableWennBackendFehlt(t *testing.T) {
	b := newBackend(t, 200, "open.json")
	r := registry(b)
	b.server.Close()
	_, err := lookup(t, r, "get_open_positions").Call(context.Background(), json.RawMessage(`{}`))
	if !errors.Is(err, apiclient.ErrUnreachable) {
		t.Fatalf("erwartet ErrUnreachable, bekam %v", err)
	}
}
