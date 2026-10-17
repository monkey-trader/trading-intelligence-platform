package httpapi

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 30 * time.Second
	maxHeaderBytes    = 64 << 10
)

// NewServer baut den http.Server mit Timeouts, Header-Limit und ohne Keep-Alive.
func NewServer(handler http.Handler, logger *slog.Logger) *http.Server {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		// Sonst beantwortet Go OPTIONS * selbst mit 200, an Host-Guard und Router vorbei.
		DisableGeneralOptionsHandler: true,
		ConnContext:                  connContext,
	}
	server.SetKeepAlivesEnabled(false)
	return server
}

// connContext legt das Flag des Listener-Wrappers in den Context; ungewrappte Verbindungen
// bleiben ohne Flag.
func connContext(ctx context.Context, conn net.Conn) context.Context {
	malformedTarget := malformedTargetFlagOf(conn)
	if malformedTarget == nil {
		return ctx
	}
	return withMalformedTargetFlag(ctx, malformedTarget)
}

// Serve wrappt den Listener und startet den Server.
func Serve(server *http.Server, listener net.Listener) error {
	return server.Serve(wrapListener(listener))
}
