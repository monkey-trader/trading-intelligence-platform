package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	backendBinaryName = "trading-intelligence"
	stopTimeout       = 10 * time.Second
	healthTimeout     = 1 * time.Second
	startupWait       = 15 * time.Second
	startupPoll       = 250 * time.Millisecond
)

// allowedEnvKeys sind die einzigen Variablen, die die App aus ~/.config/tip/env an den Server weitergibt.
var allowedEnvKeys = map[string]bool{"BX_KEY": true, "BX_SECRET": true}

// parseEnvFile liest `KEY=VALUE`-Zeilen (wie die Datei des Brew-Dienstes) und behält nur erlaubte Schlüssel.
func parseEnvFile(content string) []string {
	var entries []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "export ")
		key, value, found := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !found || !allowedEnvKeys[key] {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		entries = append(entries, key+"="+value)
	}
	return entries
}

// backendBinaryPath: Das Server-Binary liegt in <App>.app/Contents/Resources, die App selbst in Contents/MacOS.
func backendBinaryPath(executable string) string {
	return filepath.Join(filepath.Dir(executable), "..", "Resources", backendBinaryName)
}

// serverRunning prüft, ob unter der Adresse schon ein Server antwortet (z.B. ein Brew-Dienst oder `go run`).
func serverRunning(target *url.URL) bool {
	client := &http.Client{Timeout: healthTimeout}
	response, err := client.Get(target.JoinPath("health").String())
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

// backendProcess ist der von der App gestartete Server.
type backendProcess struct {
	command *exec.Cmd
	done    chan struct{}
}

// startBackend startet das Binary mit genau der übergebenen Umgebung (plus HOME für den Standardpfad der DB).
func startBackend(binary string, environment []string, logPath string) (*backendProcess, error) {
	command := exec.Command(binary)
	command.Env = append([]string{"HOME=" + os.Getenv("HOME")}, environment...)
	if logPath != "" {
		if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			_ = logFile.Chmod(0o600) // OpenFile setzt die Rechte nur beim Anlegen
			command.Stdout, command.Stderr = logFile, logFile
			defer func() { _ = logFile.Close() }() // der Kindprozess hält eine eigene Kopie des Handles
		}
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("server starten: %w", err)
	}
	process := &backendProcess{command: command, done: make(chan struct{})}
	go func() { _ = command.Wait(); close(process.done) }()
	return process, nil
}

// stop beendet den Server sauber (SIGTERM, danach hartes Ende nach stopTimeout).
func (process *backendProcess) stop() {
	if process == nil {
		return
	}
	_ = process.command.Process.Signal(syscall.SIGTERM)
	select {
	case <-process.done:
	case <-time.After(stopTimeout):
		_ = process.command.Process.Kill()
		<-process.done
	}
}

// waitUntilRunning wartet, bis der Server auf /health antwortet oder der Kontext endet.
func waitUntilRunning(ctx context.Context, target *url.URL) bool {
	deadline := time.After(startupWait)
	for !serverRunning(target) {
		select {
		case <-ctx.Done():
			return false
		case <-deadline:
			return false
		case <-time.After(startupPoll):
		}
	}
	return true
}
