package bingx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

// Fälle portiert aus backend/src/infrastructure/bingx/BingXClient.test.ts, ergänzt um
// Go-spezifische Punkte aus dem Zuschnitt (Redirect, Body-Limit, Timeouts, ctx).

const (
	testAPIKey        = "TEST-API-KEY-1234567890"
	testAPISecret     = "TEST-API-SECRET-0987654321"
	openPositionsPath = "/openApi/swap/v2/user/positions"
	historyPath       = "/openApi/swap/v1/trade/positionHistory"
	firstNowMillis    = 1_780_000_001_000
	redirectedPath    = "/redirected"
)

// fakeReply ist eine vorbereitete Antwort des Fake-Servers.
type fakeReply struct {
	status       int
	body         string
	location     string
	networkError bool // Verbindung ohne Antwort schließen
	hang         bool // erst antworten, wenn der Client abbricht
}

type recordedRequest struct {
	method   string
	path     string
	rawQuery string
	header   http.Header
}

// fakeBingX spielt vorbereitete Antworten der Reihe nach ab; die letzte wiederholt sich.
type fakeBingX struct {
	server *httptest.Server

	mutex           sync.Mutex
	replies         []fakeReply
	requests        []recordedRequest
	redirectedCount int
	release         chan struct{}
}

func newFakeBingX(t *testing.T, replies ...fakeReply) *fakeBingX {
	t.Helper()
	fake := &fakeBingX{replies: replies, release: make(chan struct{})}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(fake.server.Close)
	// Läuft vor server.Close (LIFO), damit hängende Handler freikommen.
	t.Cleanup(func() { close(fake.release) })
	return fake
}

func (fake *fakeBingX) handle(w http.ResponseWriter, r *http.Request) {
	fake.mutex.Lock()
	if r.URL.Path == redirectedPath {
		fake.redirectedCount++
		fake.mutex.Unlock()
		writeBody(w, http.StatusOK, okBody(`["umgeleitet"]`))
		return
	}
	fake.requests = append(fake.requests, recordedRequest{
		method: r.Method, path: r.URL.Path, rawQuery: r.URL.RawQuery, header: r.Header.Clone(),
	})
	reply := fake.replies[min(len(fake.requests)-1, len(fake.replies)-1)]
	fake.mutex.Unlock()

	switch {
	case reply.networkError:
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			panic("Hijacker nicht verfügbar")
		}
		connection, _, err := hijacker.Hijack()
		if err == nil {
			_ = connection.Close()
		}
	case reply.hang:
		select {
		case <-r.Context().Done():
		case <-fake.release:
		}
	default:
		if reply.location != "" {
			w.Header().Set("Location", reply.location)
		}
		writeBody(w, reply.status, reply.body)
	}
}

func writeBody(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (fake *fakeBingX) recorded() []recordedRequest {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return slices.Clone(fake.requests)
}

func (fake *fakeBingX) redirectedHits() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.redirectedCount
}

func okBody(data string) string { return `{"code":0,"msg":"","data":` + data + `}` }

func okReply(data string) fakeReply { return fakeReply{status: http.StatusOK, body: okBody(data)} }

func apiCodeReply(code int) fakeReply {
	return fakeReply{status: http.StatusOK, body: `{"code":` + strconv.Itoa(code) + `,"msg":"BingX meldet ` + strconv.Itoa(code) + `","data":{}}`}
}

var rateLimitReply = fakeReply{status: http.StatusTooManyRequests, body: `{"code":100410,"msg":"rate limited"}`}

// testHarness hält die injizierten Fakes für Sleep und Now.
type testHarness struct {
	mutex      sync.Mutex
	sleepCalls []time.Duration
	nowMillis  int64
}

func (harness *testHarness) sleep(ctx context.Context, duration time.Duration) error {
	harness.mutex.Lock()
	harness.sleepCalls = append(harness.sleepCalls, duration)
	harness.mutex.Unlock()
	return ctx.Err()
}

// now liefert bei jedem Aufruf eine um 1 s spätere Zeit, damit jeder Versuch einen neuen timestamp bekommt.
func (harness *testHarness) now() time.Time {
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	harness.nowMillis += 1_000
	return time.UnixMilli(harness.nowMillis)
}

