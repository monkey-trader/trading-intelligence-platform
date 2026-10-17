package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// headerLimit entspricht MaxHeaderBytes (64 KiB).
const headerLimit = 64 << 10

// requestLineReadLimit ist Gos echtes Leselimit für den Header-Block (MaxHeaderBytes + 4096,
// initialReadLimitSize in net/http). Bis hierhin puffert der Wrapper die erste Zeile.
const requestLineReadLimit = headerLimit + 4096

// ---------------------------------------------------------------------------
// Fakes für die Unit-Tests (kein net.Pipe)
// ---------------------------------------------------------------------------

// fakeConn liefert vorgegebene Read-Stücke; danach endlos endlessChunk oder finalError (Standard io.EOF).
type fakeConn struct {
	mutex          sync.Mutex
	chunks         [][]byte
	endlessChunk   []byte
	finalError     error
	readCalls      int
	bytesConsumed  int
	written        bytes.Buffer
	readDeadline   time.Time
	deadlineCalled bool
}

func newFakeConn(chunks ...string) *fakeConn {
	conn := &fakeConn{finalError: io.EOF}
	for _, chunk := range chunks {
		conn.chunks = append(conn.chunks, []byte(chunk))
	}
	return conn
}

// bytewise zerlegt den Text in Ein-Byte-Stücke.
func bytewise(text string) []string {
	pieces := make([]string, 0, len(text))
	for index := 0; index < len(text); index++ {
		pieces = append(pieces, text[index:index+1])
	}
	return pieces
}

func (conn *fakeConn) Read(buffer []byte) (int, error) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.readCalls++
	if len(conn.chunks) > 0 {
		bytesRead := copy(buffer, conn.chunks[0])
		conn.chunks[0] = conn.chunks[0][bytesRead:]
		if len(conn.chunks[0]) == 0 {
			conn.chunks = conn.chunks[1:]
		}
		conn.bytesConsumed += bytesRead
		return bytesRead, nil
	}
	if conn.endlessChunk != nil {
		bytesRead := copy(buffer, conn.endlessChunk)
		conn.bytesConsumed += bytesRead
		return bytesRead, nil
	}
	return 0, conn.finalError
}

func (conn *fakeConn) Write(data []byte) (int, error) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	return conn.written.Write(data)
}

func (conn *fakeConn) Close() error { return nil }
func (conn *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3000}
}
func (conn *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 50000}
}
func (conn *fakeConn) SetDeadline(deadline time.Time) error {
	return conn.SetReadDeadline(deadline)
}
func (conn *fakeConn) SetWriteDeadline(time.Time) error { return nil }
func (conn *fakeConn) SetReadDeadline(deadline time.Time) error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	conn.readDeadline = deadline
	conn.deadlineCalled = true
	return nil
}

func (conn *fakeConn) stats() (readCalls, bytesConsumed int) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	return conn.readCalls, conn.bytesConsumed
}

// fakeListener liefert genau eine Verbindung, danach net.ErrClosed.
type fakeListener struct {
	mutex sync.Mutex
	conn  net.Conn
}

func (listener *fakeListener) Accept() (net.Conn, error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()
	if listener.conn == nil {
		return nil, net.ErrClosed
	}
	conn := listener.conn
	listener.conn = nil
	return conn, nil
}

func (listener *fakeListener) Close() error { return nil }
func (listener *fakeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3000}
}

func acceptWrapped(t *testing.T, conn net.Conn) net.Conn {
	t.Helper()
	wrappedListener := wrapListener(&fakeListener{conn: conn})
	if wrappedListener == nil {
		t.Fatalf("wrapListener liefert nil")
	}
	wrappedConn, err := wrappedListener.Accept()
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	return wrappedConn
}

// readThrough liest mit fester Puffergröße bis zum ersten Fehler; io.EOF gilt als Ende ohne Fehler.
func readThrough(conn net.Conn, bufferSize int) ([]byte, error) {
	var collected []byte
	buffer := make([]byte, bufferSize)
	for {
		bytesRead, err := conn.Read(buffer)
		collected = append(collected, buffer[:bytesRead]...)
		if errors.Is(err, io.EOF) {
			return collected, nil
		}
		if err != nil {
			return collected, err
		}
	}
}

