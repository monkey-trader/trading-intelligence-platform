package mcp

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Pfad des MCP-Endpunkts (Streamable HTTP, Protokoll 2025-06-18: ein Pfad für POST und GET).
const HTTPPath = "/mcp"

const maxBodyBytes = 1 << 20

// HTTPHandler bietet den Server per Streamable HTTP an: zustandslos (keine Sessions), jede Antwort
// ein einzelnes application/json, kein SSE-Stream.
type HTTPHandler struct {
	server    *Server
	tokenHash [sha256.Size]byte
}

// NewHTTPHandler legt den Handler an. token darf nicht leer sein.
func NewHTTPHandler(server *Server, token []byte) (*HTTPHandler, error) {
	if len(bytes.TrimSpace(token)) == 0 {
		return nil, errors.New("token fehlt")
	}
	return &HTTPHandler{server: server, tokenHash: sha256.Sum256(token)}, nil
}

// ServeHTTP prüft zuerst den Token (401 ohne Begründung), dann Origin, Methode und Protokollversion.
func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.URL.Path != HTTPPath {
		http.NotFound(w, r)
		return
	}
	// DNS-Rebinding-Schutz: Agenten sind keine Browser und senden keinen Origin-Header.
	if r.Header.Get("Origin") != "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		// GET (kein SSE-Stream) und DELETE (keine Sessions) bietet der Server nicht an.
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil || len(bytes.TrimSpace(body)) == 0 || bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		// Batches gibt es seit 2025-06-18 nicht mehr.
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Der Header gilt laut Spezifikation für alle Anfragen nach der Aushandlung. Bei initialize steht die
	// Version im Body, und Clients (z.B. Hermes mit dem mcp-SDK) senden dort schon ihre eigene, neuere
	// Version im Header. Danach gilt: nicht unterstützte Version = 400.
	if version := r.Header.Get("MCP-Protocol-Version"); version != "" && !supportedProtocolVersions[version] && !isInitialize(body) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	reply, err := h.server.Handle(r.Context(), body)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
	case reply == nil:
		w.WriteHeader(http.StatusAccepted)
	default:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(reply)
	}
}

// authorized vergleicht SHA-256-Hashes in konstanter Zeit, damit weder Inhalt noch Länge des Tokens
// über die Laufzeit durchsickern.
func (h *HTTPHandler) authorized(r *http.Request) bool {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	got := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(got[:], h.tokenHash[:]) == 1
}

// isInitialize erkennt die initialize-Anfrage, ohne den Body weiter auszuwerten.
func isInitialize(body []byte) bool {
	var message struct {
		Method string `json:"method"`
	}
	return json.Unmarshal(body, &message) == nil && message.Method == "initialize"
}
