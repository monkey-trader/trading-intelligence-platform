package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Fälle portiert aus backend/src/presentation/position/http/openPositionsRoute.test.ts,
// toOpenPositionDto.test.ts und errorMapper.test.ts. Ersetzt den Phase-1-Stub-Test.

const openPositionsFetchedAt int64 = 1791800000000

// openPositionSchemaFields: required-Liste von OpenPosition in docs/openapi.yaml, in Spec-Reihenfolge.
var openPositionSchemaFields = []string{
	"positionId", "symbol", "side", "isolated", "positionAmount", "avgOpenPrice", "markPrice",
	"leverage", "initialMargin", "margin", "positionValue", "unrealizedProfit", "realisedProfit",
	"pnlRatio", "stopLoss", "takeProfit", "breakEven", "openTime", "updateTime",
}

func ratio(value string) *string { return &value }

var goldEntry = appposition.OpenPositionEntry{
	OpenPosition: position.OpenPosition{
		PositionID: "9100000000000000002", Symbol: "NCCOGOLD2USD-USDT", Side: position.Short, Isolated: true,
		PositionAmount: "0.25", AvgOpenPrice: "4150.00", MarkPrice: "4175.00", Leverage: 15,
		InitialMargin: "69.1667", Margin: "65.0000", PositionValue: "1043.75", UnrealizedProfit: "-6.2500",
		RealisedProfit: "-0.5187", PnlRatio: "-0.0904", LiquidationPrice: "4410.5",
		OpenTime: 1791700000000, UpdateTime: 1791710000000,
	},
}

// crossEntry ohne Liquidationspreis: Dezimal-Hebel.
var crossEntry = appposition.OpenPositionEntry{
	OpenPosition: position.OpenPosition{
		PositionID: "9100000000000000001", Symbol: "SUI-USDT", Side: position.Long, Isolated: false,
		PositionAmount: "500", AvgOpenPrice: "0.9600", MarkPrice: "1.0100", Leverage: 2.5,
		InitialMargin: "48.0000", Margin: "60.5000", PositionValue: "505", UnrealizedProfit: "25.0000",
		RealisedProfit: "-0.2400", PnlRatio: "0.5208", LiquidationPrice: "0",
		OpenTime: 1791500000000, UpdateTime: 1791600000000,
	},
}

const goldJSON = `{"positionId":"9100000000000000002","symbol":"NCCOGOLD2USD-USDT","side":"SHORT","isolated":true,` +
	`"positionAmount":"0.25","avgOpenPrice":"4150.00","markPrice":"4175.00","leverage":15,` +
	`"initialMargin":"69.1667","margin":"65.0000","positionValue":"1043.75","unrealizedProfit":"-6.2500",` +
	`"realisedProfit":"-0.5187","pnlRatio":"-0.0904",` +
	`"stopLoss":null,"takeProfit":null,"breakEven":null,"openTime":1791700000000,"updateTime":1791710000000}`

const crossJSON = `{"positionId":"9100000000000000001","symbol":"SUI-USDT","side":"LONG","isolated":false,` +
	`"positionAmount":"500","avgOpenPrice":"0.9600","markPrice":"1.0100","leverage":2.5,` +
	`"initialMargin":"48.0000","margin":"60.5000","positionValue":"505","unrealizedProfit":"25.0000",` +
	`"realisedProfit":"-0.2400","pnlRatio":"0.5208",` +
	`"stopLoss":null,"takeProfit":null,"breakEven":null,"openTime":1791500000000,"updateTime":1791600000000}`

func settingsWith(getOpenPositions func(context.Context) (appposition.OpenPositionsOverview, error)) Settings {
	return Settings{AllowedHosts: testSettings.AllowedHosts, GetOpenPositions: getOpenPositions}
}