// ---------------------------------------------------------------------------
// Unit-Tests des Wrappers
// ---------------------------------------------------------------------------

func TestWrapListener_AcceptLiestNichts(t *testing.T) {
	conn := newFakeConn("GET /%zz HTTP/1.1\r\n\r\n")

	wrappedConn := acceptWrapped(t, conn)

	if readCalls, _ := conn.stats(); readCalls != 0 {
		t.Fatalf("Accept hat %d-mal gelesen; erwartet 0 (Slowloris)", readCalls)
	}
	if malformedTargetFlagOf(wrappedConn) == nil {
		t.Fatalf("malformedTargetFlagOf liefert nil für eine gewrappte Verbindung")
	}
	if malformedTargetFlagOf(wrappedConn).Load() {
		t.Fatalf("Flag ist vor dem ersten Read gesetzt")
	}
}

func TestMalformedTargetFlagOf_UngewrappteVerbindung(t *testing.T) {
	if flag := malformedTargetFlagOf(newFakeConn()); flag != nil {
		t.Fatalf("malformedTargetFlagOf = %v, erwartet nil für eine ungewrappte Verbindung", flag)
	}
}

func TestWrappedConn_ErsteZeile(t *testing.T) {
	malformedRequest := "GET /%zz HTTP/1.1\r\nHost: 127.0.0.1:3000\r\n\r\n"
	rewrittenRequest := "GET / HTTP/1.1\r\nHost: 127.0.0.1:3000\r\n\r\n"

	testCases := []struct {
		name           string
		chunks         []string
		bufferSize     int
		expectedOutput string
		expectedFlag   bool
	}{
		{
			name: "ungültiges Ziel in einem Stück", chunks: []string{malformedRequest}, bufferSize: 4096,
			expectedOutput: rewrittenRequest, expectedFlag: true,
		},
		{
			name: "ungültiges Ziel Byte für Byte", chunks: bytewise(malformedRequest), bufferSize: 4096,
			expectedOutput: rewrittenRequest, expectedFlag: true,
		},
		{
			name: "LF ohne CR", chunks: []string{"GET /%zz HTTP/1.1\nHost: h\n\n"}, bufferSize: 4096,
			expectedOutput: "GET / HTTP/1.1\nHost: h\n\n", expectedFlag: true,
		},
		{
			name: "LF ohne CR Byte für Byte", chunks: bytewise("GET /%zz HTTP/1.1\nHost: h\n\n"), bufferSize: 4096,
			expectedOutput: "GET / HTTP/1.1\nHost: h\n\n", expectedFlag: true,
		},
		{
			name: "kleiner Lesepuffer beim Aufrufer", chunks: []string{malformedRequest}, bufferSize: 3,
			expectedOutput: rewrittenRequest, expectedFlag: true,
		},
		{
			name:           "nur die erste Zeile wird umgeschrieben",
			chunks:         []string{"GET /%zz HTTP/1.1\r\nX-Pfad: GET /%zz HTTP/1.1\r\n\r\nGET /%zz HTTP/1.1\r\n\r\n"},
			bufferSize:     4096,
			expectedOutput: "GET / HTTP/1.1\r\nX-Pfad: GET /%zz HTTP/1.1\r\n\r\nGET /%zz HTTP/1.1\r\n\r\n",
			expectedFlag:   true,
		},
		{
			name:           "nur die erste Zeile wird umgeschrieben, auch über mehrere Stücke",
			chunks:         []string{"GET /%zz HTTP/1.1\r\nHost: h\r\n", "\r\nGET /%zz HTTP/1.1\r\n\r\n"},
			bufferSize:     4096,
			expectedOutput: "GET / HTTP/1.1\r\nHost: h\r\n\r\nGET /%zz HTTP/1.1\r\n\r\n",
			expectedFlag:   true,
		},
		{
			name: "Methode und Version bleiben erhalten", chunks: []string{"POST /%zz HTTP/1.0\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "POST / HTTP/1.0\r\n\r\n", expectedFlag: true,
		},
		{
			name: "Ziel ohne führenden Slash", chunks: []string{"GET %zz HTTP/1.1\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GET / HTTP/1.1\r\n\r\n", expectedFlag: true,
		},
		{
			name: "gültiges Ziel bleibt unverändert", chunks: []string{"GET /health HTTP/1.1\r\nHost: h\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GET /health HTTP/1.1\r\nHost: h\r\n\r\n", expectedFlag: false,
		},
		{
			name: "ungültige Query bleibt unverändert", chunks: []string{"GET /x?y=%zz HTTP/1.1\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GET /x?y=%zz HTTP/1.1\r\n\r\n", expectedFlag: false,
		},
		{
			name: "OPTIONS * bleibt unverändert", chunks: []string{"OPTIONS * HTTP/1.1\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "OPTIONS * HTTP/1.1\r\n\r\n", expectedFlag: false,
		},
		{
			name: "absolute-form bleibt unverändert", chunks: []string{"GET http://127.0.0.1:3000/health HTTP/1.1\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GET http://127.0.0.1:3000/health HTTP/1.1\r\n\r\n", expectedFlag: false,
		},
		{
			name: "Fehlformat ohne Leerzeichen bleibt unverändert", chunks: []string{"GARBAGE\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GARBAGE\r\n\r\n", expectedFlag: false,
		},
		{
			name: "Fehlformat ohne Version bleibt unverändert", chunks: []string{"GET /%zz\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "GET /%zz\r\n\r\n", expectedFlag: false,
		},
		{
			name: "leere erste Zeile bleibt unverändert", chunks: []string{"\r\nGET /%zz HTTP/1.1\r\n\r\n"}, bufferSize: 4096,
			expectedOutput: "\r\nGET /%zz HTTP/1.1\r\n\r\n", expectedFlag: false,
		},
		{
			name: "EOF ohne LF bleibt unverändert", chunks: []string{"GET /%zz HTTP/1.1"}, bufferSize: 4096,
			expectedOutput: "GET /%zz HTTP/1.1", expectedFlag: false,
		},
		{
			name: "EOF ohne LF Byte für Byte bleibt unverändert", chunks: bytewise("GET /%zz HTTP/1.1"), bufferSize: 4096,
			expectedOutput: "GET /%zz HTTP/1.1", expectedFlag: false,
		},
		{
			name: "leere Verbindung", chunks: nil, bufferSize: 4096,
			expectedOutput: "", expectedFlag: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wrappedConn := acceptWrapped(t, newFakeConn(testCase.chunks...))
			malformedTarget := malformedTargetFlagOf(wrappedConn)
			if malformedTarget == nil {
				t.Fatalf("malformedTargetFlagOf liefert nil für eine gewrappte Verbindung")
			}

			output, err := readThrough(wrappedConn, testCase.bufferSize)

			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if string(output) != testCase.expectedOutput {
				t.Errorf("Ausgabe = %q, erwartet %q", output, testCase.expectedOutput)
			}
			if malformedTarget.Load() != testCase.expectedFlag {
				t.Errorf("Flag = %v, erwartet %v", malformedTarget.Load(), testCase.expectedFlag)
			}
		})
	}
}

func TestWrappedConn_UeberDemLimitWirdUnveraendertDurchgereicht(t *testing.T) {
	overlongLine := "GET /%zz HTTP/1.1" + strings.Repeat("a", requestLineReadLimit) + "\r\n\r\n"
	wrappedConn := acceptWrapped(t, newFakeConn(overlongLine))
	malformedTarget := malformedTargetFlagOf(wrappedConn)
	if malformedTarget == nil {
		t.Fatalf("malformedTargetFlagOf liefert nil für eine gewrappte Verbindung")
	}

	output, err := readThrough(wrappedConn, 4096)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if string(output) != overlongLine {
		t.Errorf("Ausgabe weicht ab (Länge %d, erwartet %d)", len(output), len(overlongLine))
	}
	if malformedTarget.Load() {
		t.Errorf("Flag gesetzt, obwohl die Zeile das Limit überschreitet")
	}
}

func TestWrappedConn_PuffertNichtUeberDasLimitHinaus(t *testing.T) {
	const chunkSize = 1024
	conn := newFakeConn()
	conn.endlessChunk = bytes.Repeat([]byte("a"), chunkSize)
	wrappedConn := acceptWrapped(t, conn)

	readDone := make(chan []byte, 1)
	go func() {
		buffer := make([]byte, 4096)
		bytesRead, _ := wrappedConn.Read(buffer)
		readDone <- buffer[:bytesRead]
	}()

	select {
	case firstData := <-readDone:
		if len(firstData) == 0 || firstData[0] != 'a' {
			t.Errorf("erster Read liefert %q, erwartet die durchgereichten Bytes", firstData)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("der erste Read kehrt bei endloser Zeile ohne LF nicht zurück")
	}
	if _, bytesConsumed := conn.stats(); bytesConsumed > requestLineReadLimit+chunkSize {
		t.Errorf("Wrapper hat %d Bytes gepuffert, erlaubt sind höchstens %d", bytesConsumed, requestLineReadLimit+chunkSize)
	}
}

func TestWrappedConn_LesefehlerWirdUnveraendertDurchgereicht(t *testing.T) {
	conn := newFakeConn("GET /hea")
	conn.finalError = os.ErrDeadlineExceeded
	wrappedConn := acceptWrapped(t, conn)

	output, err := readThrough(wrappedConn, 4096)

	if string(output) != "GET /hea" {
		t.Errorf("Ausgabe = %q, erwartet die bereits gelesenen Bytes %q", output, "GET /hea")
	}
	// Bewusst eine Typzusicherung statt errors.As: net/http erkennt Timeouts nur am direkten net.Error.
	netError, isNetError := err.(net.Error)
	if !isNetError || !netError.Timeout() {
		t.Fatalf("Fehler = %v (%T), erwartet den unveränderten Timeout-Fehler", err, err)
	}
}

func TestWrappedConn_WriteUndDeadlinesGehenAnDieVerbindung(t *testing.T) {
	conn := newFakeConn()
	wrappedConn := acceptWrapped(t, conn)

	if _, err := wrappedConn.Write([]byte("HTTP/1.1 200 OK\r\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	deadline := time.Now().Add(time.Minute)
	if err := wrappedConn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if conn.written.String() != "HTTP/1.1 200 OK\r\n" {
		t.Errorf("geschrieben = %q", conn.written.String())
	}
	if !conn.deadlineCalled || !conn.readDeadline.Equal(deadline) {
		t.Errorf("Read-Deadline wurde nicht an die Verbindung weitergegeben")
	}
}

func TestNewServer_ConnContextMitUngewrappterVerbindung(t *testing.T) {
	server := NewServer(NewHandler(testSettings, discardLogger()), discardLogger())
	if server.ConnContext == nil {
		t.Fatalf("ConnContext fehlt")
	}

	connectionContext := server.ConnContext(context.Background(), newFakeConn())
	request := newTestRequest(http.MethodGet, "/health", allowedTestHost).WithContext(connectionContext)
	recorder := serve(server.Handler, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Status = %d, erwartet 200", recorder.Code)
	}
}

// ---------------------------------------------------------------------------
// Integration: echter Listener, Roh-Requests über net.Dial
// ---------------------------------------------------------------------------

const clientDeadline = 3 * time.Second

type runningServer struct {
	address    string
	serveError chan error
	server     *http.Server
}

// startServer startet Serve auf 127.0.0.1:0. configure darf den Server vor dem Start anpassen
// (z.B. einen kurzen ReadHeaderTimeout setzen).
func startServer(t *testing.T, configure func(*http.Server)) *runningServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	address := listener.Addr().String()
	settings := Settings{AllowedHosts: []string{address}}
	server := NewServer(NewHandler(settings, discardLogger()), discardLogger())
	if configure != nil {
		configure(server)
	}
	running := &runningServer{address: address, serveError: make(chan error, 1), server: server}
	go func() { running.serveError <- Serve(server, listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
	})
	return running
}

func dial(t *testing.T, address string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(clientDeadline)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}
	return conn
}

type rawResponse struct {
	response *http.Response
	body     string
	reader   *bufio.Reader
}

func sendRaw(t *testing.T, address, rawRequest string) rawResponse {
	t.Helper()
	conn := dial(t, address)
	if _, err := io.WriteString(conn, rawRequest); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Body lesen: %v", err)
	}
	_ = response.Body.Close()
	return rawResponse{response: response, body: string(body), reader: reader}
}

func assertRawProblem(t *testing.T, result rawResponse, expectedStatus int, expectedBody string) {
	t.Helper()
	if result.response.StatusCode != expectedStatus {
		t.Errorf("Status = %d, erwartet %d", result.response.StatusCode, expectedStatus)
	}
	if contentType := result.response.Header.Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, erwartet %q", contentType, problemContentType)
	}
	if result.body != expectedBody {
		t.Errorf("Body = %q, erwartet %q", result.body, expectedBody)
	}
}

func TestIntegration_RohRequests(t *testing.T) {
	running := startServer(t, nil)
	allowedHost := running.address

	testCases := []struct {
		name           string
		rawRequest     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "/%zz mit fremdem Host ergibt 403",
			rawRequest:     "GET /%zz HTTP/1.1\r\nHost: evil.example\r\n\r\n",
			expectedStatus: http.StatusForbidden, expectedBody: forbiddenHostBody,
		},
		{
			name:           "/%zz mit erlaubtem Host ergibt 400",
			rawRequest:     "GET /%zz HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusBadRequest, expectedBody: badRequestBody,
		},
		{
			name:           "POST /%zz mit Body und erlaubtem Host ergibt 400",
			rawRequest:     "POST /%zz HTTP/1.1\r\nHost: " + allowedHost + "\r\nContent-Length: 5\r\n\r\n{kap}",
			expectedStatus: http.StatusBadRequest, expectedBody: badRequestBody,
		},
		{
			name:           "OPTIONS * mit fremdem Host ergibt 403",
			rawRequest:     "OPTIONS * HTTP/1.1\r\nHost: evil.example\r\n\r\n",
			expectedStatus: http.StatusForbidden, expectedBody: forbiddenHostBody,
		},
		{
			name:           "OPTIONS * mit erlaubtem Host ergibt 404 (nicht Gos 200)",
			rawRequest:     "OPTIONS * HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusNotFound, expectedBody: notFoundBody,
		},
		{
			name:           "absolute-form ergibt 403, auch mit erlaubtem Host",
			rawRequest:     "GET http://" + allowedHost + "/health HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusForbidden, expectedBody: forbiddenHostBody,
		},
		{
			name:           "absolute-form mit fremdem Ziel ergibt 403",
			rawRequest:     "GET http://evil.example/health HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusForbidden, expectedBody: forbiddenHostBody,
		},
		{
			name:           "ungültige Query ergibt 404",
			rawRequest:     "GET /x?y=%zz HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusNotFound, expectedBody: notFoundBody,
		},
		{
			name:           "DELETE mit Body ergibt 404",
			rawRequest:     "DELETE /positions/open HTTP/1.1\r\nHost: " + allowedHost + "\r\nContent-Length: 3\r\n\r\nabc",
			expectedStatus: http.StatusNotFound, expectedBody: notFoundBody,
		},
		{
			name:           "openPositions-Stub ergibt 503",
			rawRequest:     "GET /positions/open HTTP/1.1\r\nHost: " + allowedHost + "\r\n\r\n",
			expectedStatus: http.StatusServiceUnavailable, expectedBody: notConfiguredBody,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := sendRaw(t, running.address, testCase.rawRequest)
			assertRawProblem(t, result, testCase.expectedStatus, testCase.expectedBody)
		})
	}
}

func TestIntegration_HealthOhneKeepAlive(t *testing.T) {
	running := startServer(t, nil)

	result := sendRaw(t, running.address, "GET /health HTTP/1.1\r\nHost: "+running.address+"\r\n\r\n")

	if result.response.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, erwartet 200", result.response.StatusCode)
	}
	if result.body != `{"status":"ok"}` {
		t.Errorf("Body = %q", result.body)
	}
	if !result.response.Close {
		t.Errorf("Antwort ohne Connection: close; Keep-Alive muss aus sein")
	}
}

func TestIntegration_LF_OhneCR(t *testing.T) {
	running := startServer(t, nil)

	result := sendRaw(t, running.address, "GET /%zz HTTP/1.1\nHost: "+running.address+"\n\n")

	assertRawProblem(t, result, http.StatusBadRequest, badRequestBody)
}

func TestIntegration_GoLehntFehlformatSelbstAb(t *testing.T) {
	running := startServer(t, nil)

	testCases := []struct {
		name           string
		rawRequest     string
		expectedStatus int
	}{
		{name: "Fehlformat der ersten Zeile", rawRequest: "GARBAGE\r\n\r\n", expectedStatus: http.StatusBadRequest},
		{name: "HTTP/1.1 ohne Host", rawRequest: "GET /health HTTP/1.1\r\n\r\n", expectedStatus: http.StatusBadRequest},
		{
			name:           "doppelter Host-Header",
			rawRequest:     "GET /health HTTP/1.1\r\nHost: " + running.address + "\r\nHost: " + running.address + "\r\n\r\n",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := sendRaw(t, running.address, testCase.rawRequest)
			if result.response.StatusCode != testCase.expectedStatus {
				t.Errorf("Status = %d, erwartet %d", result.response.StatusCode, testCase.expectedStatus)
			}
		})
	}
}

func TestIntegration_ZeileUeberDemLimitErgibt431(t *testing.T) {
	running := startServer(t, nil)
	conn := dial(t, running.address)

	go func() {
		// Schreibfehler sind erwartbar, wenn der Server nach der 431 schließt.
		_, _ = io.WriteString(conn, "GET /"+strings.Repeat("a", headerLimit+8<<10))
	}()
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	_ = response.Body.Close()

	if response.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Errorf("Status = %d, erwartet 431", response.StatusCode)
	}
}

func TestIntegration_PipeliningBeantwortetNurDieErsteAnfrage(t *testing.T) {
	running := startServer(t, nil)
	host := running.address
	pipelined := "GET /health HTTP/1.1\r\nHost: " + host + "\r\n\r\n" +
		"GET /positions/open HTTP/1.1\r\nHost: " + host + "\r\n\r\n"

	result := sendRaw(t, running.address, pipelined)

	if result.response.StatusCode != http.StatusOK {
		t.Fatalf("erste Antwort: Status = %d, erwartet 200", result.response.StatusCode)
	}
	rest, err := io.ReadAll(result.reader)
	var netError net.Error
	if errors.As(err, &netError) && netError.Timeout() {
		t.Fatalf("Verbindung blieb offen; erwartet Schließen nach der ersten Antwort")
	}
	if len(rest) != 0 {
		t.Errorf("zweite Anfrage wurde beantwortet: %q", rest)
	}
}

func TestIntegration_LangsamerClientLoestReadHeaderTimeoutAus(t *testing.T) {
	testCases := []struct {
		name        string
		partialData string
	}{
		{name: "sendet nichts", partialData: ""},
		{name: "erste Zeile ohne LF", partialData: "GET /hea"},
		{name: "erste Zeile, Header unvollständig", partialData: "GET /health HTTP/1.1\r\nHost: x"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			running := startServer(t, func(server *http.Server) {
				server.ReadHeaderTimeout = 300 * time.Millisecond
			})
			conn := dial(t, running.address)
			if testCase.partialData != "" {
				if _, err := io.WriteString(conn, testCase.partialData); err != nil {
					t.Fatalf("Write: %v", err)
				}
			}

			startedAt := time.Now()
			_, err := io.ReadAll(conn)
			elapsed := time.Since(startedAt)

			var netError net.Error
			if errors.As(err, &netError) && netError.Timeout() {
				t.Fatalf("Server hat die Verbindung nicht nach ReadHeaderTimeout geschlossen")
			}
			if elapsed > 2*time.Second {
				t.Errorf("Schließen dauerte %v, erwartet etwa 300ms", elapsed)
			}
		})
	}
}

