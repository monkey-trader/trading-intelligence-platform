package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appaccount "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/account"
	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Vertrag: PUT/DELETE /positions/{positionId}/swing-long (204, idempotent, nur Loopback) in docs/openapi.yaml.
// Erwartet: Settings.SetSwingLong / ClearSwingLong func(ctx, positionID string) error (beide nötig, nicht bei AllowRemote);
// Felder swingLong (Open/Closed), swingLongUnrealizedProfit (Snapshot), swingLongUnrealizedPnl/swingLongRealizedPnl (Konto).

const swingPath = "/positions/" + chartPositionID + "/swing-long"

type swingCalls struct {
	setCount, clearCount int
	positionID           string
}

func swingSettings(calls *swingCalls, setErr, clearErr error) Settings {
	settings := settingsWith(overviewOf("0"))
	settings.SetSwingLong = func(_ context.Context, positionID string) error {
		calls.setCount++
		calls.positionID = positionID
		return setErr
	}
	settings.ClearSwingLong = func(_ context.Context, positionID string) error {
		calls.clearCount++
		calls.positionID = positionID
		return clearErr
	}
	return settings
}

func sendSwing(settings Settings, method, target, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	request.Host = allowedTestHost
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	return serve(NewHandler(settings, discardLogger()), request)
}

func TestSwingLong_PutUndDeleteGeben204OhneBody(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			calls := &swingCalls{}
			recorder := sendSwing(swingSettings(calls, nil, nil), method, swingPath, "")
			if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
				t.Fatalf("Status %d Body %q", recorder.Code, recorder.Body.String())
			}
			if calls.positionID != chartPositionID || calls.setCount+calls.clearCount != 1 ||
				(method == http.MethodPut) != (calls.setCount == 1) {
				t.Errorf("Aufruf: %+v", calls)
			}
		})
	}
}

func TestSwingLong_ZweimaligesPutBleibt204(t *testing.T) {
	settings := swingSettings(&swingCalls{}, nil, nil)
	for range 2 {
		if recorder := sendSwing(settings, http.MethodPut, swingPath, ""); recorder.Code != http.StatusNoContent {
			t.Fatalf("Status %d", recorder.Code)
		}
	}
}

func TestSwingLong_UngueltigeIdErgibt400OhneUseCase(t *testing.T) {
	for _, target := range []string{"/positions/abc/swing-long", "/positions/123456789012345678901/swing-long"} {
		for _, method := range []string{http.MethodPut, http.MethodDelete} {
			t.Run(method+" "+target, func(t *testing.T) {
				calls := &swingCalls{}
				recorder := sendSwing(swingSettings(calls, nil, nil), method, target, "")
				if recorder.Code != http.StatusBadRequest || problemTypeOf(t, recorder) != "bad-request" {
					t.Fatalf("Status %d Body %s", recorder.Code, recorder.Body.String())
				}
				if calls.setCount+calls.clearCount != 0 {
					t.Error("Use Case darf nicht laufen")
				}
			})
		}
	}
}

