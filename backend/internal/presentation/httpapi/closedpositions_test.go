package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

func closedSettings(captured *position.ClosedPositionQuery, views []appposition.ClosedPositionView, err error) Settings {
	settings := settingsWith(overviewOf("0"))
	settings.GetClosedPositions = func(_ context.Context, query position.ClosedPositionQuery) ([]appposition.ClosedPositionView, error) {
		if captured != nil {
			*captured = query
		}
		return views, err
	}
	return settings
}

func getClosed(t *testing.T, settings Settings, target string) (int, http.Header, string) {
	t.Helper()
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, target, allowedTestHost))
	return recorder.Code, recorder.Header(), recorder.Body.String()
}

func strPtr(value string) *string { return &value }

const closedJSON = `{"positionId":"9100000000000000003","symbol":"SUI-USDT","side":"LONG","openTime":1791500000000,` +
	`"closeTime":1791600000000,"avgOpenPrice":"100","avgClosePrice":"130","positionAmount":"2","leverage":7.5,` +
	`"realisedProfit":"61","netProfit":"60","commission":"-0.5","totalFunding":"0.1",` +
	`"rResult":"3","breakEvenPrice":"120","tp1Price":"130"}`

const closedNullJSON = `{"positionId":"9100000000000000004","symbol":"SUI-USDT","side":"SHORT","openTime":1791500000000,` +
	`"closeTime":1791500500000,"avgOpenPrice":"100","avgClosePrice":"105","positionAmount":"2","leverage":3,` +
	`"realisedProfit":"-10","netProfit":"-11","commission":"-0.5","totalFunding":"0",` +
	`"rResult":null,"breakEvenPrice":null,"tp1Price":null}`

func closedView(id, side string, closeTime int64, avgClose, netProfit string, leverage float64, withR bool) appposition.ClosedPositionView {
	view := appposition.ClosedPositionView{ClosedPosition: position.ClosedPosition{
		PositionID: id, Symbol: "SUI-USDT", Side: position.PositionSide(side), OpenTime: 1791500000000, CloseTime: closeTime,
		AvgOpenPrice: "100", AvgClosePrice: avgClose, PositionAmount: "2", Leverage: leverage,
		RealisedProfit: map[bool]string{true: "61", false: "-10"}[withR], NetProfit: netProfit,
		Commission: "-0.5", TotalFunding: map[bool]string{true: "0.1", false: "0"}[withR], FullyClosed: true}}
	if withR {
		view.RResult, view.BreakEvenPrice, view.TP1Price = strPtr("3"), strPtr("120"), strPtr("130")
	}
	return view
}

func TestClosedPositions_ErfolgLiefertHuelleMitFeldernLautSpec(t *testing.T) {
	views := []appposition.ClosedPositionView{
		closedView("9100000000000000003", "LONG", 1791600000000, "130", "60", 7.5, true),
		closedView("9100000000000000004", "SHORT", 1791500500000, "105", "-11", 3, false),
	}
	status, header, body := getClosed(t, closedSettings(nil, views, nil), "/positions/closed")
	if status != http.StatusOK || header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("status=%d content-type=%q", status, header.Get("Content-Type"))
	}
	want := `{"positions":[` + closedJSON + `,` + closedNullJSON + `]}`
	if body != want {
		t.Errorf("Body =\n%s\nerwartet\n%s", body, want)
	}
}

func TestClosedPositions_LeereListeIstArray(t *testing.T) {
	for _, views := range [][]appposition.ClosedPositionView{nil, {}} {
		_, _, body := getClosed(t, closedSettings(nil, views, nil), "/positions/closed")
		if body != `{"positions":[]}` {
			t.Errorf("Body = %s", body)
		}
	}
}

func TestClosedPositions_ReichtParameterAlsQueryDurch(t *testing.T) {
	var captured position.ClosedPositionQuery
	status, _, _ := getClosed(t, closedSettings(&captured, nil, nil), "/positions/closed?from=1000&to=2000&symbol=SUI-USDT&limit=500")
	if status != http.StatusOK || captured.From() != 1000 || captured.To() != 2000 || captured.Symbol() != "SUI-USDT" || captured.Limit() != 500 {
		t.Errorf("status=%d from=%d to=%d symbol=%q limit=%d", status, captured.From(), captured.To(), captured.Symbol(), captured.Limit())
	}
}

func TestClosedPositions_OhneParameterGiltDefaultLimit100(t *testing.T) {
	var captured position.ClosedPositionQuery
	getClosed(t, closedSettings(&captured, nil, nil), "/positions/closed")
	if captured.Limit() != 100 {
		t.Errorf("limit = %d", captured.Limit())
	}
}

func TestClosedPositions_UngueltigeParameterErgeben400BadRequest(t *testing.T) {
	targets := []string{
		"limit=0", "limit=501", "limit=-1", "limit=abc", "limit=",
		"from=abc", "to=abc", "from=-1", "to=-1",
		"from=2000&to=1000",
	}
	for _, query := range targets {
		t.Run(query, func(t *testing.T) {
			status, header, body := getClosed(t, closedSettings(nil, nil, nil), "/positions/closed?"+query)
			if status != http.StatusBadRequest || !strings.Contains(body, `"type":"bad-request"`) || !strings.Contains(body, `"status":400`) {
				t.Errorf("status=%d body=%s", status, body)
			}
			if !strings.HasPrefix(header.Get("Content-Type"), "application/problem+json") {
				t.Errorf("Content-Type = %q", header.Get("Content-Type"))
			}
		})
	}
}

func TestClosedPositions_GrenzwerteSindGueltig(t *testing.T) {
	for _, query := range []string{"limit=1", "limit=500", "from=1000&to=1000"} {
		if status, _, _ := getClosed(t, closedSettings(nil, nil, nil), "/positions/closed?"+query); status != http.StatusOK {
			t.Errorf("%s: Status %d", query, status)
		}
	}
}

func TestClosedPositions_UseCaseFehlerErgibt500OhneDetails(t *testing.T) {
	status, _, body := getClosed(t, closedSettings(nil, nil, errors.New("geheim: db kaputt")), "/positions/closed")
	if status != http.StatusInternalServerError || !strings.Contains(body, `"type":"internal"`) || strings.Contains(body, "geheim") {
		t.Errorf("status=%d body=%s", status, body)
	}
}

func TestClosedPositions_NichtKonfiguriertErgibt503(t *testing.T) {
	unavailable := &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}
	status, _, body := getClosed(t, closedSettings(nil, nil, unavailable), "/positions/closed")
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "not-configured") {
		t.Errorf("Status = %d, Body = %s", status, body)
	}
}

func TestClosedPositions_OhneUseCaseIstRouteNichtRegistriert(t *testing.T) {
	settings := settingsWith(overviewOf("0"))
	if status, _, _ := getClosed(t, settings, "/positions/closed"); status != http.StatusNotFound {
		t.Errorf("Status = %d", status)
	}
}
