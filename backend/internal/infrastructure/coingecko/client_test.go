package coingecko_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/coingecko"
)

// Pfad relativ zum Paketordner: backend/internal/infrastructure/coingecko → Repo-Root.
const fixtureDir = "../../../../fixtures/coingecko/"

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(fixtureDir + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return content
}

func bases(t *testing.T, symbols ...string) []asset.BaseAsset {
	t.Helper()
	result := make([]asset.BaseAsset, 0, len(symbols))
	for _, symbol := range symbols {
		base, err := asset.NewBaseAsset(symbol)
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, base)
	}
	return result
}

func baseOf(t *testing.T, symbol string) asset.BaseAsset { return bases(t, symbol)[0] }

// recordedRequest hält den letzten Request des Testservers.
type recordedRequest struct {
	mutex   sync.Mutex
	request *http.Request
	count   int
}

func (recorded *recordedRequest) record(request *http.Request) {
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()
	recorded.request = request
	recorded.count++
}

func serveBody(t *testing.T, status int, body []byte) (*httptest.Server, *recordedRequest) {
	t.Helper()
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server, recorded
}

func TestResolveIcons_HappyPath(t *testing.T) {
	server, _ := serveBody(t, http.StatusOK, loadFixture(t, "markets.sample.json"))
	client := coingecko.NewClient(server.URL)

	icons, err := client.ResolveIcons(context.Background(), bases(t, "BTC", "SUI", "ENA", "ONDO", "ALGO", "XRP"))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(icons) != 6 {
		t.Fatalf("Icons = %d, erwartet 6", len(icons))
	}
	suiIcon, found := icons[baseOf(t, "SUI")]
	if !found || suiIcon.String() != "https://coin-images.coingecko.com/coins/images/26375/large/sui-ocean-square.png?1727791290" {
		t.Errorf("SUI = %q (gefunden=%v)", suiIcon.String(), found)
	}
}

func TestResolveIcons_Request(t *testing.T) {
	server, recorded := serveBody(t, http.StatusOK, []byte("[]"))
	client := coingecko.NewClient(server.URL)

	if _, err := client.ResolveIcons(context.Background(), bases(t, "BTC", "1000PEPE", "SUI")); err != nil {
		t.Fatal(err)
	}

	request := recorded.request
	if request.Method != http.MethodGet || request.URL.Path != "/api/v3/coins/markets" {
		t.Errorf("Request = %s %s", request.Method, request.URL.Path)
	}
	query := request.URL.Query()
	if query.Get("symbols") != "btc,1000pepe,sui" {
		t.Errorf("symbols = %q, erwartet nur die angefragten, kleingeschriebenen Symbole", query.Get("symbols"))
	}
	for key, want := range map[string]string{"vs_currency": "usd", "order": "market_cap_desc", "per_page": "250"} {
		if query.Get(key) != want {
			t.Errorf("%s = %q, erwartet %q", key, query.Get(key), want)
		}
	}
	if request.Header.Get("User-Agent") == "" || strings.HasPrefix(request.Header.Get("User-Agent"), "Go-http-client") {
		t.Errorf("User-Agent = %q, erwartet eigener Wert", request.Header.Get("User-Agent"))
	}
}

func TestResolveIcons_LeereListeFragtNichtAn(t *testing.T) {
	server, recorded := serveBody(t, http.StatusOK, []byte("[]"))
	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), nil)
	if err != nil || len(icons) != 0 || recorded.count != 0 {
		t.Errorf("icons=%v err=%v Requests=%d, erwartet leer, kein Fehler, kein Request", icons, err, recorded.count)
	}
}

func TestResolveIcons_BesterRangGewinnt(t *testing.T) {
	server, _ := serveBody(t, http.StatusOK, loadFixture(t, "markets.collision.json"))

	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "SUI", "OND"))
	if err != nil {
		t.Fatal(err)
	}

	if got := icons[baseOf(t, "SUI")].String(); !strings.Contains(got, "/26375/") {
		t.Errorf("SUI = %q, erwartet den Treffer mit Rang 28", got)
	}
	// Ohne Rang ist besser als nichts, wenn es keinen mit Rang gibt.
	if got := icons[baseOf(t, "OND")].String(); !strings.Contains(got, "/99997/") {
		t.Errorf("OND = %q, erwartet den einzigen (rang-losen) Treffer", got)
	}
}

func TestResolveIcons_SymbolCaseInsensitiv(t *testing.T) {
	body := `[{"id":"x","symbol":"SuI","name":"x","image":"https://coin-images.coingecko.com/a.png","market_cap_rank":1}]`
	server, _ := serveBody(t, http.StatusOK, []byte(body))

	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "SUI"))
	if err != nil || len(icons) != 1 {
		t.Errorf("icons=%v err=%v, erwartet 1 Treffer trotz Mixed-Case", icons, err)
	}
}

