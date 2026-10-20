package httpapi

import (
	"context"
	"log/slog"
	"maps"
	"net/http"
	"path"
	"strings"

	appchart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/chart"
	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

// Settings enthält die Einstellungen der HTTP-API.
type Settings struct {
	AllowedHosts []string
	// GetOpenPositions liefert die offenen Positionen (Pflicht). Fehler werden per errors.As auf
	// *position.OpenPositionsUnavailableError in Problem-Antworten übersetzt.
	GetOpenPositions func(ctx context.Context) (appposition.OpenPositionsOverview, error)
	// GetPositionChart liefert den Chart einer offenen Position (optional; nil = Route nicht registriert).
	GetPositionChart func(ctx context.Context, query appchart.Query) (appchart.PositionChart, error)
	// GetAssetIcons liefert Coin-Icons (optional; nil = Route nicht registriert).
	GetAssetIcons func(ctx context.Context, list asset.SymbolList) ([]asset.CoinIcon, error)
	// WebUI bedient alle Pfade, die die API nicht kennt (optional; nil = 404 not-found).
	WebUI http.Handler
}

// NewHandler baut die Kette recover → hostGuard → malformedTarget → router → 404.
func NewHandler(settings Settings, logger *slog.Logger) http.Handler {
	return newHandlerWithRoutes(settings, logger, nil)
}

// newHandlerWithRoutes wie NewHandler, ergänzt aber die Standardrouten um extraRoutes
// (Schlüssel "METHOD /pfad"). Nur für Tests, z.B. eine Route, die panict.
func newHandlerWithRoutes(settings Settings, logger *slog.Logger, extraRoutes map[string]http.HandlerFunc) http.Handler {
	if settings.GetOpenPositions == nil {
		// Programmierfehler beim Verdrahten: sofort beim Start auffallen statt erst beim ersten Request.
		panic("httpapi: Settings.GetOpenPositions ist Pflicht")
	}
	routes := map[string]http.HandlerFunc{
		"GET /health":         handleHealth,
		"GET /positions/open": newOpenPositionsHandler(settings.GetOpenPositions, logger),
	}
	if settings.GetAssetIcons != nil {
		routes["GET /assets/icons"] = newAssetIconsHandler(settings.GetAssetIcons, logger)
	}
	maps.Copy(routes, extraRoutes)
	var chartRoute http.HandlerFunc
	if settings.GetPositionChart != nil {
		chartRoute = newPositionChartHandler(settings.GetPositionChart, logger)
	}
	allowedHosts := append([]string(nil), settings.AllowedHosts...)

	return withRecover(logger, withHostGuard(allowedHosts, withMalformedTarget(newRouter(routes, chartRoute, settings.WebUI))))
}

// newRouter sucht die Route exakt über "METHOD /pfad". Alles andere ergibt 404 not-found;
// der Request-Body wird dabei nie gelesen.
// chartRoute (optional) bedient GET /positions/{positionId}/chart per Pfad-Matching.
func newRouter(routes map[string]http.HandlerFunc, chartRoute http.HandlerFunc, webUI http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, found := routes[r.Method+" "+r.URL.Path]
		if _, isChartPath := chartPathSegment(r.URL.Path); !found && chartRoute != nil && r.Method == http.MethodGet && isChartPath {
			route, found = chartRoute, true
		}
		if !found {
			if webUI != nil && wantsWebUI(r) {
				webUI.ServeHTTP(w, r)
				return
			}
			writeProblem(w, ProblemNotFound)
			return
		}
		route(w, r)
	})
}

// apiPathPrefixes sind Pfade, die nie vom Web-UI bedient werden (immer 404 not-found).
var apiPathPrefixes = []string{"/health", "/positions", "/assets", "/api"}

// wantsWebUI: Das UI bedient nur GET/HEAD, nie API-Pfade, und nur "/", Dateien (mit Endung) und
// Browser-Navigationen (Accept: text/html). Alles andere bleibt 404 not-found im API-Vertrag.
func wantsWebUI(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if isAPIPath(r.URL.Path) {
		return false
	}
	return r.URL.Path == "/" || path.Ext(r.URL.Path) != "" || strings.Contains(r.Header.Get("Accept"), "text/html")
}

func isAPIPath(urlPath string) bool {
	for _, prefix := range apiPathPrefixes {
		if urlPath == prefix || strings.HasPrefix(urlPath, prefix+"/") {
			return true
		}
	}
	return false
}
