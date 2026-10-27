// Package mcp implementiert das Model Context Protocol über stdio (eine JSON-Nachricht pro Zeile).
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/apiclient"
	"github.com/monkey-trader/trading-intelligence-platform/clients/mcp/internal/tools"
)

const protocolVersion = "2025-06-18"

const (
	codeParseError     = -32700
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Server spricht MCP über stdio, eine JSON-Nachricht pro Zeile.
type Server struct {
	registry *tools.Registry
	name     string
	version  string
}

// NewServer legt den Server an.
func NewServer(registry *tools.Registry, name, version string) *Server {
	return &Server{registry: registry, name: name, version: version}
}

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve liest Zeilen aus in und schreibt Antworten nach out, bis in endet.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	lines := make(chan []byte)
	scanErr := make(chan error, 1)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 0, 64*1024), 8<<20)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
		scanErr <- scanner.Err()
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-lines:
			if !ok {
				select {
				case err := <-scanErr:
					if err != nil {
						return fmt.Errorf("eingabe lesen: %w", err)
					}
				default:
				}
				return nil
			}
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			reply := s.handle(ctx, line)
			if reply == nil {
				continue
			}
			data, err := json.Marshal(reply)
			if err != nil {
				return fmt.Errorf("antwort kodieren: %w", err)
			}
			if _, err := out.Write(append(data, '\n')); err != nil {
				return fmt.Errorf("antwort schreiben: %w", err)
			}
		}
	}
}

func (s *Server) handle(ctx context.Context, line []byte) *response {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		return fail(json.RawMessage("null"), codeParseError, "Ungültiges JSON")
	}
	if len(req.ID) == 0 {
		return nil // Benachrichtigung: keine Antwort
	}
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(req.Params, &params) != nil {
			return fail(req.ID, codeInvalidParams, "Ungültige Parameter")
		}
		return ok(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.name, "version": s.version},
		})
	case "ping":
		return ok(req.ID, map[string]any{})
	case "tools/list":
		return ok(req.ID, map[string]any{"tools": s.toolList()})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		return fail(req.ID, codeMethodNotFound, "Methode unbekannt: "+req.Method)
	}
}

func (s *Server) toolList() []map[string]any {
	list := []map[string]any{}
	for _, tool := range s.registry.List() {
		list = append(list, map[string]any{
			"name": tool.Name, "description": tool.Description, "inputSchema": tool.InputSchema,
		})
	}
	return list
}

func (s *Server) callTool(ctx context.Context, req request) *response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(req.Params, &params) != nil || params.Name == "" {
		return fail(req.ID, codeInvalidParams, "Tool-Name fehlt oder ist ungültig")
	}
	tool, found := s.registry.Lookup(params.Name)
	if !found {
		return fail(req.ID, codeInvalidParams, "Unbekanntes Tool: "+params.Name)
	}
	body, err := tool.Call(ctx, params.Arguments)
	if err != nil {
		return ok(req.ID, toolResult(userMessage(err), true))
	}
	return ok(req.ID, toolResult(string(body), false))
}

func toolResult(text string, isError bool) map[string]any {
	result := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
	if isError {
		result["isError"] = true
	}
	return result
}

// userMessage formuliert Fehler als deutschen Klartext ohne interne Details.
func userMessage(err error) string {
	var problem *apiclient.ProblemError
	switch {
	case errors.Is(err, tools.ErrInvalidArguments):
		return err.Error()
	case errors.Is(err, apiclient.ErrUnreachable):
		return "Das Backend ist nicht erreichbar. Läuft es und stimmt TIP_API_URL?"
	case errors.As(err, &problem):
		return "Das Backend meldet: " + problem.Error()
	default:
		return "Unerwarteter Fehler beim Abruf."
	}
}

func ok(id json.RawMessage, result any) *response {
	return &response{JSONRPC: "2.0", ID: id, Result: result}
}

func fail(id json.RawMessage, code int, message string) *response {
	return &response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}
