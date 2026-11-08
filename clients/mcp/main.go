// Command mcp ist ein MCP-Server (stdio), der das Journal über die HTTP-API des Backends lesend bereitstellt.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/mcp"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/tools"
)

const (
	defaultAPIURL = "http://127.0.0.1:3000"
	version       = "0.1.0"
	// minTokenLength verhindert triviale Tokens (Bytes, ohne Zeilenumbruch am Ende).
	minTokenLength = 16
)

func main() {
	// stdout gehört dem Protokoll, Logs gehen nach stderr.
	logger := log.New(os.Stderr, "mcp: ", log.LstdFlags)
	if err := run(logger); err != nil {
		logger.Println(err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	baseURL := os.Getenv("TIP_API_URL")
	if baseURL == "" {
		baseURL = defaultAPIURL
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registry := tools.NewRegistry(apiclient.New(baseURL, 10*time.Second))
	server := mcp.NewServer(registry, "trading-intelligence-mcp", version)
	if addr := os.Getenv("TIP_MCP_HTTP_ADDR"); addr != "" {
		return runHTTP(ctx, logger, server, addr, os.Getenv("TIP_MCP_TOKEN_FILE"))
	}
	logger.Printf("gestartet, Backend %s", baseURL)
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	logger.Println("beendet")
	return nil
}

// runHTTP startet den HTTP-Modus; ohne gültigen Token wird nicht einmal ein Port geöffnet.
func runHTTP(ctx context.Context, logger *log.Logger, server *mcp.Server, addr, tokenFile string) error {
	token, err := readToken(tokenFile)
	if err != nil {
		return err
	}
	handler, err := mcp.NewHTTPHandler(server, token)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("lauschen auf %s: %w", addr, err)
	}
	logger.Printf("HTTP-Modus gestartet auf %s%s", listener.Addr(), mcp.HTTPPath)
	if err := serveHTTP(ctx, listener, handler); err != nil {
		return err
	}
	logger.Println("beendet")
	return nil
}

// readToken liest den Bearer-Token aus einer Datei. Fehlermeldungen nennen nie den Inhalt.
func readToken(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("HTTP-Modus braucht TIP_MCP_TOKEN_FILE")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("token-datei lesen: %w", err)
	}
	token := bytes.TrimSpace(data)
	if len(token) < minTokenLength {
		return nil, fmt.Errorf("token in %s ist kürzer als %d Zeichen", path, minTokenLength)
	}
	return token, nil
}

// serveHTTP bedient listener, bis ctx endet, und fährt dann geordnet herunter.
func serveHTTP(ctx context.Context, listener net.Listener, handler http.Handler) error {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http-server: %w", err)
	}
	<-done
	return nil
}
