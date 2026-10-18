// Package bingx ist ein minimaler BingX-Client: nur signierte GET-Requests.
package bingx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// QueryParam ist ein Query-Parameter; die Reihenfolge im Slice bleibt erhalten.
type QueryParam struct {
	Name  string
	Value string
}

// GetClient ist die minimale Sicht der Sources auf den Client. Liefert das Feld data roh:
// fehlendes data → nil (len 0), data: null → json.RawMessage("null").
type GetClient interface {
	Get(ctx context.Context, path string, params []QueryParam) (json.RawMessage, error)
}

// ClientOptions konfiguriert den Client. Sleep und Now sind für Tests injizierbar.
type ClientOptions struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	HTTPClient *http.Client
	// Sleep wartet d oder bricht mit ctx ab.
	Sleep func(ctx context.Context, d time.Duration) error
	Now   func() time.Time
	// MaxRetries zählt die Wiederholungen zusätzlich zum ersten Versuch.
	MaxRetries     int
	RequestPause   time.Duration
	InitialBackoff time.Duration
	// RequestTimeout begrenzt jeden einzelnen Versuch; 0 bedeutet ohne eigene Frist.
	RequestTimeout time.Duration
}

const (
	apiKeyHeader = "X-BX-APIKEY"
	// rateLimitCode ist der BingX-Code für ein Rate-Limit bei HTTP 200.
	rateLimitCode = 100410
	// maxBodyBytes begrenzt die gelesene Antwort auf 10 MiB.
	maxBodyBytes = 10 << 20
	// maxBackoff deckelt die Wartezeit eines einzelnen Wiederholungsversuchs.
	maxBackoff = 30 * time.Second
	// maxDiscardBytes: So viel eines Fehler-Bodys wird verworfen, damit die Verbindung wiederverwendbar bleibt.
	maxDiscardBytes = 4096
)

// retryableAPICodes sind vorübergehende BingX-Fehler bei HTTP 200.
var retryableAPICodes = map[int]bool{rateLimitCode: true, 100500: true, 100503: true}

// Client spricht mit BingX.
type Client struct {
	options    ClientOptions
	httpClient *http.Client
}

// NewClient baut den Client. Ein übergebener HTTPClient wird nicht verändert.
func NewClient(options ClientOptions) *Client {
	httpClient := &http.Client{}
	if options.HTTPClient != nil {
		clientCopy := *options.HTTPClient
		httpClient = &clientCopy
	}
	// Redirects nie folgen, damit der API-Key nicht an ein anderes Ziel geht.
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	if options.Sleep == nil {
		options.Sleep = sleepWithContext
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Client{options: options, httpClient: httpClient}
}

// attemptOutcome ist das Ergebnis eines Versuchs: Daten, wiederholbarer oder endgültiger Fehler.
type attemptOutcome struct {
	data      json.RawMessage
	err       error
	retryable bool
}

// Get führt einen signierten GET aus und liefert das Feld data der Hülle.
func (client *Client) Get(ctx context.Context, path string, params []QueryParam) (json.RawMessage, error) {
	attemptCount := client.options.MaxRetries + 1
	var lastFailure error
	for attempt := range attemptCount {
		if attempt > 0 {
			// Ist die Frist des Aufrufers abgelaufen, lohnt keine Wiederholung mehr.
			if ctx.Err() != nil {
				return nil, lastFailure
			}
			backoff := backoffFor(client.options.InitialBackoff, attempt-1)
			if err := client.options.Sleep(ctx, backoff); err != nil {
				return nil, lastFailure
			}
		}

		outcome := client.attempt(ctx, path, params, attemptCount)
		if outcome.err == nil {
			if client.options.RequestPause > 0 {
				// Die Pause schont nur das Rate-Limit; ein Abbruch währenddessen ändert das Ergebnis nicht.
				_ = client.options.Sleep(ctx, client.options.RequestPause)
			}
			return outcome.data, nil
		}
		if !outcome.retryable {
			return nil, outcome.err
		}
		lastFailure = outcome.err
	}
	return nil, lastFailure
}

// attempt ist ein einzelner Versuch mit frischem timestamp, frischer Signatur und eigener Frist.
func (client *Client) attempt(ctx context.Context, path string, params []QueryParam, attemptCount int) attemptOutcome {
	attemptContext, cancel := client.attemptContext(ctx)
	defer cancel()

	request, err := http.NewRequestWithContext(attemptContext, http.MethodGet, client.signedURL(path, params), nil)
	if err != nil {
		return attemptOutcome{err: &InvalidRequestError{Path: path}}
	}
	request.Header.Set(apiKeyHeader, client.options.APIKey)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return networkFailure(path, attemptCount, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxDiscardBytes))
	}

	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		return attemptOutcome{err: &RateLimitError{Path: path, AttemptCount: attemptCount}, retryable: true}
	case response.StatusCode >= http.StatusInternalServerError:
		return attemptOutcome{err: &HTTPError{Status: response.StatusCode, Path: path}, retryable: true}
	case response.StatusCode < 200 || response.StatusCode > 299:
		return attemptOutcome{err: &HTTPError{Status: response.StatusCode, Path: path}}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
	if err != nil {
		return networkFailure(path, attemptCount, err)
	}
	if len(body) > maxBodyBytes {
		return attemptOutcome{err: &UnexpectedResponseError{
			Message: fmt.Sprintf("BingX-Antwort bei %s ist größer als 10 MiB.", path),
		}}
	}

	data, err := parseEnvelope(body, path)
	if err != nil {
		var apiError *APIError
		if errors.As(err, &apiError) && retryableAPICodes[apiError.Code] {
			if apiError.Code == rateLimitCode {
				return attemptOutcome{err: &RateLimitError{Path: path, AttemptCount: attemptCount}, retryable: true}
			}
			return attemptOutcome{err: apiError, retryable: true}
		}
		return attemptOutcome{err: err}
	}
	return attemptOutcome{data: data}
}

