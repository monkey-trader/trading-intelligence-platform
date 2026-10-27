// Package apiclient ruft die HTTP-API des Backends auf (einzige Stelle mit net/http).
package apiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUnreachable meldet einen Verbindungsfehler zum Backend.
var ErrUnreachable = errors.New("backend nicht erreichbar")

// ProblemError ist eine application/problem+json-Antwort des Backends.
type ProblemError struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func (e *ProblemError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("%s (HTTP %d)", e.Title, e.Status)
	}
	return fmt.Sprintf("%s (HTTP %d): %s", e.Title, e.Status, e.Detail)
}

// Client ruft die HTTP-API des Backends auf.
type Client struct {
	baseURL string
	http    *http.Client
}

// New legt einen Client mit Basis-URL und Timeout an.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

// Get liefert den Antwort-Body unverändert.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("anfrage aufbauen: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("anfrage abgebrochen: %w", ctx.Err())
		}
		// Die URL steht absichtlich nicht in der Meldung.
		return nil, ErrUnreachable
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrUnreachable
	}
	if resp.StatusCode >= 400 {
		problem := &ProblemError{}
		if json.Unmarshal(body, problem) != nil || problem.Title == "" {
			problem = &ProblemError{Title: http.StatusText(resp.StatusCode)}
		}
		problem.Status = resp.StatusCode
		return nil, problem
	}
	return body, nil
}
