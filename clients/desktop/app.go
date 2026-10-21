package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App hält den Wails-Kontext. Fachlogik gibt es hier nicht: Das UI spricht über die HTTP-API.
type App struct {
	ctx          context.Context
	backendMutex sync.Mutex // schützt backend: startup läuft nebenläufig zu shutdown
	backend      *backendProcess
}

func (app *App) startup(ctx context.Context) {
	app.ctx = ctx
	go app.ensureBackend(ctx, serverURL())
	path, err := windowStatePath()
	if err != nil {
		return
	}
	if state, ok := loadWindowState(path); ok {
		runtime.WindowSetSize(ctx, state.Width, state.Height)
		runtime.WindowSetPosition(ctx, state.X, state.Y)
	}
}

// beforeClose merkt Größe und Position fürs nächste Öffnen; Schließen wird nie verhindert.
func (app *App) beforeClose(ctx context.Context) bool {
	if path, err := windowStatePath(); err == nil && !runtime.WindowIsFullscreen(ctx) {
		width, height := runtime.WindowGetSize(ctx)
		x, y := runtime.WindowGetPosition(ctx)
		_ = saveWindowState(path, windowState{Width: width, Height: height, X: x, Y: y})
	}
	return false
}

func (app *App) reload() {
	runtime.WindowReload(app.ctx)
}

// ensureBackend startet den mitgelieferten Server, falls unter der Adresse noch keiner läuft.
// Eine eigene TIP_SERVER_URL (z.B. Vite) wird nie angefasst; dann ist der Nutzer für den Server zuständig.
func (app *App) ensureBackend(ctx context.Context, target *url.URL) {
	if os.Getenv("TIP_SERVER_URL") != "" || serverRunning(target) {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}
	home, _ := os.UserHomeDir()
	environment := []string{"PORT=" + target.Port()}
	if data, readErr := os.ReadFile(filepath.Join(home, ".config", "tip", "env")); readErr == nil {
		environment = append(environment, parseEnvFile(string(data))...)
	}
	process, err := startBackend(backendBinaryPath(executable), environment, serverLogPath())
	if err != nil {
		return // das Fenster zeigt die Hinweisseite
	}
	app.backendMutex.Lock()
	app.backend = process
	app.backendMutex.Unlock()
	if waitUntilRunning(ctx, target) {
		runtime.WindowReload(ctx)
	}
}

// shutdown stoppt den Server, den diese App gestartet hat.
func (app *App) shutdown(_ context.Context) {
	app.backendMutex.Lock()
	defer app.backendMutex.Unlock()
	app.backend.stop()
}

// serverLogPath liegt neben window.json; leer, wenn das Verzeichnis nicht angelegt werden kann.
func serverLogPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(configDir, "tip")
	if os.MkdirAll(dir, 0o700) != nil {
		return ""
	}
	return filepath.Join(dir, "server.log")
}