// backoffFor verdoppelt initial pro Schritt und deckelt bei maxBackoff; die Schleife schließt Überlauf aus.
func backoffFor(initial time.Duration, doublings int) time.Duration {
	backoff := initial
	for range doublings {
		if backoff >= maxBackoff/2 {
			return maxBackoff
		}
		backoff *= 2
	}
	return min(backoff, maxBackoff)
}

func (client *Client) attemptContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if client.options.RequestTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, client.options.RequestTimeout)
}

// signedURL hängt timestamp und Signatur an. Die Basis-URL wird verkettet, damit ein Pfad-Präfix erhalten bleibt.
func (client *Client) signedURL(path string, params []QueryParam) string {
	queryParts := make([]string, 0, len(params)+1)
	for _, param := range params {
		queryParts = append(queryParts, url.QueryEscape(param.Name)+"="+url.QueryEscape(param.Value))
	}
	timestamp := strconv.FormatInt(client.options.Now().UnixMilli(), 10)
	queryParts = append(queryParts, "timestamp="+timestamp)
	query := strings.Join(queryParts, "&")
	return client.options.BaseURL + path + "?" + query + "&signature=" + SignQuery(query, client.options.APISecret)
}

// networkFailure nennt nur den Typnamen der Ursache: *url.Error enthält die signierte URL.
func networkFailure(path string, attemptCount int, err error) attemptOutcome {
	cause := err
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		cause = urlError.Err
	}
	return attemptOutcome{
		err:       &NetworkError{Path: path, AttemptCount: attemptCount, CauseName: fmt.Sprintf("%T", cause)},
		retryable: true,
	}
}

// parseEnvelope prüft die Hülle {code, msg, data} und liefert data roh.
// Fehlendes data ergibt nil, data: null ergibt json.RawMessage("null").
func parseEnvelope(body []byte, path string) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return nil, &UnexpectedResponseError{Message: fmt.Sprintf("BingX lieferte kein gültiges JSON-Objekt bei %s.", path)}
	}

	rawCode, hasCode := envelope["code"]
	if !hasCode {
		return nil, &UnexpectedResponseError{Message: fmt.Sprintf("BingX-Antwort ohne Feld code bei %s.", path)}
	}
	var code int
	if string(rawCode) == "null" || json.Unmarshal(rawCode, &code) != nil {
		return nil, &UnexpectedResponseError{Message: fmt.Sprintf("BingX-Antwort mit ungültigem code bei %s.", path)}
	}

	if code != 0 {
		var apiMessage string
		if rawMessage, hasMessage := envelope["msg"]; hasMessage {
			// Eine Nicht-String-Meldung bleibt leer, wie im TS-Client.
			if json.Unmarshal(rawMessage, &apiMessage) != nil {
				apiMessage = ""
			}
		}
		return nil, &APIError{Code: code, Msg: apiMessage, Path: path}
	}
	return envelope["data"], nil
}

// sleepWithContext wartet die Dauer oder bricht mit dem Context ab.
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
