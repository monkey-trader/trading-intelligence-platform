package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseEnvFile_liestNurBekannteSchluessel(t *testing.T) {
	content := "# Kommentar\n\nBX_KEY=abc\nexport BX_SECRET=\"geheim\"\nANDERES=x\nPORT=1\nkaputt\n"

	got := parseEnvFile(content)

	want := []string{"BX_KEY=abc", "BX_SECRET=geheim"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseEnvFile = %v, erwartet %v", got, want)
	}
}

func TestBackendBinaryPath_liegtImResourcesOrdnerDerApp(t *testing.T) {
	got := backendBinaryPath("/Applications/Trading Intelligence.app/Contents/MacOS/TIP")

	want := "/Applications/Trading Intelligence.app/Contents/Resources/trading-intelligence"
	if got != want {
		t.Fatalf("backendBinaryPath = %q, erwartet %q", got, want)
	}
}

func TestServerRunning(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/health" {
			http.NotFound(writer, request)
		}
	}))
	defer healthy.Close()
	target, _ := url.Parse(healthy.URL)
	if !serverRunning(target) {
		t.Fatal("laufender Server nicht erkannt")
	}

	closed := httptest.NewServer(http.NotFoundHandler())
	closedTarget, _ := url.Parse(closed.URL)
	closed.Close()
	if serverRunning(closedTarget) {
		t.Fatal("nicht erreichbarer Server als laufend erkannt")
	}
}

func TestStartBackend_startetMitUmgebungUndStopptPerSigterm(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "env.txt")
	script := filepath.Join(dir, "fake-backend")
	body := "#!/bin/sh\necho \"$PORT $BX_KEY\" > '" + marker + "'\ntrap 'exit 0' TERM\nwhile true; do sleep 0.1; done\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	process, err := startBackend(script, []string{"PORT=3999", "BX_KEY=abc"}, filepath.Join(dir, "server.log"))
	if err != nil {
		t.Fatalf("startBackend: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		if data, readErr := os.ReadFile(marker); readErr == nil && len(data) > 0 {
			if string(data) != "3999 abc\n" {
				t.Fatalf("Umgebung = %q", data)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Prozess hat nicht gestartet")
		}
		time.Sleep(20 * time.Millisecond)
	}

	stopped := make(chan struct{})
	go func() { process.stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop() kehrt nicht zurück")
	}
}

func TestStartBackend_meldetFehlendesBinary(t *testing.T) {
	if _, err := startBackend(filepath.Join(t.TempDir(), "gibt-es-nicht"), nil, ""); err == nil {
		t.Fatal("Fehler erwartet")
	}
}