func (harness *testHarness) sleeps() []time.Duration {
	harness.mutex.Lock()
	defer harness.mutex.Unlock()
	return slices.Clone(harness.sleepCalls)
}

func newTestClient(fake *fakeBingX, modify func(*bingx.ClientOptions)) (*bingx.Client, *testHarness) {
	harness := &testHarness{nowMillis: firstNowMillis - 1_000}
	options := bingx.ClientOptions{
		APIKey:         testAPIKey,
		APISecret:      testAPISecret,
		BaseURL:        fake.server.URL,
		HTTPClient:     fake.server.Client(),
		Sleep:          harness.sleep,
		Now:            harness.now,
		MaxRetries:     3,
		RequestPause:   0,
		InitialBackoff: time.Second,
		RequestTimeout: 2 * time.Second,
	}
	if modify != nil {
		modify(&options)
	}
	return bingx.NewClient(options), harness
}

// splitSignedQuery trennt den signierten Teil von der Signatur am Ende.
func splitSignedQuery(t *testing.T, rawQuery string) (signedQuery, signature string) {
	t.Helper()
	index := strings.LastIndex(rawQuery, "&signature=")
	if index < 0 {
		t.Fatalf("keine Signatur am Ende des Query-Strings: %q", rawQuery)
	}
	return rawQuery[:index], rawQuery[index+len("&signature="):]
}

func assertValidSignature(t *testing.T, request recordedRequest) {
	t.Helper()
	signedQuery, signature := splitSignedQuery(t, request.rawQuery)
	if expected := referenceSignature(signedQuery, testAPISecret); signature != expected {
		t.Errorf("Signatur = %q, erwartet %q für %q", signature, expected, signedQuery)
	}
}

// assertNoSecrets prüft die ganze Fehlerkette: kein Key, kein Secret, keine signierte URL.
func assertNoSecrets(t *testing.T, err error, fake *fakeBingX) {
	t.Helper()
	forbidden := []string{testAPIKey, testAPISecret, "signature=", "timestamp=", fake.server.URL,
		strings.TrimPrefix(fake.server.URL, "http://")}
	for current := err; current != nil; current = errors.Unwrap(current) {
		text := current.Error()
		for _, part := range forbidden {
			if strings.Contains(text, part) {
				t.Errorf("Fehlertext %q (%T) enthält %q", text, current, part)
			}
		}
	}
}

func assertRequestCount(t *testing.T, fake *fakeBingX, expected int) {
	t.Helper()
	if got := len(fake.recorded()); got != expected {
		t.Errorf("%d Requests, erwartet %d", got, expected)
	}
}

func compactJSON(t *testing.T, raw []byte) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		t.Fatalf("kein gültiges JSON: %q (%v)", raw, err)
	}
	return buffer.String()
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	// Pfad relativ zum Paketordner internal/infrastructure/bingx.
	content, err := os.ReadFile("../../../../fixtures/bingx/" + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return string(content)
}

func TestClient_HappyPath(t *testing.T) {
	fake := newFakeBingX(t, okReply(`[{"positionId":"9100000000000000001"}]`))
	client, harness := newTestClient(fake, nil)

	data, err := client.Get(context.Background(), historyPath, []bingx.QueryParam{
		{Name: "symbol", Value: "ONDO-USDT"},
		{Name: "startTs", Value: "1780000000000"},
		{Name: "endTs", Value: "1780100000000"},
		{Name: "pageIndex", Value: "0"},
		{Name: "pageSize", Value: "100"},
	})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if got := compactJSON(t, data); got != `[{"positionId":"9100000000000000001"}]` {
		t.Errorf("data = %s", got)
	}
	requests := fake.recorded()
	if len(requests) != 1 {
		t.Fatalf("%d Requests, erwartet 1", len(requests))
	}
	request := requests[0]
	if request.method != http.MethodGet {
		t.Errorf("Methode = %q, erwartet GET", request.method)
	}
	if request.path != historyPath {
		t.Errorf("Pfad = %q, erwartet %q", request.path, historyPath)
	}
	if got := request.header.Get("X-BX-APIKEY"); got != testAPIKey {
		t.Errorf("X-BX-APIKEY = %q", got)
	}
	signedQuery, _ := splitSignedQuery(t, request.rawQuery)
	expectedQuery := "symbol=ONDO-USDT&startTs=1780000000000&endTs=1780100000000&pageIndex=0&pageSize=100&timestamp=" +
		strconv.FormatInt(firstNowMillis, 10)
	if signedQuery != expectedQuery {
		t.Errorf("signierter Query = %q, erwartet %q", signedQuery, expectedQuery)
	}
	assertValidSignature(t, request)
	if sleeps := harness.sleeps(); len(sleeps) != 0 {
		t.Errorf("Sleep-Aufrufe = %v, erwartet keine bei RequestPause 0", sleeps)
	}
}

