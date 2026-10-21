package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

const defaultServerURL = "http://127.0.0.1:3000"

// serverURL liefert die Adresse des trading-intelligence (TIP_SERVER_URL überschreibt den Standard).
func serverURL() *url.URL {
	raw := os.Getenv("TIP_SERVER_URL")
	if raw == "" {
		raw = defaultServerURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || !isLoopbackHTTP(parsed) {
		log.Printf("TIP_SERVER_URL %q abgelehnt (nur http/https auf Loopback), nutze %s", raw, defaultServerURL)
		parsed, _ = url.Parse(defaultServerURL)
	}
	return parsed
}

// isLoopbackHTTP: Die API hat keine Authentifizierung, die App darf sie nur lokal ansprechen.
func isLoopbackHTTP(target *url.URL) bool {
	if target.Scheme != "http" && target.Scheme != "https" {
		return false
	}
	switch target.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

const serverDownPage = `<!doctype html><meta charset="utf-8"><title>trading-intelligence nicht erreichbar</title>
<body style="font-family:-apple-system,sans-serif;display:grid;place-items:center;height:100vh;margin:0;background:#282828;color:#ebdbb2">
<main style="text-align:center"><h1>Server nicht erreichbar</h1>
<p>Der Dienst läuft nicht. Im Terminal starten:</p>
<pre>brew services start trading-intelligence</pre>
<p><small>%s</small></p></main>`

// newServerProxy reicht alle Anfragen des Fensters an den trading-intelligence weiter.
// So bleibt die App ein reiner Client der HTTP-API (API-first).
func newServerProxy(target *url.URL) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.Host = target.Host // der Host-Guard des Servers lässt nur Loopback-Hosts zu
		},
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, _ error) {
		if !strings.Contains(request.Header.Get("Accept"), "text/html") {
			http.Error(writer, "trading-intelligence nicht erreichbar", http.StatusBadGateway) // API-Aufrufe bekommen kein HTML
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(writer, serverDownPage, html.EscapeString(target.Host))
	}
	return proxy
}
