package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	appchart "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Vertrag: GET /positions/{positionId}/chart (getPositionChart) in docs/openapi.yaml.
// Neue Settings-Eigenschaft (optional, nil = Route nicht registriert):
//   GetPositionChart func(ctx context.Context, query appchart.Query) (appchart.PositionChart, error)

const chartPositionID = "9100000000000000001"

const expectedChartBody = `{"positionId":"9100000000000000001","symbol":"SUI-USDT","side":"LONG","interval":"15m",` +
	`"candles":[{"time":1000,"open":"1","high":"2","low":"0.5","close":"1.5","volume":"10"}],` +
	`"entry":{"price":"0.9600","time":1791500000000},` +
	`"stopLoss":[{"orderId":"9200000000000000001","triggerPrice":"0.8875","quantity":"500","distanceRatio":"-0.07552083"}],` +
	`"takeProfit":[{"orderId":"9200000000000000002","triggerPrice":"1.2000","quantity":"250","distanceRatio":"0.25000000"},` +
	`{"orderId":"9200000000000000003","triggerPrice":"1.3500","quantity":"250","distanceRatio":"0.40625000"}],` +
	`"markers":[{"time":1791500000000,"kind":"ENTRY","price":"0.9600"}],"fetchedAt":1791800000000}`

func sampleChart() appchart.PositionChart {
	order := func(orderID, trigger, quantity, ratio string) appchart.ProtectiveOrderWithDistance {
		return appchart.ProtectiveOrderWithDistance{
			ProtectiveOrder: chart.ProtectiveOrder{OrderID: orderID, TriggerPrice: trigger, Quantity: quantity},
			DistanceRatio:   ratio,
		}
	}
	return appchart.PositionChart{
		PositionID: chartPositionID, Symbol: "SUI-USDT", Side: position.Long, Interval: chart.Interval15m,
		Candles: []chart.Candle{{Symbol: "SUI-USDT", Interval: chart.Interval15m, OpenTime: 1000,
			Open: "1", High: "2", Low: "0.5", Close: "1.5", Volume: "10"}},
		Entry:    appchart.EntryLevel{Price: "0.9600", Time: 1791500000000},
		StopLoss: []appchart.ProtectiveOrderWithDistance{order("9200000000000000001", "0.8875", "500", "-0.07552083")},
		TakeProfit: []appchart.ProtectiveOrderWithDistance{
			order("9200000000000000002", "1.2000", "250", "0.25000000"),
			order("9200000000000000003", "1.3500", "250", "0.40625000"),
		},
		Markers:   []appchart.Marker{{Time: 1791500000000, Kind: appchart.MarkerEntry, Price: "0.9600"}},
		FetchedAt: 1791800000000,
	}
}

type chartCall struct {
	query     appchart.Query
	callCount int
}

func chartSettings(call *chartCall, result appchart.PositionChart, err error) Settings {
	settings := settingsWith(notConfiguredOpenPositions)
	settings.GetPositionChart = func(_ context.Context, query appchart.Query) (appchart.PositionChart, error) {
		call.callCount++
		call.query = query
		return result, err
	}
	return settings
}

func getChart(settings Settings, target string) *chartResponseView {
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, target, allowedTestHost))
	return &chartResponseView{status: recorder.Code, header: recorder.Header(), body: recorder.Body.String()}
}

type chartResponseView struct {
	status int
	header http.Header
	body   string
}

func TestPositionChart_ErfolgExakteHuelle(t *testing.T) {
	call := &chartCall{}

	response := getChart(chartSettings(call, sampleChart(), nil), "/positions/"+chartPositionID+"/chart")

	if response.status != http.StatusOK {
		t.Fatalf("Status = %d, Body = %s", response.status, response.body)
	}
	if contentType := response.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", contentType)
	}
	if response.body != expectedChartBody {
		t.Errorf("Body =\n%s\nerwartet\n%s", response.body, expectedChartBody)
	}
}