func TestClient_QueryAufbau(t *testing.T) {
	testCases := []struct {
		name          string
		baseSuffix    string
		path          string
		params        []bingx.QueryParam
		expectedPath  string
		expectedQuery string
	}{
		{
			name:          "ohne Parameter nur timestamp",
			path:          openPositionsPath,
			params:        nil,
			expectedPath:  openPositionsPath,
			expectedQuery: "timestamp=" + strconv.FormatInt(firstNowMillis, 10),
		},
		{
			name:          "Pfad-Präfix in der Basis-URL bleibt erhalten",
			baseSuffix:    "/ok",
			path:          openPositionsPath,
			expectedPath:  "/ok" + openPositionsPath,
			expectedQuery: "timestamp=" + strconv.FormatInt(firstNowMillis, 10),
		},
		{
			name:          "Werte werden mit url.QueryEscape kodiert, Reihenfolge bleibt",
			path:          historyPath,
			params:        []bingx.QueryParam{{Name: "symbol", Value: "A B&C/D"}, {Name: "a", Value: "1"}},
			expectedPath:  historyPath,
			expectedQuery: "symbol=A+B%26C%2FD&a=1&timestamp=" + strconv.FormatInt(firstNowMillis, 10),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBingX(t, okReply(`[]`))
			client, _ := newTestClient(fake, func(options *bingx.ClientOptions) {
				options.BaseURL = fake.server.URL + testCase.baseSuffix
			})

			if _, err := client.Get(context.Background(), testCase.path, testCase.params); err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}

			requests := fake.recorded()
			if len(requests) != 1 {
				t.Fatalf("%d Requests, erwartet 1", len(requests))
			}
			if requests[0].path != testCase.expectedPath {
				t.Errorf("Pfad = %q, erwartet %q", requests[0].path, testCase.expectedPath)
			}
			signedQuery, _ := splitSignedQuery(t, requests[0].rawQuery)
			if signedQuery != testCase.expectedQuery {
				t.Errorf("signierter Query = %q, erwartet %q", signedQuery, testCase.expectedQuery)
			}
			assertValidSignature(t, requests[0])
		})
	}
}

func TestClient_Huelle(t *testing.T) {
	testCases := []struct {
		name         string
		body         string
		expectedData string // "" bedeutet: kein data (nil)
	}{
		{name: "data als Array", body: okBody(`[{"a":1}]`), expectedData: `[{"a":1}]`},
		{name: "data als Objekt", body: `{"code":0,"data":{"x":"1"}}`, expectedData: `{"x":"1"}`},
		{name: "data null bleibt null", body: okBody(`null`), expectedData: `null`},
		{name: "data fehlt ergibt nil", body: `{"code":0,"msg":""}`, expectedData: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: testCase.body})
			client, _ := newTestClient(fake, nil)

			data, err := client.Get(context.Background(), openPositionsPath, nil)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if testCase.expectedData == "" {
				if len(data) != 0 {
					t.Errorf("data = %q, erwartet nil bei fehlendem Feld", data)
				}
				return
			}
			if got := compactJSON(t, data); got != testCase.expectedData {
				t.Errorf("data = %s, erwartet %s", got, testCase.expectedData)
			}
		})
	}
}