func TestIntegration_LangsamerClientBlockiertAndereNicht(t *testing.T) {
	running := startServer(t, nil)
	slowConn := dial(t, running.address)
	if _, err := io.WriteString(slowConn, "GET /hea"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	result := sendRaw(t, running.address, "GET /health HTTP/1.1\r\nHost: "+running.address+"\r\n\r\n")

	if result.response.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, erwartet 200", result.response.StatusCode)
	}
}

func TestServe_LiefertErrServerClosedNachClose(t *testing.T) {
	running := startServer(t, nil)
	// Erst eine Anfrage, damit Serve sicher läuft.
	sendRaw(t, running.address, "GET /health HTTP/1.1\r\nHost: "+running.address+"\r\n\r\n")

	if err := running.server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case err := <-running.serveError:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve = %v, erwartet http.ErrServerClosed", err)
		}
	case <-time.After(clientDeadline):
		t.Fatalf("Serve kehrt nach Close nicht zurück")
	}
}

// ---------------------------------------------------------------------------
// CloseWrite: net/http sendet nur dann ein FIN (statt nach 500 ms ein RST), wenn die
// Verbindung CloseWrite anbietet.
// ---------------------------------------------------------------------------

// closeWriter entspricht dem unexportierten Interface, das net/http vor dem Schließen prüft.
type closeWriter interface {
	CloseWrite() error
}