func TestResolveIcons_AllowlistUndFehlendeFelder(t *testing.T) {
	body := `[
	 {"id":"a","symbol":"evil","name":"a","image":"https://evil.example.com/a.png","market_cap_rank":1},
	 {"id":"b","symbol":"plain","name":"b","image":"http://coin-images.coingecko.com/a.png","market_cap_rank":1},
	 {"id":"c","symbol":"noimg","name":"c","market_cap_rank":1},
	 {"id":"d","symbol":"nullimg","name":"d","image":null,"market_cap_rank":1},
	 {"id":"e","symbol":"good","name":"e","image":"https://coin-images.coingecko.com/e.png","market_cap_rank":3}]`
	server, _ := serveBody(t, http.StatusOK, []byte(body))

	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "EVIL", "PLAIN", "NOIMG", "NULLIMG", "GOOD"))
	if err != nil {
		t.Fatal(err)
	}
	if len(icons) != 1 {
		t.Errorf("Icons = %v, erwartet nur GOOD", icons)
	}
	if _, found := icons[baseOf(t, "GOOD")]; !found {
		t.Error("GOOD fehlt")
	}
}

func TestResolveIcons_FixtureMitFremdemHost(t *testing.T) {
	server, _ := serveBody(t, http.StatusOK, loadFixture(t, "markets.collision.json"))
	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "EVIL", "PLAIN"))
	if err != nil || len(icons) != 0 {
		t.Errorf("icons=%v err=%v, erwartet leere Map ohne Fehler", icons, err)
	}
}

func TestResolveIcons_BesserRangMitFremdemHostWirdUebersprungenNichtGewaehlt(t *testing.T) {
	body := `[
	 {"id":"a","symbol":"sui","name":"a","image":"https://evil.example.com/a.png","market_cap_rank":1},
	 {"id":"b","symbol":"sui","name":"b","image":"https://coin-images.coingecko.com/b.png","market_cap_rank":5}]`
	server, _ := serveBody(t, http.StatusOK, []byte(body))
	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "SUI"))
	if err != nil {
		t.Fatal(err)
	}
	if got := icons[baseOf(t, "SUI")]; got.String() != "https://coin-images.coingecko.com/b.png" {
		t.Errorf("SUI = %q, erwartet den Treffer mit erlaubtem Host", got.String())
	}
}

func TestResolveIcons_Fehler(t *testing.T) {
	oversized := []byte(`[{"id":"x","symbol":"btc","name":"` + strings.Repeat("a", 1<<20+1) + `"}]`)
	tests := []struct {
		name   string
		status int
		body   []byte
	}{
		{"HTTP 500", http.StatusInternalServerError, []byte(`{"error":"boom"}`)},
		{"HTTP 503", http.StatusServiceUnavailable, nil},
		{"ungültiges JSON", http.StatusOK, []byte("<html>nope</html>")},
		{"JSON-Objekt statt Array", http.StatusOK, []byte(`{"status":{"error_code":1}}`)},
		{"Antwort größer als 1 MB", http.StatusOK, oversized},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := serveBody(t, testCase.status, testCase.body)
			icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "BTC"))
			if err == nil {
				t.Fatalf("Fehler erwartet, Icons = %v", icons)
			}
			var rateLimited *coingecko.RateLimitedError
			if errors.As(err, &rateLimited) {
				t.Errorf("darf nicht als Rate-Limit gelten: %v", err)
			}
		})
	}
}

func TestResolveIcons_RateLimit429(t *testing.T) {
	server, _ := serveBody(t, http.StatusTooManyRequests, []byte(`{"status":{"error_code":429}}`))
	_, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "BTC"))
	var rateLimited *coingecko.RateLimitedError
	if !errors.As(err, &rateLimited) {
		t.Fatalf("Fehler = %v (%T), erwartet *RateLimitedError", err, err)
	}
}

func TestResolveIcons_AntwortGenau1MBWirdNichtAlsZuGrossAbgelehnt(t *testing.T) {
	// Grenze: Padding bis exakt 1 MiB gültiges JSON muss noch gelesen werden.
	prefix := `[{"id":"x","symbol":"btc","name":"`
	suffix := `","image":"https://coin-images.coingecko.com/a.png","market_cap_rank":1}]`
	body := prefix + strings.Repeat("a", 1<<20-len(prefix)-len(suffix)) + suffix
	if len(body) != 1<<20 {
		t.Fatalf("Testaufbau: %d Bytes", len(body))
	}
	server, _ := serveBody(t, http.StatusOK, []byte(body))
	icons, err := coingecko.NewClient(server.URL).ResolveIcons(context.Background(), bases(t, "BTC"))
	if err != nil || len(icons) != 1 {
		t.Errorf("icons=%v err=%v, erwartet Erfolg bei genau 1 MiB", icons, err)
	}
}

func TestResolveIcons_ContextTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := coingecko.NewClient(server.URL).ResolveIcons(ctx, bases(t, "BTC"))
	if err == nil {
		t.Fatal("Fehler erwartet bei abgelaufenem Context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Fehler = %v, erwartet context.DeadlineExceeded in der Kette", err)
	}
}

func TestResolveIcons_ServerNichtErreichbar(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	if _, err := coingecko.NewClient(url).ResolveIcons(context.Background(), bases(t, "BTC")); err == nil {
		t.Error("Fehler erwartet")
	}
}
