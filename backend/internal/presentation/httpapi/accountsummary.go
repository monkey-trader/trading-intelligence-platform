package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
)

// accountSummaryDto ist die JSON-Form von GET /account/summary.
type accountSummaryDto struct {
	Balance            string         `json:"balance"`
	Equity             string         `json:"equity"`
	AvailableMargin    string         `json:"availableMargin"`
	Month              string         `json:"month"`
	MonthlyRealizedPnl string         `json:"monthlyRealizedPnl"`
	MonthlyPnlPercent  *string        `json:"monthlyPnlPercent"`
	Periods            []periodPnlDto `json:"periods"`

	UnrealizedPnl        string  `json:"unrealizedPnl"`
	UnrealizedPnlPercent *string `json:"unrealizedPnlPercent"`
}

// periodPnlDto ist ein Eintrag in "periods".
type periodPnlDto struct {
	Period      string  `json:"period"`
	RealizedPnl string  `json:"realizedPnl"`
	PnlPercent  *string `json:"pnlPercent"`
}

// newAccountSummaryHandler beantwortet GET /account/summary; die Fehlerursache landet nur im Log.
func newAccountSummaryHandler(
	getSummary func(ctx context.Context) (appaccount.AccountSummary, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := getSummary(r.Context())
		if err != nil {
			problemType := problemTypeFor(err)
			level := slog.LevelWarn
			if errors.Is(err, context.Canceled) {
				level = slog.LevelInfo
			}
			logger.Log(r.Context(), level, "Kontoübersicht nicht verfügbar",
				slog.String("problem", string(problemType)),
				slog.String("error", err.Error()),
				slog.String("cause", causeText(err)))
			if problemType == ProblemRateLimited {
				w.Header().Set("Retry-After", rateLimitRetryAfterSeconds)
			}
			writeProblem(w, problemType)
			return
		}
		periods := make([]periodPnlDto, 0, len(summary.Periods))
		for _, period := range summary.Periods {
			periods = append(periods, periodPnlDto(period))
		}
		body, err := json.Marshal(accountSummaryDto{
			Balance: summary.Balance, Equity: summary.Equity, AvailableMargin: summary.AvailableMargin,
			Month: summary.Month, MonthlyRealizedPnl: summary.MonthlyRealizedPnl,
			MonthlyPnlPercent: summary.MonthlyPnlPercent, Periods: periods,
			UnrealizedPnl: summary.UnrealizedPnl, UnrealizedPnlPercent: summary.UnrealizedPnlPercent,
		})
		if err != nil {
			logger.Error("Kontoübersicht konnte nicht serialisiert werden", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
