// Command mcp ist ein MCP-Server (stdio), der das Journal über die HTTP-API des Backends lesend bereitstellt.
package main

import (
	"context"
	"fmt"
	"log"
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
	logger.Printf("gestartet, Backend %s", baseURL)
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	logger.Println("beendet")
	return nil
}