// fakeConnWithCloseWrite ist eine Fake-Conn, die wie *net.TCPConn CloseWrite anbietet.
type fakeConnWithCloseWrite struct {
	*fakeConn
	closeWriteCalls int
	closeWriteError error
}

func (conn *fakeConnWithCloseWrite) CloseWrite() error {
	conn.closeWriteCalls++
	return conn.closeWriteError
}

var errCloseWriteFake = errors.New("CloseWrite-Fehler der inneren Verbindung")

func TestWrappedConn_CloseWrite(t *testing.T) {
	testCases := []struct {
		name          string
		innerError    error
		expectedError error
	}{
		{name: "reicht an die innere Verbindung durch", innerError: nil, expectedError: nil},
		{name: "reicht den Fehler der inneren Verbindung durch", innerError: errCloseWriteFake, expectedError: errCloseWriteFake},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			innerConn := &fakeConnWithCloseWrite{fakeConn: newFakeConn(), closeWriteError: testCase.innerError}
			wrappedConn := acceptWrapped(t, innerConn)

			halfCloser, implementsCloseWrite := wrappedConn.(closeWriter)
			if !implementsCloseWrite {
				t.Fatalf("gewrappte Verbindung (%T) bietet kein CloseWrite an; net/http sendet dann kein FIN", wrappedConn)
			}
			err := halfCloser.CloseWrite()

			if !errors.Is(err, testCase.expectedError) {
				t.Errorf("CloseWrite = %v, erwartet %v", err, testCase.expectedError)
			}
			if innerConn.closeWriteCalls != 1 {
				t.Errorf("CloseWrite der inneren Verbindung %d-mal aufgerufen, erwartet 1", innerConn.closeWriteCalls)
			}
		})
	}
}