func TestClient_UnerwarteteAntwort(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{"kein JSON", "kein json"},
		{"leerer Body", ""},
		{"JSON null", "null"},
		{"JSON-Array", "[]"},
		{"JSON-String", `"text"`},
		{"code fehlt", `{"msg":"x","data":[]}`},
		{"code als String", `{"code":"0","data":[]}`},
		{"code null", `{"code":null,"data":[]}`},
		{"abgeschnittenes JSON", `{"code":0,"data":[`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: testCase.body})
			client, _ := newTestClient(fake, nil)

			data, err := client.Get(context.Background(), openPositionsPath, nil)

			var unexpected *bingx.UnexpectedResponseError
			if !errors.As(err, &unexpected) {
				t.Fatalf("Fehler = %v (%T), erwartet *UnexpectedResponseError", err, err)
			}
			if data != nil {
				t.Errorf("data = %q, erwartet nil bei Fehler", data)
			}
			assertRequestCount(t, fake, 1)
			assertNoSecrets(t, err, fake)
		})
	}
}

func TestClient_APIFehlerOhneRetry(t *testing.T) {
	t.Run("code 109400 aus der Fixture", func(t *testing.T) {
		fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: readFixture(t, "positionHistory.error.rangeTooLarge.json")})
		client, harness := newTestClient(fake, nil)

		_, err := client.Get(context.Background(), historyPath, []bingx.QueryParam{{Name: "symbol", Value: "ONDO-USDT"}})

		var apiError *bingx.APIError
		if !errors.As(err, &apiError) {
			t.Fatalf("Fehler = %v (%T), erwartet *APIError", err, err)
		}
		if apiError.Code != 109400 {
			t.Errorf("Code = %d, erwartet 109400", apiError.Code)
		}
		if apiError.Msg == "" {
			t.Errorf("Msg leer, erwartet die BingX-Meldung")
		}
		if apiError.Path != historyPath {
			t.Errorf("Path = %q", apiError.Path)
		}
		if text := err.Error(); !strings.Contains(text, "109400") || !strings.Contains(text, historyPath) {
			t.Errorf("Fehlertext %q nennt Code oder Pfad nicht", text)
		}
		assertRequestCount(t, fake, 1)
		if sleeps := harness.sleeps(); len(sleeps) != 0 {
			t.Errorf("Sleep-Aufrufe = %v, erwartet keine", sleeps)
		}
		assertNoSecrets(t, err, fake)
	})

	t.Run("ohne msg", func(t *testing.T) {
		fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: `{"code":100001,"data":{}}`})
		client, _ := newTestClient(fake, nil)

		_, err := client.Get(context.Background(), openPositionsPath, nil)

		var apiError *bingx.APIError
		if !errors.As(err, &apiError) {
			t.Fatalf("Fehler = %v (%T), erwartet *APIError", err, err)
		}
		if apiError.Code != 100001 || apiError.Msg != "" {
			t.Errorf("APIError = %+v", apiError)
		}
		assertRequestCount(t, fake, 1)
	})
}

