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

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/clock"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/config"
	infraposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/presentation/httpapi"
)

// shutdownTimeout begrenzt, wie lange laufende Anfragen beim Beenden noch abgeschlossen werden.
const shutdownTimeout = 10 * time.Second

// Einstellungen für GET /positions/open. Gesamtfrist: FetchTimeout 8 s liegt unter dem
// WriteTimeout des Servers (10 s), damit immer noch eine Problem-Antwort geschrieben werden kann.
const (
	bingXMaxRetries     = 1
	bingXRequestTimeout = 5 * time.Second
	bingXRequestPause   = 0
	bingXInitialBackoff = 1 * time.Second
	openPositionsTTL    = 5 * time.Second
	rateLimitHold       = 30 * time.Second
	openPositionsFetch  = 8 * time.Second
)

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
	openPositionSource, err := newOpenPositionSource(logger, os.Getenv)
	if err != nil {
		return err
	}
	getOpenPositions := appposition.NewGetOpenPositionsUseCase(openPositionSource)

	listener, err := net.Listen("tcp", net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port)))
	if err != nil {
		return err
	}

	handler := httpapi.NewHandler(httpapi.Settings{
		AllowedHosts:     serverConfig.AllowedHosts,
		GetOpenPositions: getOpenPositions.Execute,
	}, logger)
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

// newOpenPositionSource baut mit Keys BingX-Client → BingX-Source → Cache, ohne Keys die
// Unconfigured-Source (ohne Cache). Ein ungültiges BX_BASE ist ein Startfehler.
func newOpenPositionSource(logger *slog.Logger, lookupEnv func(string) string) (interfaces.OpenPositionSource, error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil {
		return nil, err
	}
	if !configured {
		logMissingBingXCredentials(logger, config.MissingBingXVariables(lookupEnv))
		return infraposition.UnconfiguredOpenPositionSource{}, nil
	}

	client := bingx.NewClient(bingx.ClientOptions{
		APIKey:         bingXConfig.APIKey,
		APISecret:      bingXConfig.APISecret,
		BaseURL:        bingXConfig.BaseURL,
		MaxRetries:     bingXMaxRetries,
		RequestTimeout: bingXRequestTimeout,
		RequestPause:   bingXRequestPause,
		InitialBackoff: bingXInitialBackoff,
	})
	systemClock := clock.SystemClock{}
	source := infraposition.NewBingXOpenPositionSource(client, systemClock)
	return infraposition.NewCachingSource(source, systemClock, infraposition.CacheOptions{
		TTL:           openPositionsTTL,
		RateLimitHold: rateLimitHold,
		FetchTimeout:  openPositionsFetch,
	}), nil
}

// logMissingBingXCredentials gibt nur einen Hinweis mit den Variablennamen aus, nie Werte.
func logMissingBingXCredentials(logger *slog.Logger, missingVariables []string) {
	if len(missingVariables) > 0 {
		logger.Warn("BingX-Zugang nicht konfiguriert; GET /positions/open liefert 503 not-configured",
			slog.Any("missingVariables", missingVariables))
	}
}
