package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewServerProxy_leitetAnfragenAnDenServerWeiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, "pfad="+request.URL.Path+" host="+request.Host)
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)

	recorder := httptest.NewRecorder()
	newServerProxy(target).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/positions/open", nil))

	if got := recorder.Body.String(); got != "pfad=/positions/open host="+target.Host {
		t.Fatalf("Antwort = %q", got)
	}
}

func TestNewServerProxy_zeigtHinweisWennServerNichtLaeuft(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	target, _ := url.Parse(server.URL)
	server.Close() // Port ist danach nicht mehr erreichbar

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	newServerProxy(target).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("Status = %d, erwartet 502", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "brew services start trading-intelligence") {
		t.Fatalf("Hinweis fehlt: %q", recorder.Body.String())
	}
}

func TestServerURL_Standardwert_und_Umgebungsvariable(t *testing.T) {
	t.Setenv("TIP_SERVER_URL", "")
	if got := serverURL().String(); got != "http://127.0.0.1:3000" {
		t.Fatalf("Standard = %q", got)
	}
	t.Setenv("TIP_SERVER_URL", "http://127.0.0.1:3100")
	if got := serverURL().String(); got != "http://127.0.0.1:3100" {
		t.Fatalf("Override = %q", got)
	}
}

func TestNewServerProxy_ApiAnfragenBekommenKeinHtmlBeiFehler(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	target, _ := url.Parse(server.URL)
	server.Close()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/positions/open", nil)
	request.Header.Set("Accept", "application/json")
	newServerProxy(target).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway || strings.Contains(recorder.Body.String(), "<") {
		t.Fatalf("erwartet 502 ohne HTML, bekam %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestNewServerProxy_HinweisseiteEscaptDenHost(t *testing.T) {
	target := &url.URL{Scheme: "http", Host: "<b>x</b>"}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	newServerProxy(target).ServeHTTP(recorder, request)

	if strings.Contains(recorder.Body.String(), "<b>") {
		t.Fatalf("Host nicht escaped: %q", recorder.Body.String())
	}
}

func TestServerURL_LehntFremdeSchemesUndHostsAb(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "ftp://127.0.0.1", "http://example.com:3000", "https://10.0.0.5", "nicht-gueltig"} {
		t.Setenv("TIP_SERVER_URL", raw)
		if got := serverURL().String(); got != defaultServerURL {
			t.Errorf("%q → %q, erwartet Standard", raw, got)
		}
	}
	for _, raw := range []string{"http://localhost:3000", "http://[::1]:3000", "https://127.0.0.1:8443"} {
		t.Setenv("TIP_SERVER_URL", raw)
		if got := serverURL().String(); got != raw {
			t.Errorf("%q → %q, erwartet unverändert", raw, got)
		}
	}
}
