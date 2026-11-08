package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTokenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadTokenTrimmtZeilenumbruch(t *testing.T) {
	token, err := readToken(writeTokenFile(t, "abcdefghijklmnopqrstuvwxyz\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(token) != "abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("token = %q", token)
	}
}

func TestReadTokenFehlerNennenNieDenInhalt(t *testing.T) {
	cases := map[string]string{
		"ohne Pfad":       "",
		"Datei fehlt":     filepath.Join(t.TempDir(), "gibt-es-nicht"),
		"leere Datei":     writeTokenFile(t, "\n"),
		"zu kurzer Token": writeTokenFile(t, "geheim123"),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readToken(path)
			if err == nil {
				t.Fatal("Fehler erwartet")
			}
			if strings.Contains(err.Error(), "geheim123") {
				t.Fatalf("Fehlermeldung enthält den Token: %v", err)
			}
		})
	}
}

func TestServeHTTPBedientAnfragenUndEndetMitDemKontext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	go func() { done <- serveHTTP(ctx, listener, handler) }()

	resp, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveHTTP endet nicht")
	}
}

func TestRunHTTPScheitertOhneTokenDateiVorDemOeffnenDesPorts(t *testing.T) {
	// Adresse ist absichtlich ungültig: der Token-Fehler muss vor dem Lauschen kommen.
	err := runHTTP(context.Background(), nil, nil, "invalid-addr", "")
	if err == nil || !strings.Contains(err.Error(), "TIP_MCP_TOKEN_FILE") {
		t.Fatalf("err = %v", err)
	}
}
