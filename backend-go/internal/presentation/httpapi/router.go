package httpapi

import (
	"context"
	"log/slog"
	"maps"
	"net/http"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/application/position"
)

// Settings enthält die Einstellungen der HTTP-API.
type Settings struct {
	AllowedHosts []string
	// GetOpenPositions liefert die offenen Positionen (Pflicht). Fehler werden per errors.As auf
	// *position.OpenPositionsUnavailableError in Problem-Antworten übersetzt.
	GetOpenPositions func(ctx context.Context) (appposition.OpenPositionsOverview, error)
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
	maps.Copy(routes, extraRoutes)
	allowedHosts := append([]string(nil), settings.AllowedHosts...)

	return withRecover(logger, withHostGuard(allowedHosts, withMalformedTarget(newRouter(routes))))
}

// newRouter sucht die Route exakt über "METHOD /pfad". Alles andere ergibt 404 not-found;
// der Request-Body wird dabei nie gelesen.
func newRouter(routes map[string]http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, found := routes[r.Method+" "+r.URL.Path]
		if !found {
			writeProblem(w, ProblemNotFound)
			return
		}
		route(w, r)
	})
}
