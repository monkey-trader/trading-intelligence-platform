package apiclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../../fixtures/api/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestGetReichtDenBodyUnveraendertDurch(t *testing.T) {
	body := fixture(t, "closed.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	got, err := apiclient.New(server.URL, time.Second).Get(context.Background(), "/positions/closed", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Errorf("Body verändert:\n%s", got)
	}
	if !strings.Contains(string(got), `"positionAmount":"150.5"`) {
		t.Error("Dezimal-String nicht erhalten")
	}
}

func TestGetSendetPfadUndQueryParameter(t *testing.T) {
	var gotPath, gotQuery, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotMethod = r.URL.Path, r.URL.RawQuery, r.Method
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	query := url.Values{"symbol": {"SUI-USDT"}, "limit": {"5"}}
	if _, err := apiclient.New(server.URL, time.Second).Get(context.Background(), "/positions/closed", query); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/positions/closed" {
		t.Errorf("Methode/Pfad: %s %s", gotMethod, gotPath)
	}
	if gotQuery != "limit=5&symbol=SUI-USDT" {
		t.Errorf("Query: %s", gotQuery)
	}
}

func TestGetLiefertProblemErrorAusProblemJSON(t *testing.T) {
	cases := []struct {
		fixture string
		status  int
		typ     string
	}{
		{"problem-404.json", 404, "not-found"},
		{"problem-503.json", 503, "not-configured"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(tc.status)
				_, _ = w.Write(fixture(t, tc.fixture))
			}))
			defer server.Close()

			_, err := apiclient.New(server.URL, time.Second).Get(context.Background(), "/x", nil)
			var problem *apiclient.ProblemError
			if !errors.As(err, &problem) {
				t.Fatalf("kein *ProblemError: %v", err)
			}
			if problem.Status != tc.status || problem.Type != tc.typ || problem.Title == "" || problem.Detail == "" {
				t.Errorf("Problem unvollständig: %+v", problem)
			}
		})
	}
}

func TestGetOhneProblemJSONLiefertTrotzdemFehlerMitStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "kaputt", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := apiclient.New(server.URL, time.Second).Get(context.Background(), "/x", nil)
	var problem *apiclient.ProblemError
	if !errors.As(err, &problem) || problem.Status != 500 {
		t.Fatalf("erwartet *ProblemError mit Status 500, bekam %v", err)
	}
}

func TestGetMeldetErrUnreachableBeiVerbindungsfehler(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address := server.URL
	server.Close()

	_, err := apiclient.New(address, time.Second).Get(context.Background(), "/x", nil)
	if !errors.Is(err, apiclient.ErrUnreachable) {
		t.Fatalf("erwartet ErrUnreachable, bekam %v", err)
	}
}

func TestGetBrichtNachTimeoutMitErrUnreachableAb(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)

	start := time.Now()
	_, err := apiclient.New(server.URL, 50*time.Millisecond).Get(context.Background(), "/x", nil)
	if !errors.Is(err, apiclient.ErrUnreachable) {
		t.Fatalf("erwartet ErrUnreachable, bekam %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("Timeout greift nicht")
	}
}

func TestGetBrichtBeiAbgebrochenemKontextAb(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := apiclient.New(server.URL, time.Second).Get(ctx, "/x", nil); err == nil {
		t.Fatal("Fehler erwartet")
	}
}