func TestWrappedConn_CloseWriteOhneInneresCloseWriteLiefertNil(t *testing.T) {
	wrappedConn := acceptWrapped(t, newFakeConn())

	halfCloser, implementsCloseWrite := wrappedConn.(closeWriter)
	if !implementsCloseWrite {
		t.Fatalf("gewrappte Verbindung (%T) bietet kein CloseWrite an", wrappedConn)
	}
	if err := halfCloser.CloseWrite(); err != nil {
		t.Errorf("CloseWrite = %v, erwartet nil, wenn die innere Verbindung kein CloseWrite kennt", err)
	}
}

func TestIntegration_UngelesenerBodyEndetMitEOFStattReset(t *testing.T) {
	// Größer als die Socket-Puffer, damit beim Schließen ungelesene Daten übrig bleiben.
	const bodySize = 1 << 20
	// net/http wartet nach CloseWrite 500 ms (rstAvoidanceDelay), bevor es schließt.
	const maximumWaitForEOF = 250 * time.Millisecond

	running := startServer(t, nil)
	conn := dial(t, running.address)

	header := "POST /gibt-es-nicht HTTP/1.1\r\nHost: " + running.address +
		"\r\nContent-Type: application/octet-stream\r\nContent-Length: " + fmt.Sprint(bodySize) + "\r\n\r\n"
	go func() {
		// Schreibfehler sind erwartbar, sobald der Server die Verbindung schließt.
		_, _ = io.WriteString(conn, header)
		_, _ = conn.Write(bytes.Repeat([]byte("x"), bodySize))
	}()

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatalf("Body lesen: %v", err)
	}
	if response.StatusCode != http.StatusNotFound || string(body) != notFoundBody {
		t.Fatalf("Antwort = %d %q, erwartet 404 %q", response.StatusCode, body, notFoundBody)
	}

	startedAt := time.Now()
	rest, err := io.ReadAll(reader)
	elapsed := time.Since(startedAt)

	if errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("Verbindung nach %v mit RST beendet, erwartet EOF (FIN über CloseWrite)", elapsed)
	}
	if err != nil {
		t.Fatalf("Lesen nach der Antwort: %v, erwartet EOF", err)
	}
	if len(rest) != 0 {
		t.Errorf("unerwartete Daten nach der Antwort: %q", rest)
	}
	if elapsed > maximumWaitForEOF {
		t.Errorf("EOF kam erst nach %v, erwartet deutlich vor 500 ms", elapsed)
	}
}

