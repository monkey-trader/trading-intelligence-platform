package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	appprice "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"
)

// syncRecorder ist ein race-sauberer ResponseWriter mit Flush.
type syncRecorder struct {
	mutex   sync.Mutex
	header  http.Header
	body    strings.Builder
	code    int
	flushes int
}

func newSyncRecorder() *syncRecorder { return &syncRecorder{header: http.Header{}} }

func (recorder *syncRecorder) Header() http.Header { return recorder.header }
func (recorder *syncRecorder) WriteHeader(code int) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.code = code
}
func (recorder *syncRecorder) Write(data []byte) (int, error) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.Write(data)
}
func (recorder *syncRecorder) Flush() {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.flushes++
}
func (recorder *syncRecorder) bodyText() string {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	return recorder.body.String()
}

func eventually(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(message)
}

func text(value string) *string { return &value }

func sampleSnapshot() appprice.PriceSnapshot {
	return appprice.PriceSnapshot{
		FetchedAt: 1_791_800_000_000,
		Prices: []appprice.PositionPrice{{
			PositionID: "1", Symbol: "SUI-USDT", Price: "3.2105", Change24hPercent: "1.30",
			DistanceToBreakEvenPercent: text("2.5"),
		}},
	}
}

const sampleEventJSON = `{"prices":[{"positionId":"1","symbol":"SUI-USDT","price":"3.2105","change24hPercent":"1.30",` +
	`"distanceToBreakEvenPercent":"2.5","distanceToTp1Percent":null}],"fetchedAt":1791800000000,"stale":false}`

func startStream(t *testing.T, channel chan appprice.PriceSnapshot, heartbeat time.Duration) (*syncRecorder, context.CancelFunc, chan struct{}) {
	t.Helper()
	unsubscribed := make(chan struct{})
	handler := newPriceStreamHandler(func() (<-chan appprice.PriceSnapshot, func()) {
		return channel, func() { close(unsubscribed) }
	}, heartbeat, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	request := newTestRequest(http.MethodGet, "/prices/stream", allowedTestHost).WithContext(ctx)
	recorder := newSyncRecorder()
	done := make(chan struct{})
	go func() { handler.ServeHTTP(recorder, request); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return recorder, cancel, done
}

func TestPriceStreamSendsEventStreamHeadersAndPricesEvent(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot, 1)
	channel <- sampleSnapshot()
	recorder, _, _ := startStream(t, channel, time.Hour)

	expectedEvent := "event: prices\ndata: " + sampleEventJSON + "\n\n"
	eventually(t, func() bool { return recorder.bodyText() == expectedEvent }, "Event fehlt oder falsch: "+recorder.bodyText())

	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if got := recorder.header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("Content-Type = %q", got)
	}
	if got := recorder.header.Get("Cache-Control"); !strings.Contains(got, "no-cache") {
		t.Errorf("Cache-Control = %q, erwartet no-cache", got)
	}
	if recorder.flushes == 0 {
		t.Errorf("Event wurde nicht geflusht")
	}
}

func TestPriceStreamSendsStaleFlagAndEmptyPricesAsArray(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot, 1)
	channel <- appprice.PriceSnapshot{Prices: []appprice.PositionPrice{}, FetchedAt: 5, Stale: true}
	recorder, _, _ := startStream(t, channel, time.Hour)
	expected := "event: prices\ndata: {\"prices\":[],\"fetchedAt\":5,\"stale\":true}\n\n"
	eventually(t, func() bool { return recorder.bodyText() == expected }, "Body: "+recorder.bodyText())
}

func TestPriceStreamSendsHeartbeatCommentWhenIdle(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot)
	recorder, _, _ := startStream(t, channel, 20*time.Millisecond)
	eventually(t, func() bool {
		return strings.Contains(recorder.bodyText(), ":") && strings.HasPrefix(recorder.bodyText(), ":")
	},
		"kein Heartbeat-Kommentar: "+recorder.bodyText())
	if strings.Contains(recorder.bodyText(), "event:") {
		t.Errorf("Heartbeat darf kein Event sein: %q", recorder.bodyText())
	}
}

func TestPriceStreamHeartbeatIntervalIsFifteenSeconds(t *testing.T) {
	if priceHeartbeatInterval != 15*time.Second {
		t.Errorf("priceHeartbeatInterval = %v, erwartet 15s", priceHeartbeatInterval)
	}
}

