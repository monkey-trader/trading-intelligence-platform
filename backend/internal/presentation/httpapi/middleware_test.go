package httpapi

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// panicValueOf ruft den Handler direkt auf und liefert einen erneut geworfenen Panic-Wert.
func panicValueOf(handler http.Handler, request *http.Request) (recorder *httptest.ResponseRecorder, panicValue any) {
	recorder = httptest.NewRecorder()
	defer func() { panicValue = recover() }()
	handler.ServeHTTP(recorder, request)
	return recorder, nil
}

func TestRecover_ErrAbortHandlerWirdWeitergeworfen(t *testing.T) {
	testCases := []struct {
		name       string
		panicValue error
	}{
		{name: "http.ErrAbortHandler direkt", panicValue: http.ErrAbortHandler},
		{name: "http.ErrAbortHandler gewrappt", panicValue: fmt.Errorf("Abbruch: %w", http.ErrAbortHandler)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newHandlerWithRoutes(testSettings, discardLogger(), map[string]http.HandlerFunc{
				"GET /abort": func(http.ResponseWriter, *http.Request) { panic(testCase.panicValue) },
			})

			recorder, rethrown := panicValueOf(handler, newTestRequest(http.MethodGet, "/abort", allowedTestHost))

			rethrownError, isError := rethrown.(error)
			if !isError || !errors.Is(rethrownError, http.ErrAbortHandler) {
				t.Fatalf("erneut geworfen = %v, erwartet http.ErrAbortHandler", rethrown)
			}
			if recorder.Body.Len() != 0 {
				t.Errorf("Body = %q, erwartet keinen Problem-Body", recorder.Body.String())
			}
		})
	}
}

func TestRecover_ErrAbortHandlerBrichtVerbindungOhneAntwortAb(t *testing.T) {
	handler := newHandlerWithRoutes(testSettings, discardLogger(), map[string]http.HandlerFunc{
		"GET /abort": func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) },
	})
	testServer := httptest.NewUnstartedServer(handler)
	testServer.Start()
	t.Cleanup(testServer.Close)
	address := testServer.Listener.Addr().String()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}
	// Host 127.0.0.1:3000 ist in testSettings erlaubt; der Host-Guard prüft den Header, nicht die Adresse.
	if _, err := io.WriteString(conn, "GET /abort HTTP/1.1\r\nHost: "+allowedTestHost+"\r\n\r\n"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err == nil {
		_ = response.Body.Close()
		t.Fatalf("Antwort mit Status %d erhalten, erwartet Abbruch ohne Antwort", response.StatusCode)
	}
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		t.Fatalf("Verbindung blieb offen; erwartet Abbruch")
	}
}
