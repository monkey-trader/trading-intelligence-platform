// Command trading-intelligence startet die HTTP-API der Trading Intelligence Platform.
package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
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
	_ "time/tzdata" // Zeitzonen eingebettet, damit Europe/Zurich ohne System-tzdata geladen wird

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
	appasset "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/asset"
	appchart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/chart"
	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart/interfaces"
	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	infraaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/account"
	infraasset "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/asset"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
	infrachart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/clock"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/coingecko"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/config"
	infraposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/scheduler"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/sqlite"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/presentation/httpapi"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/presentation/webui"
)

// shutdownTimeout begrenzt, wie lange laufende Anfragen beim Beenden noch abgeschlossen werden.
// coinGeckoBaseURL ist die öffentliche CoinGecko-API für Coin-Icons.
const coinGeckoBaseURL = "https://api.coingecko.com"

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
	database, err := openDatabaseIfConfigured(os.Getenv)
	if err != nil {
		return err
	}
	if database != nil {
		defer closeDatabase(logger, database)
	}
	orderSource, err := newProtectiveOrderSource(os.Getenv)
	if err != nil {
		return err
	}
	chartClient, err := newChartClient(os.Getenv, database)
	if err != nil {
		return err
	}
	// Ein gemeinsamer Resolver (Cooldown, Budget, Negativ-Cache) für Positionsliste und Chart.
	var riskResolver *appposition.RiskResolver
	if chartClient != nil {
		riskResolver = appposition.NewRiskResolver(sqlite.NewPositionRiskRepository(database), time.Now().UnixMilli).
			WithHistory(infrachart.NewBingXStopOrderHistorySource(chartClient))
	}
	// Manueller Ursprungs-SL: nur mit Datenbank und Resolver; der gemeinsame Resolver liest ihn live.
	settings := httpapi.Settings{AllowRemote: os.Getenv("ALLOW_REMOTE") == "true"}
	if riskResolver != nil {
		overrides := sqlite.NewPositionInitialStopLossRepository(database)
		riskResolver.WithOverrides(overrides)
		setStopLoss := appposition.NewSetInitialStopLossUseCase(sqlite.NewPositionEntryLookup(database), overrides, riskResolver, time.Now().UnixMilli)
		clearStopLoss := appposition.NewClearInitialStopLossUseCase(overrides, riskResolver)
		settings.SetInitialStopLoss, settings.ClearInitialStopLoss = setStopLoss.Execute, clearStopLoss.Execute
	} else {
		// Ohne BingX-Zugang oder Datenbank bleiben die Routen registriert (Anfrageprüfung greift) und liefern 503 not-configured.
		settings.SetInitialStopLoss = func(context.Context, string, string) (appposition.SetInitialStopLossResult, error) {
			return appposition.SetInitialStopLossResult{}, &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}
		}
		settings.ClearInitialStopLoss = func(context.Context, string) error {
			return &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}
		}
	}
	getOpenPositions := newGetOpenPositions(openPositionSource, orderSource, database, riskResolver)
	getPositionChart := newGetPositionChart(logger, chartClient, openPositionSource, orderSource, database, riskResolver)

	iconClock := clock.SystemClock{}
	getAssetIcons := appasset.NewGetAssetIconsUseCase(
		coingecko.NewClient(coinGeckoBaseURL), infraasset.NewMemoryIconCache(iconClock), iconClock)

	listener, err := net.Listen("tcp", net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port)))
	if err != nil {
		return err
	}

	settings.AllowedHosts = serverConfig.AllowedHosts
	settings.GetOpenPositions = getOpenPositions.Execute
	settings.GetPositionChart = getPositionChart
	settings.GetAssetIcons = getAssetIcons.Execute
	getAccountSummary, err := newGetAccountSummary(chartClient, database)
	if err != nil {
		return err
	}
	settings.GetAccountSummary = getAccountSummary
	if database != nil {
		getClosedPositions := appposition.NewGetClosedPositionsUseCase(
			sqlite.NewPositionRepository(database, time.Now), sqlite.NewPositionRiskRepository(database))
		settings.GetClosedPositions = getClosedPositions.Execute
	} else {
		settings.GetClosedPositions = func(context.Context, domainposition.ClosedPositionQuery) ([]appposition.ClosedPositionView, error) {
			return nil, &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}
		}
	}
	settings.WebUI = newWebUI()
	handler := httpapi.NewHandler(settings, logger)
	server := httpapi.NewServer(handler, logger)

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	waitForImport, err := startImport(signalContext, logger, os.Getenv, database)
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
// aller Hintergrundläufe (bricht sie vorher ab). Die Datenbank gehört dem Aufrufer (nil = nicht konfiguriert).
func startImport(ctx context.Context, logger *slog.Logger, lookupEnv func(string) string, database *sql.DB) (func(), error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil {
		return nil, err
	}
	if !configured || database == nil {
		return func() {}, nil
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
		return nil, err
	}
	incomeImporter := appposition.NewImportIncomeUseCase(infraposition.NewBingXIncomeSource(client), incomeRepository, state, time.Now)
	useCase := appposition.NewScheduledImportUseCase(appposition.ScheduledImportDeps{
		Income:          incomeImporter,
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
	background.Add(3)
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
	go func() { // Einmalig: Income ab Monatsanfang nachladen, damit der Monats-PnL vollständig ist.
		defer background.Done()
		location, locationErr := time.LoadLocation(accountTimezone)
		if locationErr != nil {
			logger.Error("Income-Historie übersprungen", slog.String("error", locationErr.Error()))
			return
		}
		now := time.Now().In(location)
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
		historySince := monthStart.UnixMilli()
		if thirtyDaysAgo := now.AddDate(0, 0, -30).UnixMilli(); thirtyDaysAgo < historySince {
			historySince = thirtyDaysAgo
		}
		if historyErr := incomeImporter.EnsureHistorySince(ctx, historySince); historyErr != nil && ctx.Err() == nil {
			logger.Error("Income-Historie nachladen fehlgeschlagen", slog.String("error", historyErr.Error()))
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
	}, nil
}

// openDatabaseIfConfigured öffnet die Datenbank nur mit gültigen BingX-Keys; sonst nil.
func openDatabaseIfConfigured(lookupEnv func(string) string) (*sql.DB, error) {
	_, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil || !configured {
		return nil, err
	}
	databasePath, err := config.ReadDatabasePath(lookupEnv, os.UserConfigDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(databasePath), databaseDirMode); err != nil {
		return nil, err
	}
	return sqlite.OpenWithClock(databasePath, clock.SystemClock{})
}

func closeDatabase(logger *slog.Logger, database *sql.DB) {
	if err := database.Close(); err != nil {
		logger.Error("Datenbank schließen fehlgeschlagen", slog.String("error", err.Error()))
	}
}

// newChartClient baut den BingX-Client für Candles und Order-Historie. Ohne Keys oder Datenbank ist er nil.
func newChartClient(lookupEnv func(string) string, database *sql.DB) (*bingx.Client, error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil {
		return nil, err
	}
	if !configured || database == nil {
		return nil, nil
	}
	return bingx.NewClient(bingx.ClientOptions{
		APIKey:         bingXConfig.APIKey,
		APISecret:      bingXConfig.APISecret,
		BaseURL:        bingXConfig.BaseURL,
		MaxRetries:     bingXMaxRetries,
		RequestTimeout: bingXRequestTimeout,
		RequestPause:   bingXRequestPause,
		InitialBackoff: bingXInitialBackoff,
	}), nil
}

// newGetPositionChart verdrahtet den Chart-Use-Case. Ohne Client (keine Keys oder keine DB) antwortet die
// Funktion immer not-configured, damit die Route wie GET /positions/open mit 503 antwortet.
func newGetPositionChart(
	logger *slog.Logger,
	client *bingx.Client,
	openPositionSource interfaces.OpenPositionSource,
	orderSource chartinterfaces.ProtectiveOrderSource,
	database *sql.DB,
	riskResolver *appposition.RiskResolver,
) func(context.Context, appchart.Query) (appchart.PositionChart, error) {
	if client == nil {
		return func(context.Context, appchart.Query) (appchart.PositionChart, error) {
			return appchart.PositionChart{}, &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}
		}
	}
	return appchart.NewGetPositionChartUseCase(
		openPositionSource,
		orderSource,
		infrachart.NewBingXCandleSource(client),
		sqlite.NewCandleRepository(database),
		logger).
		WithRiskResolver(riskResolver, sqlite.NewPositionRiskRepository(database), time.Now().UnixMilli).Execute
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

// newWebUI liefert das eingebettete UI, oder nil, wenn nur der Platzhalter eingebettet ist
// (Entwicklungs-Build ohne UI): dann gilt für unbekannte Pfade der API-Vertrag (404 not-found).
func newWebUI() http.Handler {
	files := webui.EmbeddedFiles()
	if _, err := fs.Stat(files, "index.html"); err != nil {
		return nil
	}
	return webui.NewHandler(files)
}

// newGetOpenPositions nutzt SL/TP/Break-even nur, wenn eine Orders-Source existiert (Keys gesetzt),
// und die R-Werte nur mit Datenbank.
func newGetOpenPositions(source interfaces.OpenPositionSource, orderSource chartinterfaces.ProtectiveOrderSource, database *sql.DB, riskResolver *appposition.RiskResolver) *appposition.GetOpenPositionsUseCase {
	var useCase *appposition.GetOpenPositionsUseCase
	if orderSource == nil {
		useCase = appposition.NewGetOpenPositionsUseCase(source)
	} else {
		useCase = appposition.NewGetOpenPositionsUseCaseWithProtection(source, orderSource)
	}
	if database == nil {
		return useCase
	}
	if riskResolver == nil {
		return useCase.WithRisk(sqlite.NewPositionRiskRepository(database), sqlite.NewCandleRepository(database), time.Now().UnixMilli)
	}
	return useCase.WithRiskResolver(riskResolver, sqlite.NewPositionRiskRepository(database), sqlite.NewCandleRepository(database), time.Now().UnixMilli)
}

// newProtectiveOrderSource baut BingX-Orders-Source mit 5-s-Cache, der von Chart und Positionsliste geteilt wird.
// Ohne Keys ist das Ergebnis nil.
func newProtectiveOrderSource(lookupEnv func(string) string) (chartinterfaces.ProtectiveOrderSource, error) {
	bingXConfig, configured, err := config.ReadBingXConfig(lookupEnv)
	if err != nil || !configured {
		return nil, err
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
	return infrachart.NewCachingProtectiveOrderSource(
		infrachart.NewBingXProtectiveOrderSource(client, systemClock), systemClock, openPositionsTTL), nil
}

// accountTimezone bestimmt die Monatsgrenzen der Kontoübersicht.
const accountTimezone = "Europe/Zurich"

// accountBalanceTTL cached die Balance wie die offenen Positionen.
const accountBalanceTTL = openPositionsTTL

// newGetAccountSummary verdrahtet GET /account/summary. Ohne Client (keine Keys oder keine DB)
// antwortet die Funktion immer not-configured (503).
func newGetAccountSummary(client *bingx.Client, database *sql.DB) (func(context.Context) (appaccount.AccountSummary, error), error) {
	if client == nil {
		return func(context.Context) (appaccount.AccountSummary, error) {
			return appaccount.AccountSummary{}, &domainposition.OpenPositionsUnavailableError{Reason: domainposition.ReasonNotConfigured}
		}, nil
	}
	location, err := time.LoadLocation(accountTimezone)
	if err != nil {
		return nil, err
	}
	source := infraaccount.NewBingXBalanceSource(client)
	useCase := appaccount.NewGetAccountSummaryUseCase(
		infraaccount.NewCachingBalanceSource(source, time.Now, accountBalanceTTL),
		sqlite.NewIncomeRepository(database, time.Now), func() int64 { return time.Now().UnixMilli() }, location)
	return useCase.Execute, nil
}