func TestClient_RetryMitNeuerSignatur(t *testing.T) {
	testCases := []struct {
		name  string
		first fakeReply
	}{
		{"HTTP 429", rateLimitReply},
		{"code 100410", apiCodeReply(100410)},
		{"code 100500", apiCodeReply(100500)},
		{"code 100503", apiCodeReply(100503)},
		{"HTTP 500", fakeReply{status: http.StatusInternalServerError, body: `{"message":"internal"}`}},
		{"HTTP 503", fakeReply{status: http.StatusServiceUnavailable, body: `{"message":"busy"}`}},
		{"Netzwerkfehler", fakeReply{networkError: true}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBingX(t, testCase.first, okReply(`["ok"]`))
			client, harness := newTestClient(fake, nil)

			data, err := client.Get(context.Background(), openPositionsPath, []bingx.QueryParam{{Name: "symbol", Value: "ONDO-USDT"}})
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}

			if got := compactJSON(t, data); got != `["ok"]` {
				t.Errorf("data = %s", got)
			}
			requests := fake.recorded()
			if len(requests) != 2 {
				t.Fatalf("%d Requests, erwartet 2", len(requests))
			}
			firstSigned, firstSignature := splitSignedQuery(t, requests[0].rawQuery)
			secondSigned, secondSignature := splitSignedQuery(t, requests[1].rawQuery)
			if firstSigned == secondSigned {
				t.Errorf("zweiter Versuch hat denselben timestamp: %q", secondSigned)
			}
			if firstSignature == secondSignature {
				t.Errorf("zweiter Versuch hat dieselbe Signatur")
			}
			for _, request := range requests {
				assertValidSignature(t, request)
				if request.method != http.MethodGet || request.header.Get("X-BX-APIKEY") != testAPIKey {
					t.Errorf("Request %s mit Key %q", request.method, request.header.Get("X-BX-APIKEY"))
				}
			}
			if !strings.HasSuffix(secondSigned, "timestamp="+strconv.FormatInt(firstNowMillis+1_000, 10)) {
				t.Errorf("zweiter timestamp passt nicht zur zweiten Now-Zeit: %q", secondSigned)
			}
			if sleeps := harness.sleeps(); !slices.Equal(sleeps, []time.Duration{time.Second}) {
				t.Errorf("Sleep-Aufrufe = %v, erwartet [1s]", sleeps)
			}
		})
	}
}

func TestClient_ExponentiellerBackoff(t *testing.T) {
	fake := newFakeBingX(t, rateLimitReply, rateLimitReply, rateLimitReply, okReply(`[]`))
	client, harness := newTestClient(fake, func(options *bingx.ClientOptions) {
		options.MaxRetries = 3
		options.InitialBackoff = 500 * time.Millisecond
	})

	if _, err := client.Get(context.Background(), openPositionsPath, nil); err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	expected := []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
	if sleeps := harness.sleeps(); !slices.Equal(sleeps, expected) {
		t.Errorf("Sleep-Aufrufe = %v, erwartet %v", sleeps, expected)
	}
	assertRequestCount(t, fake, 4)
}

func TestClient_RequestPauseNurNachErfolg(t *testing.T) {
	t.Run("nach Erfolg", func(t *testing.T) {
		fake := newFakeBingX(t, okReply(`[]`))
		client, harness := newTestClient(fake, func(options *bingx.ClientOptions) { options.RequestPause = 250 * time.Millisecond })

		if _, err := client.Get(context.Background(), openPositionsPath, nil); err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if sleeps := harness.sleeps(); !slices.Equal(sleeps, []time.Duration{250 * time.Millisecond}) {
			t.Errorf("Sleep-Aufrufe = %v, erwartet [250ms]", sleeps)
		}
	})

	t.Run("nach Retry und Erfolg", func(t *testing.T) {
		fake := newFakeBingX(t, rateLimitReply, okReply(`[]`))
		client, harness := newTestClient(fake, func(options *bingx.ClientOptions) { options.RequestPause = 250 * time.Millisecond })

		if _, err := client.Get(context.Background(), openPositionsPath, nil); err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if sleeps := harness.sleeps(); !slices.Equal(sleeps, []time.Duration{time.Second, 250 * time.Millisecond}) {
			t.Errorf("Sleep-Aufrufe = %v, erwartet [1s 250ms]", sleeps)
		}
	})

	t.Run("nicht nach Fehler", func(t *testing.T) {
		fake := newFakeBingX(t, fakeReply{status: http.StatusUnauthorized, body: `{}`})
		client, harness := newTestClient(fake, func(options *bingx.ClientOptions) { options.RequestPause = 250 * time.Millisecond })

		if _, err := client.Get(context.Background(), openPositionsPath, nil); err == nil {
			t.Fatal("Fehler erwartet")
		}
		if sleeps := harness.sleeps(); len(sleeps) != 0 {
			t.Errorf("Sleep-Aufrufe = %v, erwartet keine", sleeps)
		}
	})
}

