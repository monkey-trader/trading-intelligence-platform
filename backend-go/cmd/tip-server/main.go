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
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/clock"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/config"
	infraposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/scheduler"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/sqlite"
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

	waitForImport, err := startImport(signalContext, logger, os.Getenv)
	if err != nil {
		return err
	}
	defer waitForImport()

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

// Einstellungen des Hintergrund-Imports.
const (
	importInterval       = 5 * time.Minute
	importRequestTimeout = 15 * time.Second
	importRequestPause   = 200 * time.Millisecond
	importMaxRetries     = 3
	databaseDirMode      = 0o700
)

// startImport startet mit gültigen BingX-Keys den 5-Minuten-Scheduler. Importiert wird ab dem ersten
// Serverstart (`import_started_at`), keine Historie und keine Vollsuche. Ohne Keys passiert nichts. Die zurückgegebene Funktion wartet auf das Ende
// aller Hintergrundläufe (bricht sie vorher ab) und schließt die Datenbank.
func startImport(ctx context.Context, logger *slog.Logger, lookupEnv func(string) string) (func(), error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil {
		return nil, err
	}
	if !configured {
		return func() {}, nil
	}
	databasePath, err := config.ReadDatabasePath(lookupEnv, os.UserConfigDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(databasePath), databaseDirMode); err != nil {
		return nil, err
	}
	database, err := sqlite.OpenWithClock(databasePath, clock.SystemClock{})
	if err != nil {
		return nil, err
	}

	client := bingx.NewClient(bingx.ClientOptions{
		APIKey:         bingXConfig.APIKey,
		APISecret:      bingXConfig.APISecret,
		BaseURL:        bingXConfig.BaseURL,
		MaxRetries:     importMaxRetries,
		RequestTimeout: importRequestTimeout,
		RequestPause:   importRequestPause,
		InitialBackoff: bingXInitialBackoff,
	})
	incomeRepository := sqlite.NewIncomeRepository(database, time.Now)
	state := sqlite.NewImportStateRepository(database)
	positionRepository := sqlite.NewPositionRepository(database, time.Now)
	positionImporter := appposition.NewImportClosedPositionsUseCase(
		infraposition.NewBingXClosedPositionSource(client),
		positionRepository,
		sqlite.NewIncomeLookup(database))
	importStart, err := appposition.ResolveImportStart(ctx, state, time.Now())
	if err != nil {
		_ = database.Close()
		return nil, err
	}
	useCase := appposition.NewScheduledImportUseCase(appposition.ScheduledImportDeps{
		Income:          appposition.NewImportIncomeUseCase(infraposition.NewBingXIncomeSource(client), incomeRepository, state, time.Now),
		Positions:       positionImporter,
		Symbols:         incomeRepository,
		Contracts:       infraposition.NewBingXContractSource(client),
		OpenSource:      infraposition.NewBingXOpenPositionSource(client, clock.SystemClock{}),
		OpenRecorder:    positionRepository,
		State:           state,
		Now:             time.Now,
		FullImportStart: importStart,
	})

	ctx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(importInterval)
	ticks := make(chan struct{})
	var background sync.WaitGroup
	background.Add(2)
	go func() { // Brücke time.Ticker -> chan struct{}
		defer background.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				select {
				case ticks <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	go func() {
		defer background.Done()
		scheduler.New(scheduler.Config{
			Ticks: ticks,
			Job: func(jobContext context.Context) error {
				_, err := useCase.RunIncremental(jobContext)
				return err
			},
			OnError: func(err error) { logger.Error("Import fehlgeschlagen", slog.String("error", err.Error())) },
			OnSkip:  func() { logger.Warn("Import-Tick übersprungen, vorheriger Lauf läuft noch") },
		}).Run(ctx)
	}()

	logger.Info("Import aktiv", slog.Duration("interval", importInterval), slog.Time("seit", time.UnixMilli(importStart)))
	return func() {
		cancel()
		ticker.Stop()
		background.Wait()
		if err := database.Close(); err != nil {
			logger.Error("Datenbank schließen fehlgeschlagen", slog.String("error", err.Error()))
		}
	}, nil
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