func TestPriceStreamEndsAndUnsubscribesWhenClientDisconnects(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot)
	unsubscribed := make(chan struct{})
	handler := newPriceStreamHandler(func() (<-chan appprice.PriceSnapshot, func()) {
		return channel, func() { close(unsubscribed) }
	}, time.Hour, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	request := newTestRequest(http.MethodGet, "/prices/stream", allowedTestHost).WithContext(ctx)
	done := make(chan struct{})
	go func() { handler.ServeHTTP(newSyncRecorder(), request); close(done) }()

	cancel()

	for name, signal := range map[string]chan struct{}{"Handler endet": done, "Abmeldung": unsubscribed} {
		select {
		case <-signal:
		case <-time.After(time.Second):
			t.Fatalf("%s nicht nach Abbruch des Requests", name)
		}
	}
}

func TestPriceStreamEndsWhenSubscriptionChannelCloses(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot)
	_, _, done := startStream(t, channel, time.Hour)
	close(channel)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Handler endet nicht bei geschlossenem Kanal")
	}
}

func TestRouterRegistersPriceStreamOnlyWithSubscription(t *testing.T) {
	without := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, "/prices/stream", allowedTestHost))
	if without.Code != http.StatusNotFound {
		t.Errorf("ohne SubscribePrices: Status = %d, erwartet 404", without.Code)
	}

	channel := make(chan appprice.PriceSnapshot, 1)
	channel <- sampleSnapshot()
	settings := testSettings
	settings.SubscribePrices = func() (<-chan appprice.PriceSnapshot, func()) { return channel, func() {} }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := newTestRequest(http.MethodGet, "/prices/stream", allowedTestHost).WithContext(ctx)
	recorder := newSyncRecorder()
	done := make(chan struct{})
	go func() { NewHandler(settings, discardLogger()).ServeHTTP(recorder, request); close(done) }()
	eventually(t, func() bool { return strings.Contains(recorder.bodyText(), "event: prices") }, "kein Event über den Router")
	cancel()
	<-done
}

func TestRouterRejectsPriceStreamForForeignHost(t *testing.T) {
	settings := testSettings
	settings.SubscribePrices = func() (<-chan appprice.PriceSnapshot, func()) {
		return make(chan appprice.PriceSnapshot), func() {}
	}
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/prices/stream", "evil.example"))
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Status = %d, erwartet 403 (Host-Guard)", recorder.Code)
	}
}

// Der Stream darf nicht am WriteTimeout des Servers (10 s) sterben: Der Handler muss die Frist aufheben.
func TestPriceStreamSurvivesServerWriteTimeout(t *testing.T) {
	channel := make(chan appprice.PriceSnapshot, 4)
	server := httptest.NewUnstartedServer(nil)
	settings := testSettings
	settings.AllowedHosts = []string{server.Listener.Addr().String()}
	settings.SubscribePrices = func() (<-chan appprice.PriceSnapshot, func()) { return channel, func() {} }
	configuredServer := NewServer(NewHandler(settings, discardLogger()), discardLogger())
	configuredServer.WriteTimeout = 150 * time.Millisecond
	server.Config = configuredServer
	server.Start()
	t.Cleanup(server.Close)

	channel <- sampleSnapshot()
	response, err := http.Get(server.URL + "/prices/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	reader := bufio.NewReader(response.Body)
	if line, _ := reader.ReadString('\n'); line != "event: prices\n" {
		t.Fatalf("erste Zeile = %q", line)
	}

	time.Sleep(500 * time.Millisecond) // deutlich über dem WriteTimeout
	channel <- appprice.PriceSnapshot{Prices: []appprice.PositionPrice{}, FetchedAt: 9}

	// Zeilen lesen, bis die Daten des zweiten Events kommen (Verbindungsabbruch beendet die Schleife mit Fehler).
	var rest string
	for range 12 {
		line, readErr := reader.ReadString('\n')
		rest += line
		if readErr != nil || strings.Contains(line, `"fetchedAt":9`) {
			break
		}
	}
	if !strings.Contains(rest, `"fetchedAt":9`) {
		t.Errorf("zweites Event kam nicht an (Stream vom WriteTimeout gekappt?): %q", rest)
	}
}