func TestClient_ErschoepfteRetries(t *testing.T) {
	testCases := []struct {
		name   string
		reply  fakeReply
		assert func(*testing.T, error)
	}{
		{
			name:  "HTTP 429 ergibt RateLimitError",
			reply: rateLimitReply,
			assert: func(t *testing.T, err error) {
				var rateLimit *bingx.RateLimitError
				if !errors.As(err, &rateLimit) {
					t.Fatalf("Fehler = %v (%T), erwartet *RateLimitError", err, err)
				}
				if rateLimit.AttemptCount != 3 || rateLimit.Path != openPositionsPath {
					t.Errorf("RateLimitError = %+v, erwartet 3 Versuche und Pfad", rateLimit)
				}
			},
		},
		{
			name:  "code 100410 ergibt RateLimitError",
			reply: apiCodeReply(100410),
			assert: func(t *testing.T, err error) {
				var rateLimit *bingx.RateLimitError
				if !errors.As(err, &rateLimit) {
					t.Fatalf("Fehler = %v (%T), erwartet *RateLimitError", err, err)
				}
			},
		},
		{
			name:  "code 100500 ergibt APIError",
			reply: apiCodeReply(100500),
			assert: func(t *testing.T, err error) {
				var apiError *bingx.APIError
				if !errors.As(err, &apiError) || apiError.Code != 100500 {
					t.Fatalf("Fehler = %v (%T), erwartet *APIError mit Code 100500", err, err)
				}
			},
		},
		{
			name:  "code 100503 ergibt APIError",
			reply: apiCodeReply(100503),
			assert: func(t *testing.T, err error) {
				var apiError *bingx.APIError
				if !errors.As(err, &apiError) || apiError.Code != 100503 {
					t.Fatalf("Fehler = %v (%T), erwartet *APIError mit Code 100503", err, err)
				}
			},
		},
		{
			name:  "HTTP 500 ergibt HTTPError",
			reply: fakeReply{status: http.StatusInternalServerError, body: `{}`},
			assert: func(t *testing.T, err error) {
				var httpError *bingx.HTTPError
				if !errors.As(err, &httpError) || httpError.Status != http.StatusInternalServerError {
					t.Fatalf("Fehler = %v (%T), erwartet *HTTPError 500", err, err)
				}
				if httpError.Path != openPositionsPath {
					t.Errorf("Path = %q", httpError.Path)
				}
			},
		},
		{
			name:  "HTTP 502 ergibt HTTPError",
			reply: fakeReply{status: http.StatusBadGateway, body: `<html>bad gateway</html>`},
			assert: func(t *testing.T, err error) {
				var httpError *bingx.HTTPError
				if !errors.As(err, &httpError) || httpError.Status != http.StatusBadGateway {
					t.Fatalf("Fehler = %v (%T), erwartet *HTTPError 502", err, err)
				}
			},
		},
		{
			name:  "Netzwerkfehler ergibt NetworkError",
			reply: fakeReply{networkError: true},
			assert: func(t *testing.T, err error) {
				var networkError *bingx.NetworkError
				if !errors.As(err, &networkError) {
					t.Fatalf("Fehler = %v (%T), erwartet *NetworkError", err, err)
				}
				if networkError.CauseName == "" || networkError.AttemptCount != 3 || networkError.Path != openPositionsPath {
					t.Errorf("NetworkError = %+v", networkError)
				}
				if !strings.Contains(err.Error(), openPositionsPath) {
					t.Errorf("Fehlertext %q nennt den Pfad nicht", err.Error())
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeBingX(t, testCase.reply)
			client, harness := newTestClient(fake, func(options *bingx.ClientOptions) { options.MaxRetries = 2 })

			data, err := client.Get(context.Background(), openPositionsPath, nil)

			testCase.assert(t, err)
			if data != nil {
				t.Errorf("data = %q, erwartet nil bei Fehler", data)
			}
			// erster Versuch + 2 Wiederholungen
			assertRequestCount(t, fake, 3)
			if sleeps := harness.sleeps(); !slices.Equal(sleeps, []time.Duration{time.Second, 2 * time.Second}) {
				t.Errorf("Sleep-Aufrufe = %v, erwartet [1s 2s]", sleeps)
			}
			assertNoSecrets(t, err, fake)
		})
	}
}

func TestClient_AndereStatusOhneRetry(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTeapot} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			fake := newFakeBingX(t, fakeReply{status: status, body: `{"message":"nein"}`})
			client, harness := newTestClient(fake, nil)

			_, err := client.Get(context.Background(), openPositionsPath, nil)

			var httpError *bingx.HTTPError
			if !errors.As(err, &httpError) || httpError.Status != status {
				t.Fatalf("Fehler = %v (%T), erwartet *HTTPError %d", err, err, status)
			}
			assertRequestCount(t, fake, 1)
			if sleeps := harness.sleeps(); len(sleeps) != 0 {
				t.Errorf("Sleep-Aufrufe = %v, erwartet keine", sleeps)
			}
			assertNoSecrets(t, err, fake)
		})
	}
}