func overviewOf(total string, entries ...appposition.OpenPositionEntry) func(context.Context) (appposition.OpenPositionsOverview, error) {
	return func(context.Context) (appposition.OpenPositionsOverview, error) {
		return appposition.OpenPositionsOverview{Positions: entries, TotalUnrealizedProfit: total, FetchedAt: openPositionsFetchedAt}, nil
	}
}

func failingWith(err error) func(context.Context) (appposition.OpenPositionsOverview, error) {
	return func(context.Context) (appposition.OpenPositionsOverview, error) {
		return appposition.OpenPositionsOverview{}, err
	}
}

func getOpenPositions(t *testing.T, settings Settings, logger *slog.Logger) (status int, header http.Header, body string) {
	t.Helper()
	recorder := serve(NewHandler(settings, logger), newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))
	return recorder.Code, recorder.Header(), recorder.Body.String()
}

func TestOpenPositions_Erfolg(t *testing.T) {
	testCases := []struct {
		name         string
		getPositions func(context.Context) (appposition.OpenPositionsOverview, error)
		expectedBody string
	}{
		{
			name:         "Hülle exakt laut Spec, Reihenfolge wie geliefert, Abstand String und null",
			getPositions: overviewOf("18.75", goldEntry, crossEntry),
			expectedBody: `{"positions":[` + goldJSON + `,` + crossJSON + `],"totalUnrealizedProfit":"18.75","fetchedAt":1791800000000}`,
		},
		{
			name:         "leere Liste als []",
			getPositions: overviewOf("0", []appposition.OpenPositionEntry{}...),
			expectedBody: `{"positions":[],"totalUnrealizedProfit":"0","fetchedAt":1791800000000}`,
		},
		{
			name: "Positions nil ergibt trotzdem []",
			getPositions: func(context.Context) (appposition.OpenPositionsOverview, error) {
				return appposition.OpenPositionsOverview{Positions: nil, TotalUnrealizedProfit: "0", FetchedAt: openPositionsFetchedAt}, nil
			},
			expectedBody: `{"positions":[],"totalUnrealizedProfit":"0","fetchedAt":1791800000000}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, header, body := getOpenPositions(t, settingsWith(testCase.getPositions), discardLogger())

			if status != http.StatusOK {
				t.Errorf("Status = %d, erwartet 200", status)
			}
			if contentType := header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", contentType)
			}
			if body != testCase.expectedBody {
				t.Errorf("Body =\n%s\nerwartet\n%s", body, testCase.expectedBody)
			}
			if retryAfter := header.Get("Retry-After"); retryAfter != "" {
				t.Errorf("Retry-After = %q, bei Erfolg nicht erlaubt", retryAfter)
			}
		})
	}
}

func TestOpenPositions_FeldnamenExaktWieSpec(t *testing.T) {
	_, _, body := getOpenPositions(t, settingsWith(overviewOf("25.0000", crossEntry)), discardLogger())

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("kein JSON: %q (%v)", body, err)
	}
	envelopeKeys := sortedKeys(envelope)
	if !slices.Equal(envelopeKeys, []string{"fetchedAt", "positions", "totalUnrealizedProfit"}) {
		t.Errorf("Hüllen-Felder = %v", envelopeKeys)
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(envelope["positions"], &entries); err != nil || len(entries) != 1 {
		t.Fatalf("positions = %s (%v)", envelope["positions"], err)
	}
	expectedFields := slices.Sorted(slices.Values(openPositionSchemaFields))
	if got := sortedKeys(entries[0]); !slices.Equal(got, expectedFields) {
		t.Errorf("Positions-Felder = %v, erwartet %v", got, expectedFields)
	}
}

func sortedKeys(object map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func TestOpenPositions_Fehlerabbildung(t *testing.T) {
	testCases := []struct {
		name               string
		err                error
		expectedStatus     int
		expectedBody       string
		expectedRetryAfter string
	}{
		{
			name:           "not-configured",
			err:            &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   notConfiguredBody,
		},
		{
			name:               "rate-limited mit Retry-After 30",
			err:                &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited, Cause: errors.New("geheime-ursache-4711")},
			expectedStatus:     http.StatusServiceUnavailable,
			expectedBody:       `{"type":"rate-limited","title":"BingX ist ausgelastet, bitte später erneut versuchen.","status":503}`,
			expectedRetryAfter: "30",
		},
		{
			name:           "unreachable",
			err:            &position.OpenPositionsUnavailableError{Reason: position.ReasonUnreachable, Cause: errors.New("geheime-ursache-4711")},
			expectedStatus: http.StatusGatewayTimeout,
			expectedBody:   `{"type":"unreachable","title":"BingX ist nicht erreichbar.","status":504}`,
		},
		{
			name:           "upstream-error",
			err:            &position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError, Cause: errors.New("geheime-ursache-4711")},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"type":"upstream-error","title":"BingX hat mit einem Fehler geantwortet.","status":502}`,
		},
		{
			name:           "invalid-response",
			err:            &position.OpenPositionsUnavailableError{Reason: position.ReasonInvalidResponse, Cause: errors.New("geheime-ursache-4711")},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"type":"invalid-response","title":"BingX hat eine ungültige Antwort geliefert.","status":502}`,
		},
		{
			name:               "gewrappter rate-limited über errors.As",
			err:                fmt.Errorf("Hülle: %w", &position.OpenPositionsUnavailableError{Reason: position.ReasonRateLimited}),
			expectedStatus:     http.StatusServiceUnavailable,
			expectedBody:       `{"type":"rate-limited","title":"BingX ist ausgelastet, bitte später erneut versuchen.","status":503}`,
			expectedRetryAfter: "30",
		},
		{
			name:           "unbekannter Grund ergibt internal",
			err:            &position.OpenPositionsUnavailableError{Reason: position.UnavailableReason("bad-request")},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   internalBody,
		},
		{
			name:           "unbekannter Fehler ergibt internal",
			err:            errors.New("geheime-ursache-4711"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   internalBody,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serve(NewHandler(settingsWith(failingWith(testCase.err)), discardLogger()),
				newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))

			assertProblem(t, recorder, testCase.expectedStatus, testCase.expectedBody)
			if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != testCase.expectedRetryAfter {
				t.Errorf("Retry-After = %q, erwartet %q", retryAfter, testCase.expectedRetryAfter)
			}
			if strings.Contains(recorder.Body.String(), "geheime-ursache-4711") {
				t.Errorf("Body enthält die Ursache: %q", recorder.Body.String())
			}
		})
	}
}

func TestOpenPositions_UrsacheNurImLog(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, nil))
	cause := errors.New("geheime-ursache-4711")

	_, _, body := getOpenPositions(t,
		settingsWith(failingWith(&position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError, Cause: cause})), logger)

	if strings.Contains(body, "geheime-ursache-4711") {
		t.Errorf("Body enthält die Ursache: %q", body)
	}
	if !strings.Contains(logOutput.String(), "geheime-ursache-4711") {
		t.Errorf("Log enthält die Ursache nicht: %q", logOutput.String())
	}
}

func TestOpenPositions_ReichtRequestContextDurch(t *testing.T) {
	type contextKey struct{}
	var receivedValue any
	settings := settingsWith(func(ctx context.Context) (appposition.OpenPositionsOverview, error) {
		receivedValue = ctx.Value(contextKey{})
		return appposition.OpenPositionsOverview{Positions: nil, TotalUnrealizedProfit: "0", FetchedAt: openPositionsFetchedAt}, nil
	})
	request := newTestRequest(http.MethodGet, "/positions/open", allowedTestHost)
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "anfrage-3"))

	recorder := serve(NewHandler(settings, discardLogger()), request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200", recorder.Code)
	}
	if receivedValue != "anfrage-3" {
		t.Errorf("Use Case bekam nicht den Request-Context (Wert %v)", receivedValue)
	}
}

func TestOpenPositions_UseCaseWirdBeiAbgelehntenRequestsNichtAufgerufen(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		host   string
	}{
		{"fremder Host", http.MethodGet, "evil.example:3000"},
		{"POST", http.MethodPost, allowedTestHost},
		{"DELETE", http.MethodDelete, allowedTestHost},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			callCount := 0
			settings := settingsWith(func(context.Context) (appposition.OpenPositionsOverview, error) {
				callCount++
				return appposition.OpenPositionsOverview{}, nil
			})

			recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(testCase.method, "/positions/open", testCase.host))

			if recorder.Code == http.StatusOK {
				t.Errorf("Status 200, erwartet Ablehnung")
			}
			if callCount != 0 {
				t.Errorf("Use Case %d-mal aufgerufen, erwartet 0", callCount)
			}
		})
	}
}

// logLines teilt die Ausgabe eines slog-TextHandlers in Zeilen.
func logLines(output string) []string {
	var lines []string
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestOpenPositions_ClientAbbruchWirdNichtAlsWarnGeloggt(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectInternal bool
	}{
		{"nacktes context.Canceled", context.Canceled, true},
		{"eingewickeltes context.Canceled", fmt.Errorf("abgebrochen: %w", context.Canceled), true},
		// Status/Problem bleiben beim bestehenden Mapping; nur das Log-Level zählt.
		{"als Ursache im Unavailable-Fehler", &position.OpenPositionsUnavailableError{Reason: position.ReasonUnreachable, Cause: context.Canceled}, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))

			status, _, body := getOpenPositions(t, settingsWith(failingWith(testCase.err)), logger)

			if testCase.expectInternal {
				if status != http.StatusInternalServerError {
					t.Errorf("Status = %d, erwartet 500", status)
				}
				if !strings.Contains(body, "internal") {
					t.Errorf("Body = %q, erwartet Problem internal", body)
				}
			}
			for _, line := range logLines(logOutput.String()) {
				if strings.Contains(line, "level=WARN") || strings.Contains(line, "level=ERROR") {
					t.Errorf("Logzeile mit Level > INFO bei Client-Abbruch: %q", line)
				}
			}
		})
	}
}

func TestOpenPositions_EchterFehlerBleibtWarn(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))

	getOpenPositions(t, settingsWith(failingWith(errors.New("kaputt"))), logger)

	if !strings.Contains(logOutput.String(), "level=WARN") {
		t.Errorf("Log = %q, erwartet WARN bei echtem Fehler", logOutput.String())
	}
}

func TestOpenPositions_LogKuerztLangenCauseGenerisch(t *testing.T) {
	const maxMessageLength = 200
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cause := errors.New(strings.Repeat("Z", 500))

	getOpenPositions(t, settingsWith(failingWith(
		&position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError, Cause: cause})), logger)

	logged := logOutput.String()
	longestRun, currentRun := 0, 0
	for _, character := range logged {
		if character == 'Z' {
			currentRun++
			longestRun = max(longestRun, currentRun)
		} else {
			currentRun = 0
		}
	}
	if longestRun == 0 {
		t.Fatalf("Meldung fehlt komplett im Log: %q", logged)
	}
	if longestRun > maxMessageLength {
		t.Errorf("geloggte Meldung hat %d Zeichen, erwartet <= %d", longestRun, maxMessageLength)
	}
	if !strings.Contains(logged, "…") {
		t.Errorf("Log enthält keinen Kürzungshinweis „…“: %q", logged)
	}
}

func TestOpenPositions_LogLaesstKurzenCauseUnveraendert(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cause := errors.New("the query range is more than ninety days")

	getOpenPositions(t, settingsWith(failingWith(
		&position.OpenPositionsUnavailableError{Reason: position.ReasonUpstreamError, Cause: cause})), logger)

	if !strings.Contains(logOutput.String(), "the query range is more than ninety days") {
		t.Errorf("kurze Meldung fehlt im Log: %q", logOutput.String())
	}
	if strings.Contains(logOutput.String(), "…") {
		t.Errorf("kurze Meldung darf keinen Kürzungshinweis tragen: %q", logOutput.String())
	}
}

func TestToOpenPositionsSnapshotDto_SchutzfelderWerdenAbgebildet(t *testing.T) {
	yes, no := true, false
	testCases := []struct {
		name      string
		stopLoss  *string
		take      *string
		breakEven *bool
		expected  string
	}{
		{"alle gesetzt, BE true", ratio("0.97"), ratio("1.20"), &yes, `"stopLoss":"0.97","takeProfit":"1.20","breakEven":true`},
		{"BE false", ratio("0.90"), nil, &no, `"stopLoss":"0.90","takeProfit":null,"breakEven":false`},
		{"Orders unbekannt: alle null", nil, nil, nil, `"stopLoss":null,"takeProfit":null,"breakEven":null`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			entry := goldEntry
			entry.StopLoss, entry.TakeProfit, entry.BreakEven = testCase.stopLoss, testCase.take, testCase.breakEven

			encoded, err := json.Marshal(toOpenPositionsSnapshotDto(appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{entry}, TotalUnrealizedProfit: "0"}))

			if err != nil || !strings.Contains(string(encoded), testCase.expected) {
				t.Errorf("JSON = %s (%v), erwartet Teilstring %s", encoded, err, testCase.expected)
			}
		})
	}
}

func TestToOpenPositionsSnapshotDto_Levels(t *testing.T) {
	t.Run("mit Levels: kind, rMultiple, price, percent", func(t *testing.T) {
		entry := goldEntry
		entry.Levels = []position.RLevel{
			{Multiple: 1, Price: "1.02", Percent: "6.25"},
			{Multiple: 2, Price: "1.08", Percent: "12.5"},
			{Multiple: 3, Price: "1.14", Percent: "18.75"},
		}
		encoded, err := json.Marshal(toOpenPositionsSnapshotDto(appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{entry}, TotalUnrealizedProfit: "0"}))
		for _, part := range []string{
			`"levels":[`,
			`{"kind":"ONE_R","rMultiple":1,"price":"1.02","percent":"6.25"}`,
			`{"kind":"TWO_R","rMultiple":2,"price":"1.08","percent":"12.5"}`,
			`{"kind":"THREE_R","rMultiple":3,"price":"1.14","percent":"18.75"}`,
		} {
			if err != nil || !strings.Contains(string(encoded), part) {
				t.Errorf("JSON = %s (%v), erwartet Teilstring %s", encoded, err, part)
			}
		}
	})
	t.Run("ohne Levels: Feld fehlt (omitempty)", func(t *testing.T) {
		entry := goldEntry
		entry.Levels = nil
		encoded, err := json.Marshal(toOpenPositionsSnapshotDto(appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{entry}, TotalUnrealizedProfit: "0"}))
		if err != nil || strings.Contains(string(encoded), `"levels"`) {
			t.Errorf("JSON = %s (%v)", encoded, err)
		}
	})
}

// Erwartet: OpenPositionEntry.InitialStopLossSource *string -> DTO-Feld "initialStopLossSource" (omitempty bei nil).
func TestToOpenPositionsSnapshotDto_InitialStopLossSource(t *testing.T) {
	manual := "MANUAL"
	withSource := appposition.OpenPositionEntry{InitialStopLossSource: &manual}
	body, err := json.Marshal(toOpenPositionsSnapshotDto(appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{withSource}}))
	if err != nil || !strings.Contains(string(body), `"initialStopLossSource":"MANUAL"`) {
		t.Errorf("Body = %s err=%v", body, err)
	}
	body, _ = json.Marshal(toOpenPositionsSnapshotDto(appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{{}}}))
	if strings.Contains(string(body), "initialStopLossSource") {
		t.Errorf("Feld muss bei nil fehlen: %s", body)
	}
}