func TestSwingLong_FremderOriginErgibt403(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		calls := &swingCalls{}
		recorder := sendSwing(swingSettings(calls, nil, nil), method, swingPath, "http://evil.example")
		if recorder.Code != http.StatusForbidden || problemTypeOf(t, recorder) != "forbidden-origin" || calls.setCount+calls.clearCount != 0 {
			t.Errorf("%s: Status %d Body %s", method, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSwingLong_UseCaseFehlerErgibt500(t *testing.T) {
	boom := errors.New("db kaputt")
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		recorder := sendSwing(swingSettings(&swingCalls{}, boom, boom), method, swingPath, "")
		if recorder.Code != http.StatusInternalServerError || problemTypeOf(t, recorder) != "internal" {
			t.Errorf("%s: Status %d Body %s", method, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSwingLong_NichtRegistriertBeiAllowRemoteOderOhneUseCase(t *testing.T) {
	remote := swingSettings(&swingCalls{}, nil, nil)
	remote.AllowRemote = true
	onlySet := swingSettings(&swingCalls{}, nil, nil)
	onlySet.ClearSwingLong = nil
	for name, settings := range map[string]Settings{"AllowRemote": remote, "nur Set": onlySet, "keine": settingsWith(overviewOf("0"))} {
		t.Run(name, func(t *testing.T) {
			if recorder := sendSwing(settings, http.MethodPut, swingPath, ""); recorder.Code != http.StatusNotFound {
				t.Errorf("Status %d, erwartet 404", recorder.Code)
			}
		})
	}
}

func decodeObject(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("kein JSON-Objekt: %s", body)
	}
	return object
}

func TestSwingLong_OffenePositionenLiefernFlagUndSummen(t *testing.T) {
	flagged := appposition.OpenPositionEntry{OpenPosition: position.OpenPosition{PositionID: "9100000000000000001", Symbol: "SUI-USDT", Side: position.Long}, SwingLong: true}
	normal := appposition.OpenPositionEntry{OpenPosition: position.OpenPosition{PositionID: "9100000000000000002", Symbol: "SUI-USDT", Side: position.Long}}
	settings := settingsWith(func(context.Context) (appposition.OpenPositionsOverview, error) {
		return appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{flagged, normal},
			TotalUnrealizedProfit: "4", SwingLongUnrealizedProfit: "-100.25"}, nil
	})
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/open", allowedTestHost))
	envelope := decodeObject(t, recorder.Body.Bytes())
	if string(envelope["totalUnrealizedProfit"]) != `"4"` || string(envelope["swingLongUnrealizedProfit"]) != `"-100.25"` {
		t.Errorf("Summen: %s", recorder.Body.String())
	}
	var positions []map[string]json.RawMessage
	if err := json.Unmarshal(envelope["positions"], &positions); err != nil || len(positions) != 2 {
		t.Fatalf("positions: %s", envelope["positions"])
	}
	if string(positions[0]["swingLong"]) != "true" || string(positions[1]["swingLong"]) != "false" {
		t.Errorf("swingLong: %s / %s", positions[0]["swingLong"], positions[1]["swingLong"])
	}
}

func TestSwingLong_GeschlosseneLiefernFlag(t *testing.T) {
	view := closedView("9100000000000000003", "LONG", 1791600000000, "130", "60", 7.5, true)
	view.SwingLong = true
	status, _, body := getClosed(t, closedSettings(nil, []appposition.ClosedPositionView{view}, nil), "/positions/closed")
	if status != http.StatusOK || !strings.Contains(body, `"swingLong":true`) {
		t.Errorf("Status %d Body %s", status, body)
	}
	plain := closedView("9100000000000000004", "SHORT", 1791500500000, "105", "-11", 3, false)
	_, _, body = getClosed(t, closedSettings(nil, []appposition.ClosedPositionView{plain}, nil), "/positions/closed")
	if !strings.Contains(body, `"swingLong":false`) {
		t.Errorf("swingLong:false fehlt: %s", body)
	}
}

func TestSwingLong_KontoLiefertGetrennteFelder(t *testing.T) {
	summary := appaccount.AccountSummary{Balance: "1000", Equity: "1000", AvailableMargin: "1", Month: "2026-10",
		MonthlyRealizedPnl: "10", Periods: []appaccount.PeriodPnl{}, UnrealizedPnl: "50",
		SwingLongUnrealizedPnl: "-200.5", SwingLongRealizedPnl: "-45"}
	recorder := getAccountSummary(t, accountSettings(func(context.Context) (appaccount.AccountSummary, error) { return summary, nil }))
	object := decodeObject(t, recorder.Body.Bytes())
	if string(object["swingLongUnrealizedPnl"]) != `"-200.5"` || string(object["swingLongRealizedPnl"]) != `"-45"` {
		t.Errorf("Body: %s", recorder.Body.String())
	}
}

func TestSwingLong_NichtKonfiguriertErgibt503(t *testing.T) {
	notConfigured := &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			recorder := sendSwing(swingSettings(&swingCalls{}, notConfigured, notConfigured), method, swingPath, "")
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("Status %d, erwartet 503", recorder.Code)
			}
		})
	}
}