func TestClient_RedirectWirdNichtGefolgt(t *testing.T) {
	statuses := []int{http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect}
	httpClientVariants := []struct {
		name       string
		httpClient func(*fakeBingX) *http.Client
	}{
		{"eigener HTTPClient ohne CheckRedirect", func(fake *fakeBingX) *http.Client { return fake.server.Client() }},
		{"ohne HTTPClient", func(*fakeBingX) *http.Client { return nil }},
	}

	for _, variant := range httpClientVariants {
		for _, status := range statuses {
			t.Run(variant.name+"/"+strconv.Itoa(status), func(t *testing.T) {
				fake := newFakeBingX(t, fakeReply{status: status, location: redirectedPath})
				client, _ := newTestClient(fake, func(options *bingx.ClientOptions) {
					options.HTTPClient = variant.httpClient(fake)
				})

				_, err := client.Get(context.Background(), openPositionsPath, nil)

				var httpError *bingx.HTTPError
				if !errors.As(err, &httpError) || httpError.Status != status {
					t.Fatalf("Fehler = %v (%T), erwartet *HTTPError %d", err, err, status)
				}
				if hits := fake.redirectedHits(); hits != 0 {
					t.Errorf("Redirect-Ziel %d-mal aufgerufen, erwartet 0 (API-Key darf nicht mitgehen)", hits)
				}
				assertRequestCount(t, fake, 1)
				assertNoSecrets(t, err, fake)
			})
		}
	}
}

func TestNewClient_VeraendertDenUebergebenenHTTPClientNicht(t *testing.T) {
	fake := newFakeBingX(t, okReply(`[]`))
	httpClient := fake.server.Client()

	bingx.NewClient(bingx.ClientOptions{
		APIKey: testAPIKey, APISecret: testAPISecret, BaseURL: fake.server.URL, HTTPClient: httpClient,
		MaxRetries: 1, RequestTimeout: time.Second, InitialBackoff: time.Second,
	})

	if httpClient.CheckRedirect != nil {
		t.Errorf("CheckRedirect des übergebenen HTTPClient wurde gesetzt; erwartet eine Kopie")
	}
}

func TestClient_BodyLimit(t *testing.T) {
	const tenMiB = 10 << 20

	t.Run("Body über 10 MiB ergibt UnexpectedResponseError ohne Retry", func(t *testing.T) {
		oversized := `{"code":0,"msg":"","data":"` + strings.Repeat("a", tenMiB) + `"}`
		fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: oversized})
		client, _ := newTestClient(fake, nil)

		_, err := client.Get(context.Background(), openPositionsPath, nil)

		var unexpected *bingx.UnexpectedResponseError
		if !errors.As(err, &unexpected) {
			t.Fatalf("Fehler = %v (%T), erwartet *UnexpectedResponseError", err, err)
		}
		assertRequestCount(t, fake, 1)
	})

	t.Run("Body knapp unter 10 MiB wird gelesen", func(t *testing.T) {
		payload := strings.Repeat("a", tenMiB-1024)
		fake := newFakeBingX(t, fakeReply{status: http.StatusOK, body: `{"code":0,"msg":"","data":"` + payload + `"}`})
		client, _ := newTestClient(fake, nil)

		data, err := client.Get(context.Background(), openPositionsPath, nil)
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if len(data) != len(payload)+2 {
			t.Errorf("data hat %d Bytes, erwartet %d", len(data), len(payload)+2)
		}
	})
}