func TestPositionChart_LeereListenSindArraysNichtNull(t *testing.T) {
	empty := sampleChart()
	empty.Candles, empty.StopLoss, empty.TakeProfit = []chart.Candle{}, []appchart.ProtectiveOrderWithDistance{}, []appchart.ProtectiveOrderWithDistance{}

	response := getChart(chartSettings(&chartCall{}, empty, nil), "/positions/"+chartPositionID+"/chart")

	for _, fragment := range []string{`"candles":[]`, `"stopLoss":[]`, `"takeProfit":[]`} {
		if !strings.Contains(response.body, fragment) {
			t.Errorf("Body enthält %s nicht: %s", fragment, response.body)
		}
	}
}

func TestPositionChart_QueryParameterGueltig(t *testing.T) {
	testCases := []struct {
		name             string
		query            string
		expectedInterval chart.CandleInterval
		expectedLimit    int
	}{
		{"Defaults", "", chart.Interval15m, 300},
		{"nur Intervall", "?interval=1h", chart.Interval1h, 300},
		{"nur Limit", "?limit=50", chart.Interval15m, 50},
		{"alle Intervalle 5m", "?interval=5m&limit=1", chart.Interval5m, 1},
		{"4h und Obergrenze", "?interval=4h&limit=1000", chart.Interval4h, 1000},
		{"1d", "?interval=1d", chart.Interval1d, 300},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			call := &chartCall{}

			response := getChart(chartSettings(call, sampleChart(), nil), "/positions/"+chartPositionID+"/chart"+testCase.query)

			if response.status != http.StatusOK {
				t.Fatalf("Status = %d, Body = %s", response.status, response.body)
			}
			expected := appchart.Query{PositionID: chartPositionID, Interval: testCase.expectedInterval, Limit: testCase.expectedLimit}
			if call.query != expected {
				t.Errorf("Query = %+v, erwartet %+v", call.query, expected)
			}
		})
	}
}

func TestPositionChart_UngueltigeParameterErgeben400OhneUseCase(t *testing.T) {
	testCases := []struct{ name, target string }{
		{"Intervall unbekannt", "/positions/" + chartPositionID + "/chart?interval=2h"},
		{"Intervall 1m (BingX kennt es, API nicht)", "/positions/" + chartPositionID + "/chart?interval=1m"},
		{"Intervall Großschreibung", "/positions/" + chartPositionID + "/chart?interval=1H"},
		{"Intervall leer", "/positions/" + chartPositionID + "/chart?interval="},
		{"Limit 0", "/positions/" + chartPositionID + "/chart?limit=0"},
		{"Limit 1001", "/positions/" + chartPositionID + "/chart?limit=1001"},
		{"Limit negativ", "/positions/" + chartPositionID + "/chart?limit=-5"},
		{"Limit keine Zahl", "/positions/" + chartPositionID + "/chart?limit=abc"},
		{"Limit Dezimal", "/positions/" + chartPositionID + "/chart?limit=1.5"},
		{"Limit leer", "/positions/" + chartPositionID + "/chart?limit="},
		{"Intervall doppelt", "/positions/" + chartPositionID + "/chart?interval=5m&interval=1h"},
		{"positionId keine Ziffern", "/positions/abc/chart"},
		{"positionId zu lang", "/positions/123456789012345678901/chart"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			call := &chartCall{}

			response := getChart(chartSettings(call, sampleChart(), nil), testCase.target)

			assertChartProblem(t, response, http.StatusBadRequest, ProblemBadRequest)
			if call.callCount != 0 {
				t.Error("Use Case darf bei ungültiger Anfrage nicht laufen")
			}
		})
	}
}

func assertChartProblem(t *testing.T, response *chartResponseView, status int, problemType ProblemType) {
	t.Helper()
	if response.status != status {
		t.Errorf("Status = %d, erwartet %d (Body %s)", response.status, status, response.body)
	}
	if contentType := response.header.Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, erwartet %q", contentType, problemContentType)
	}
	if !strings.Contains(response.body, `"type":"`+string(problemType)+`"`) {
		t.Errorf("Body %s enthält nicht type %s", response.body, problemType)
	}
}

