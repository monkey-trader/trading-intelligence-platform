package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App hält den Wails-Kontext. Fachlogik gibt es hier nicht: Das UI spricht über die HTTP-API.
type App struct {
	ctx context.Context
}

func (app *App) startup(ctx context.Context) {
	app.ctx = ctx
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
