package bingx

import "fmt"

// Fehler des BingX-Adapters. Die Texte enthalten nie Key, Secret oder die signierte URL.

// APIError meldet: BingX hat mit code != 0 geantwortet.
type APIError struct {
	Code int
	Msg  string
	Path string
}

// HTTPError meldet: HTTP-Status außerhalb von 2xx (429 und 5xx erst nach allen Wiederholungen).
type HTTPError struct {
	Status int
	Path   string
}

// RateLimitError meldet: HTTP 429 oder code 100410, auch nach allen Wiederholungen.
type RateLimitError struct {
	Path         string
	AttemptCount int
}

// NetworkError meldet: Netzwerkfehler oder Timeout nach allen Wiederholungen. Nur der Typname der Ursache.
type NetworkError struct {
	Path         string
	AttemptCount int
	CauseName    string
}

// UnexpectedResponseError meldet: Die Antwort hat nicht die erwartete Form.
type UnexpectedResponseError struct {
	Message string
}

// InvalidRequestError meldet: Der Request ließ sich nicht bauen (z.B. ungültige Basis-URL).
// Die URL steht bewusst nicht im Text, weil sie die Signatur enthält.
type InvalidRequestError struct {
	Path string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("BingX-Fehler %d bei %s: %s", err.Code, err.Path, err.Msg)
}

func (err *HTTPError) Error() string {
	return fmt.Sprintf("BingX antwortete mit HTTP %d bei %s.", err.Status, err.Path)
}

func (err *RateLimitError) Error() string {
	return fmt.Sprintf("BingX-Rate-Limit bei %s: nach %d Versuchen aufgegeben.", err.Path, err.AttemptCount)
}

func (err *NetworkError) Error() string {
	return fmt.Sprintf("BingX nicht erreichbar bei %s (%s): nach %d Versuchen aufgegeben.",
		err.Path, err.CauseName, err.AttemptCount)
}

func (err *UnexpectedResponseError) Error() string { return err.Message }

func (err *InvalidRequestError) Error() string {
	return fmt.Sprintf("BingX-Request für %s konnte nicht gebaut werden.", err.Path)
}