func TestClient_RequestTimeoutProVersuch(t *testing.T) {
	fake := newFakeBingX(t, fakeReply{hang: true})
	client, _ := newTestClient(fake, func(options *bingx.ClientOptions) {
		options.MaxRetries = 1
		options.RequestTimeout = 50 * time.Millisecond
	})

	started := time.Now()
	_, err := client.Get(context.Background(), openPositionsPath, nil)
	elapsed := time.Since(started)

	var networkError *bingx.NetworkError
	if !errors.As(err, &networkError) {
		t.Fatalf("Fehler = %v (%T), erwartet *NetworkError", err, err)
	}
	if networkError.CauseName == "" {
		t.Errorf("CauseName leer")
	}
	assertRequestCount(t, fake, 2)
	if elapsed > 3*time.Second {
		t.Errorf("Dauer %v, erwartet Abbruch nach RequestTimeout je Versuch", elapsed)
	}
	assertNoSecrets(t, err, fake)
}

func TestClient_ContextDeadlineDesAufrufers(t *testing.T) {
	fake := newFakeBingX(t, fakeReply{hang: true})
	client, _ := newTestClient(fake, func(options *bingx.ClientOptions) {
		options.MaxRetries = 3
		options.RequestTimeout = 10 * time.Second
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := client.Get(ctx, openPositionsPath, nil)
	elapsed := time.Since(started)

	var networkError *bingx.NetworkError
	if !errors.As(err, &networkError) {
		t.Fatalf("Fehler = %v (%T), erwartet *NetworkError (wird zu unreachable)", err, err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("Dauer %v, erwartet Abbruch mit der Frist des Aufrufers", elapsed)
	}
	// Nach Ablauf der Frist des Aufrufers wird nicht mehr wiederholt.
	assertRequestCount(t, fake, 1)
	assertNoSecrets(t, err, fake)
}

func TestClient_ErfuelltGetClient(_ *testing.T) {
	var _ bingx.GetClient = (*bingx.Client)(nil)
}

// maxBackoff ist die vereinbarte Obergrenze eines einzelnen Backoffs (Konstante im Client).
const maxBackoff = 30 * time.Second

func TestClient_BackoffUeberlaeuftNieUndIstGedeckelt(t *testing.T) {
	const retries = 70
	replies := make([]fakeReply, retries+1)
	for index := range replies {
		replies[index] = rateLimitReply
	}
	fake := newFakeBingX(t, replies...)
	client, harness := newTestClient(fake, func(options *bingx.ClientOptions) {
		options.MaxRetries = retries
		options.InitialBackoff = time.Second
	})

	_, err := client.Get(context.Background(), openPositionsPath, nil)

	var rateLimit *bingx.RateLimitError
	if !errors.As(err, &rateLimit) {
		t.Fatalf("Fehler = %v (%T), erwartet *RateLimitError", err, err)
	}
	sleeps := harness.sleeps()
	if len(sleeps) != retries {
		t.Fatalf("Sleep-Aufrufe = %d, erwartet %d", len(sleeps), retries)
	}
	if sleeps[0] != time.Second {
		t.Errorf("erster Backoff = %v, erwartet 1s", sleeps[0])
	}
	for index, backoff := range sleeps {
		if backoff <= 0 {
			t.Fatalf("Backoff[%d] = %v, erwartet > 0", index, backoff)
		}
		if backoff > maxBackoff {
			t.Fatalf("Backoff[%d] = %v, erwartet <= %v", index, backoff, maxBackoff)
		}
		if index > 0 && backoff < sleeps[index-1] {
			t.Fatalf("Backoff[%d] = %v fällt unter Vorgänger %v", index, backoff, sleeps[index-1])
		}
	}
	if last := sleeps[len(sleeps)-1]; last != maxBackoff {
		t.Errorf("letzter Backoff = %v, erwartet Deckel %v", last, maxBackoff)
	}
}