// ---------------------------------------------------------------------------
// Leselimit des Wrappers = Gos echtes Leselimit (MaxHeaderBytes + 4096)
// ---------------------------------------------------------------------------

// longMalformedRequest baut eine Anfrage mit ungültigem Ziel, deren erste Zeile ohne CRLF
// genau lineLength Bytes lang ist.
func longMalformedRequest(lineLength int, host string) string {
	const prefix = "GET /%zz"
	const suffix = " HTTP/1.1"
	padding := strings.Repeat("a", lineLength-len(prefix)-len(suffix))
	return prefix + padding + suffix + "\r\nHost: " + host + "\r\n\r\n"
}

func TestWrappedConn_LangeZeileUnterGosLeselimitWirdUmgeschrieben(t *testing.T) {
	const lineLength = 66000
	host := "127.0.0.1:3000"
	wrappedConn := acceptWrapped(t, newFakeConn(longMalformedRequest(lineLength, host)))
	malformedTarget := malformedTargetFlagOf(wrappedConn)
	if malformedTarget == nil {
		t.Fatalf("malformedTargetFlagOf liefert nil für eine gewrappte Verbindung")
	}

	output, err := readThrough(wrappedConn, 4096)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	expectedOutput := "GET / HTTP/1.1\r\nHost: " + host + "\r\n\r\n"
	if string(output) != expectedOutput {
		t.Errorf("Ausgabe (Länge %d) = %.80q…, erwartet %q", len(output), output, expectedOutput)
	}
	if !malformedTarget.Load() {
		t.Errorf("Flag nicht gesetzt, obwohl die Zeile (%d Bytes) unter Gos Leselimit (%d) liegt",
			lineLength, requestLineReadLimit)
	}
}

func TestIntegration_LangeZeileMitUngueltigemZiel(t *testing.T) {
	running := startServer(t, nil)

	t.Run("unter Gos Leselimit ergibt 400 problem+json", func(t *testing.T) {
		// 66 000 Bytes: über MaxHeaderBytes (65 536), aber unter MaxHeaderBytes + 4096.
		result := sendRaw(t, running.address, longMalformedRequest(66000, running.address))

		assertRawProblem(t, result, http.StatusBadRequest, badRequestBody)
	})

	t.Run("knapp über Gos Leselimit antwortet Go selbst", func(t *testing.T) {
		// Dokumentierter Grenzfall: Der Wrapper findet kein LF bis zum Limit und reicht unverändert
		// durch; Go antwortet selbst (heute 431). Hier nur Status >= 400 prüfen.
		conn := dial(t, running.address)
		go func() {
			_, _ = io.WriteString(conn, longMalformedRequest(requestLineReadLimit+16, running.address))
		}()
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatalf("ReadResponse: %v", err)
		}
		_ = response.Body.Close()

		if response.StatusCode < http.StatusBadRequest {
			t.Errorf("Status = %d, erwartet >= 400", response.StatusCode)
		}
	})
}