func TestPositionChart_DomainFehlerWerdenAufProblemeAbgebildet(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		expectedCode int
		expectedType ProblemType
	}{
		{"unbekannte Position", &chart.PositionNotFoundError{PositionID: chartPositionID}, 404, ProblemNotFound},
		{"unbekannte Position verpackt", errors.Join(errors.New("x"), &chart.PositionNotFoundError{PositionID: "1"}), 404, ProblemNotFound},
		{"Intervall im Use Case abgelehnt", &chart.UnsupportedIntervalError{Value: "2h"}, 400, ProblemBadRequest},
		{"Limit im Use Case abgelehnt", &chart.LimitOutOfRangeError{Value: 0}, 400, ProblemBadRequest},
		{"ohne Keys", &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}, 503, ProblemNotConfigured},
		{"Positionen BingX-Müll", &position.OpenPositionsUnavailableError{Reason: position.ReasonInvalidResponse}, 502, ProblemInvalidResponse},
		{"Orders Müll", &chart.DataUnavailableError{Reason: position.ReasonInvalidResponse}, 502, ProblemInvalidResponse},
		{"Orders nicht konfiguriert", &chart.DataUnavailableError{Reason: position.ReasonNotConfigured}, 503, ProblemNotConfigured},
		{"Kerzen 429", &chart.DataUnavailableError{Reason: position.ReasonRateLimited}, 503, ProblemRateLimited},
		{"Kerzen unerreichbar", &chart.DataUnavailableError{Reason: position.ReasonUnreachable}, 504, ProblemUnreachable},
		{"Kerzen code != 0", &chart.DataUnavailableError{Reason: position.ReasonUpstreamError}, 502, ProblemUpstreamError},
		{"Symbol bei BingX unbekannt", &chart.DataUnavailableError{Reason: position.ReasonUpstreamError, Cause: &chart.SymbolNotFoundError{Symbol: "X-USDT"}}, 502, ProblemUpstreamError},
		{"unbekannter Fehler", errors.New("db kaputt"), 500, ProblemInternal},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response := getChart(chartSettings(&chartCall{}, appchart.PositionChart{}, testCase.err), "/positions/"+chartPositionID+"/chart")

			assertChartProblem(t, response, testCase.expectedCode, testCase.expectedType)
			if strings.Contains(response.body, "db kaputt") {
				t.Error("interne Fehlertexte dürfen nicht im Body stehen")
			}
		})
	}
}

func TestPositionChart_RateLimitSetztRetryAfterWieBeiOffenenPositionen(t *testing.T) {
	err := &chart.DataUnavailableError{Reason: position.ReasonRateLimited}

	response := getChart(chartSettings(&chartCall{}, appchart.PositionChart{}, err), "/positions/"+chartPositionID+"/chart")

	if got := response.header.Get("Retry-After"); got != rateLimitRetryAfterSeconds {
		t.Errorf("Retry-After = %q, erwartet %q", got, rateLimitRetryAfterSeconds)
	}
}

func TestPositionChart_NurGetUndKeineKollisionMitOpen(t *testing.T) {
	call := &chartCall{}
	settings := chartSettings(call, sampleChart(), nil)

	post := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodPost, "/positions/"+chartPositionID+"/chart", allowedTestHost))
	if post.Code == http.StatusOK || call.callCount != 0 {
		t.Errorf("POST darf die Route nicht erreichen: %d", post.Code)
	}
	// "open" ist keine positionId: /positions/open bleibt die Liste, /positions/open/chart gibt 400.
	open := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))
	if open.Code != http.StatusServiceUnavailable {
		t.Errorf("/positions/open Status = %d, erwartet 503 not-configured aus dem Fake", open.Code)
	}
	response := getChart(settings, "/positions/open/chart")
	assertChartProblem(t, response, http.StatusBadRequest, ProblemBadRequest)
}

func TestPositionChart_FremderHostIst403(t *testing.T) {
	request := newTestRequest(http.MethodGet, "/positions/"+chartPositionID+"/chart", "evil.example")
	recorder := serve(NewHandler(chartSettings(&chartCall{}, sampleChart(), nil), discardLogger()), request)
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403", recorder.Code)
	}
}
