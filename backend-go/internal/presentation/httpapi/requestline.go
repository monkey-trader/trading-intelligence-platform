package httpapi

import (
	"bytes"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
)

// requestLineLimit begrenzt, wie viele Bytes der Wrapper auf der Suche nach dem Ende der
// ersten Zeile puffert. Entspricht Gos echtem Leselimit für den Header-Block:
// MaxHeaderBytes plus 4096 Bytes Puffer (initialReadLimitSize in net/http/server.go).
const requestLineLimit = maxHeaderBytes + 4096

// requestLineReadSize ist die Größe eines einzelnen Lesevorgangs beim Puffern.
const requestLineReadSize = 4 << 10

// wrapListener wrappt jede angenommene Verbindung, damit ein ungültiges Request-Ziel in der
// ersten Zeile durch "/" ersetzt und markiert wird.
func wrapListener(listener net.Listener) net.Listener {
	return &requestLineListener{Listener: listener}
}

// malformedTargetFlagOf liefert das Flag einer von wrapListener gewrappten Verbindung,
// sonst nil.
func malformedTargetFlagOf(conn net.Conn) *atomic.Bool {
	wrappedConn, isWrapped := conn.(*requestLineConn)
	if !isWrapped {
		return nil
	}
	return wrappedConn.malformedTarget
}

// requestLineListener wrappt angenommene Verbindungen, ohne beim Accept zu lesen
// (sonst könnte ein langsamer Client die Accept-Schleife blockieren).
type requestLineListener struct {
	net.Listener
}

// Accept nimmt die nächste Verbindung an und wrappt sie.
func (listener *requestLineListener) Accept() (net.Conn, error) {
	conn, err := listener.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &requestLineConn{Conn: conn, malformedTarget: &atomic.Bool{}}, nil
}

// requestLineConn liest beim ersten Read die erste Zeile, ersetzt ein ungültiges Ziel durch "/"
// und ist danach reiner Passthrough. Deadlines und Write gehen über das eingebettete net.Conn.
// net/http liest eine Verbindung nie nebenläufig, deshalb braucht der Lesezustand keinen Mutex.
type requestLineConn struct {
	net.Conn
	malformedTarget *atomic.Bool
	firstLineRead   bool
	pendingData     []byte
	pendingError    error
}

// Read liefert zuerst die (eventuell umgeschriebene) gepufferte erste Zeile, dann einen
// dabei aufgetretenen Lesefehler unverändert und danach direkt die Daten der Verbindung.
func (conn *requestLineConn) Read(buffer []byte) (int, error) {
	if !conn.firstLineRead {
		conn.firstLineRead = true
		conn.bufferFirstLine()
	}
	if len(conn.pendingData) > 0 {
		bytesCopied := copy(buffer, conn.pendingData)
		conn.pendingData = conn.pendingData[bytesCopied:]
		return bytesCopied, nil
	}
	if conn.pendingError != nil {
		readError := conn.pendingError
		conn.pendingError = nil
		return 0, readError
	}
	return conn.Conn.Read(buffer)
}

// CloseWrite reicht das halbseitige Schließen an die innere Verbindung durch, falls sie es
// anbietet, sonst nil. net/http sucht per Type-Assertion nach einem closeWriter, um nach
// der Antwort ein sauberes FIN statt eines RST zu senden.
func (conn *requestLineConn) CloseWrite() error {
	halfCloser, supportsCloseWrite := conn.Conn.(interface{ CloseWrite() error })
	if !supportsCloseWrite {
		return nil
	}
	return halfCloser.CloseWrite()
}

// bufferFirstLine puffert Teil-Reads bis zum ersten LF, bis zum Limit oder bis zu einem Fehler.
func (conn *requestLineConn) bufferFirstLine() {
	var collected []byte
	readBuffer := make([]byte, requestLineReadSize)
	for len(collected) < requestLineLimit {
		readSize := min(requestLineReadSize, requestLineLimit-len(collected))
		bytesRead, err := conn.Conn.Read(readBuffer[:readSize])
		collected = append(collected, readBuffer[:bytesRead]...)
		if lineEnd := bytes.IndexByte(collected, '\n'); lineEnd >= 0 {
			conn.pendingData = conn.rewriteFirstLine(collected, lineEnd)
			conn.pendingError = err
			return
		}
		if err != nil {
			conn.pendingData = collected
			conn.pendingError = err
			return
		}
	}
	// Kein LF bis zum Limit: unverändert durchreichen, net/http antwortet selbst (431).
	conn.pendingData = collected
}

// rewriteFirstLine ersetzt ein ungültiges Ziel in "METHOD SP target SP version" durch "/"
// und setzt das Flag. Alles andere bleibt unverändert.
func (conn *requestLineConn) rewriteFirstLine(collected []byte, lineEnd int) []byte {
	line := string(collected[:lineEnd])
	lineBreak := ""
	if trimmedLine, hadCarriageReturn := strings.CutSuffix(line, "\r"); hadCarriageReturn {
		line = trimmedLine
		lineBreak = "\r"
	}
	method, rest, hasFirstSpace := strings.Cut(line, " ")
	target, version, hasSecondSpace := strings.Cut(rest, " ")
	if !hasFirstSpace || !hasSecondSpace {
		return collected
	}
	if _, err := url.ParseRequestURI(target); err == nil {
		return collected
	}
	conn.malformedTarget.Store(true)
	rewritten := []byte(method + " / " + version + lineBreak)
	return append(rewritten, collected[lineEnd:]...)
}
