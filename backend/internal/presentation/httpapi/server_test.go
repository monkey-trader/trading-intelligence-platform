package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer_Einstellungen(t *testing.T) {
	handler := NewHandler(testSettings, discardLogger())
	server := NewServer(handler, discardLogger())

	if server.Handler == nil {
		t.Errorf("Handler fehlt")
	}
	durationChecks := []struct {
		name     string
		actual   time.Duration
		expected time.Duration
	}{
		{"ReadHeaderTimeout", server.ReadHeaderTimeout, 5 * time.Second},
		{"ReadTimeout", server.ReadTimeout, 10 * time.Second},
		{"WriteTimeout", server.WriteTimeout, 10 * time.Second},
		{"IdleTimeout", server.IdleTimeout, 30 * time.Second},
	}
	for _, check := range durationChecks {
		if check.actual != check.expected {
			t.Errorf("%s = %v, erwartet %v", check.name, check.actual, check.expected)
		}
	}
	if server.MaxHeaderBytes != 64<<10 {
		t.Errorf("MaxHeaderBytes = %d, erwartet %d", server.MaxHeaderBytes, 64<<10)
	}
	if server.ErrorLog == nil {
		t.Errorf("ErrorLog fehlt (slog.NewLogLogger)")
	}
	if !server.DisableGeneralOptionsHandler {
		t.Errorf("DisableGeneralOptionsHandler muss true sein, sonst beantwortet Go OPTIONS * selbst mit 200")
	}
	if server.ConnContext == nil {
		t.Fatalf("ConnContext fehlt")
	}
}

func TestNewServer_ConnContextReichtFlagDesWrappersWeiter(t *testing.T) {
	server := NewServer(NewHandler(testSettings, discardLogger()), discardLogger())
	if server.ConnContext == nil {
		t.Fatalf("ConnContext fehlt")
	}
	wrappedConn := acceptWrapped(t, newFakeConn())
	malformedTarget := malformedTargetFlagOf(wrappedConn)
	if malformedTarget == nil {
		t.Fatalf("malformedTargetFlagOf liefert nil für eine gewrappte Verbindung")
	}

	connectionContext := server.ConnContext(context.Background(), wrappedConn)
	malformedTarget.Store(true)
	request := newTestRequest(http.MethodGet, "/", allowedTestHost).WithContext(connectionContext)
	recorder := serve(server.Handler, request)

	assertProblem(t, recorder, http.StatusBadRequest, badRequestBody)
}
