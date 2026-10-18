// Package webui liefert das eingebettete Web-UI (SPA) aus.
package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embeddedDist embed.FS

// EmbeddedFiles liefert den eingebetteten Ordner dist (im Quellbaum nur .gitkeep, im Release das gebaute UI).
func EmbeddedFiles() fs.FS {
	sub, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		// Kann nur passieren, wenn der Ordner nicht eingebettet ist; dann ist das Web-UI leer.
		return embed.FS{}
	}
	return sub
}

const notEmbeddedMessage = "Web-UI nicht eingebettet"

// NewHandler liefert Dateien aus files aus, mit SPA-Fallback auf index.html für Pfade ohne Dateiendung.
func NewHandler(files fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Methode nicht erlaubt", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if serveFile(w, r, files, name) {
			return
		}
		if path.Ext(name) != "" && name != "index.html" {
			http.NotFound(w, r)
			return
		}
		if !serveFile(w, r, files, "index.html") {
			http.Error(w, notEmbeddedMessage, http.StatusNotFound)
		}
	})
}

// setCacheControl: index.html muss immer neu validiert werden; gehashte Vite-Dateien unter assets/ ändern sich nie.
func setCacheControl(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

// serveFile liefert eine reguläre Datei aus; false, wenn sie fehlt oder ein Ordner ist.
func serveFile(w http.ResponseWriter, r *http.Request, files fs.FS, name string) bool {
	file, err := files.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	setCacheControl(w, name)
	seeker, ok := file.(interface {
		fs.File
		Seek(offset int64, whence int) (int64, error)
	})
	if !ok {
		data, readErr := fs.ReadFile(files, name)
		if readErr != nil {
			return false
		}
		http.ServeContent(w, r, name, info.ModTime(), bytes.NewReader(data))
		return true
	}
	http.ServeContent(w, r, name, info.ModTime(), seeker)
	return true
}
