// Command tip-server startet die HTTP-API der Trading Intelligence Platform.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/config"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/presentation/httpapi"
)

// shutdownTimeout begrenzt, wie lange laufende Anfragen beim Beenden noch abgeschlossen werden.
const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("Server beendet mit Fehler", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	serverConfig, err := config.ReadServerConfig(os.Getenv)
	if err != nil {
		return err
	}
	logMissingBingXCredentials(logger, config.MissingBingXVariables(os.Getenv))

	listener, err := net.Listen("tcp", net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port)))
	if err != nil {
		return err
	}

	handler := httpapi.NewHandler(httpapi.Settings{AllowedHosts: serverConfig.AllowedHosts}, logger)
	server := httpapi.NewServer(handler, logger)

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serveResult := make(chan error, 1)
	go func() { serveResult <- httpapi.Serve(server, listener) }()
	logger.Info("API läuft", slog.String("address", listener.Addr().String()))

	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signalContext.Done():
		// Signalbehandlung sofort zurücksetzen: Ein zweites Ctrl+C beendet den Prozess dann
		// hart, statt auf das Ende des Shutdowns zu warten.
		stopSignals()
		logger.Info("Signal empfangen, Server wird beendet")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}
	if err := <-serveResult; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// logMissingBingXCredentials gibt nur einen Hinweis mit den Variablennamen aus, nie Werte.
// Die BingX-Anbindung folgt in Phase 2.
func logMissingBingXCredentials(logger *slog.Logger, missingVariables []string) {
	if len(missingVariables) > 0 {
		logger.Warn("BingX-Zugang nicht konfiguriert; GET /positions/open liefert 503 not-configured",
			slog.Any("missingVariables", missingVariables))
	}
}
