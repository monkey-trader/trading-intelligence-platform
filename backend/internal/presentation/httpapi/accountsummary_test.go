package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Erwartet: Settings.GetAccountSummary func(ctx) (appaccount.AccountSummary, error), optional (nil = Route nicht registriert).

func accountSettings(fn func(context.Context) (appaccount.AccountSummary, error)) Settings {
	settings := settingsWith(func(context.Context) (appposition.OpenPositionsOverview, error) {
		return appposition.OpenPositionsOverview{}, nil
	})
	settings.GetAccountSummary = fn
	return settings
}

func getAccountSummary(t *testing.T, settings Settings) *httptest.ResponseRecorder {
	t.Helper()
	return serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/account/summary", allowedTestHost))
}

func TestAccountSummary_Erfolg(t *testing.T) {
	percent := "9.71"
	testCases := []struct {
		name     string
		summary  appaccount.AccountSummary
		expected string
	}{
		{"mit Prozent", appaccount.AccountSummary{Balance: "1000", Equity: "1259.5", AvailableMargin: "1150.25",
			Month: "2026-10", MonthlyRealizedPnl: "97.125", MonthlyPnlPercent: &percent,
			UnrealizedPnl: "25", UnrealizedPnlPercent: &percent,
			Periods: []appaccount.PeriodPnl{
				{Period: "today", RealizedPnl: "10", PnlPercent: &percent},
				{Period: "7d", RealizedPnl: "0", PnlPercent: nil},
				{Period: "30d", RealizedPnl: "-5.5", PnlPercent: &percent},
			}},
			`{"balance":"1000","equity":"1259.5","availableMargin":"1150.25","month":"2026-10","monthlyRealizedPnl":"97.125","monthlyPnlPercent":"9.71",` +
				`"periods":[{"period":"today","realizedPnl":"10","pnlPercent":"9.71"},{"period":"7d","realizedPnl":"0","pnlPercent":null},{"period":"30d","realizedPnl":"-5.5","pnlPercent":"9.71"}],"unrealizedPnl":"25","unrealizedPnlPercent":"9.71"}`},
		{"Prozent null", appaccount.AccountSummary{Balance: "0", Equity: "0", AvailableMargin: "0",
			Month: "2026-10", MonthlyRealizedPnl: "0", UnrealizedPnl: "0"},
			`{"balance":"0","equity":"0","availableMargin":"0","month":"2026-10","monthlyRealizedPnl":"0","monthlyPnlPercent":null,"periods":[],"unrealizedPnl":"0","unrealizedPnlPercent":null}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := getAccountSummary(t, accountSettings(func(context.Context) (appaccount.AccountSummary, error) {
				return testCase.summary, nil
			}))
			if recorder.Code != http.StatusOK {
				t.Errorf("Status = %d", recorder.Code)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", contentType)
			}
			if body := recorder.Body.String(); body != testCase.expected {
				t.Errorf("Body =\n%s\nerwartet\n%s", body, testCase.expected)
			}
		})
	}
}

func TestAccountSummary_NichtRegistriertOhneUseCase(t *testing.T) {
	recorder := getAccountSummary(t, accountSettings(nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404", recorder.Code)
	}
}

func TestAccountSummary_Fehlerabbildung(t *testing.T) {
	unavailable := func(reason position.UnavailableReason) error {
		return &position.OpenPositionsUnavailableError{Reason: reason, Cause: errors.New("geheim-ursache")}
	}
	testCases := []struct {
		name   string
		err    error
		status int
	}{
		{"not-configured", unavailable(position.ReasonNotConfigured), http.StatusServiceUnavailable},
		{"rate-limited", unavailable(position.ReasonRateLimited), http.StatusServiceUnavailable},
		{"unreachable", unavailable(position.ReasonUnreachable), http.StatusGatewayTimeout},
		{"upstream-error", unavailable(position.ReasonUpstreamError), http.StatusBadGateway},
		{"invalid-response", unavailable(position.ReasonInvalidResponse), http.StatusBadGateway},
		{"sonstiger Fehler", errors.New("db kaputt"), http.StatusInternalServerError},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := getAccountSummary(t, accountSettings(func(context.Context) (appaccount.AccountSummary, error) {
				return appaccount.AccountSummary{}, testCase.err
			}))
			if recorder.Code != testCase.status {
				t.Errorf("Status = %d, erwartet %d", recorder.Code, testCase.status)
			}
			if body := recorder.Body.String(); strings.Contains(body, "geheim-ursache") || strings.Contains(body, "db kaputt") {
				t.Errorf("Ursache im Body: %s", body)
			}
		})
	}
}

func TestAccountSummary_RateLimitSetztRetryAfter(t *testing.T) {
	recorder := getAccountSummary(t, accountSettings(func(context.Context) (appaccount.AccountSummary, error) {
		return appaccount.AccountSummary{}, &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}
	}))
	if recorder.Header().Get("Retry-After") != "30" {
		t.Errorf("Retry-After = %q", recorder.Header().Get("Retry-After"))
	}
}

func TestAccountSummary_ReichtRequestContextDurch(t *testing.T) {
	type key struct{}
	var received context.Context
	handler := NewHandler(accountSettings(func(ctx context.Context) (appaccount.AccountSummary, error) {
		received = ctx
		return appaccount.AccountSummary{}, nil
	}), discardLogger())
	request := newTestRequest(http.MethodGet, "/account/summary", allowedTestHost)
	request = request.WithContext(context.WithValue(request.Context(), key{}, "x"))
	serve(handler, request)
	if received == nil || received.Value(key{}) != "x" {
		t.Error("Request-Context nicht durchgereicht")
	}
}
