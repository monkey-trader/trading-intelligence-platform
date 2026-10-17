// Package httpapi enthält die HTTP-API (Host-Prüfung, Routing, Problem-Antworten).
package httpapi

import (
	"encoding/json"
	"net/http"
)

// ProblemType ist der Problem-Typ laut docs/openapi.yaml.
type ProblemType string

// Problem-Typen laut docs/openapi.yaml.
const (
	ProblemBadRequest      ProblemType = "bad-request"
	ProblemForbiddenHost   ProblemType = "forbidden-host"
	ProblemNotFound        ProblemType = "not-found"
	ProblemNotConfigured   ProblemType = "not-configured"
	ProblemRateLimited     ProblemType = "rate-limited"
	ProblemUnreachable     ProblemType = "unreachable"
	ProblemUpstreamError   ProblemType = "upstream-error"
	ProblemInvalidResponse ProblemType = "invalid-response"
	ProblemInternal        ProblemType = "internal"
)

const problemContentTypeHeader = "application/problem+json; charset=utf-8"

type problemDefinition struct {
	status int
	title  string
}

// problemDefinitions: Status und Titel exakt wie in backend/src/presentation/http/problem.ts.
var problemDefinitions = map[ProblemType]problemDefinition{
	ProblemBadRequest:      {http.StatusBadRequest, "Die Anfrage ist ungültig."},
	ProblemForbiddenHost:   {http.StatusForbidden, "Der Host-Header ist nicht erlaubt."},
	ProblemNotFound:        {http.StatusNotFound, "Die angefragte Ressource gibt es nicht."},
	ProblemNotConfigured:   {http.StatusServiceUnavailable, "BingX-Zugang ist nicht konfiguriert."},
	ProblemRateLimited:     {http.StatusServiceUnavailable, "BingX ist ausgelastet, bitte später erneut versuchen."},
	ProblemUnreachable:     {http.StatusGatewayTimeout, "BingX ist nicht erreichbar."},
	ProblemUpstreamError:   {http.StatusBadGateway, "BingX hat mit einem Fehler geantwortet."},
	ProblemInvalidResponse: {http.StatusBadGateway, "BingX hat eine ungültige Antwort geliefert."},
	ProblemInternal:        {http.StatusInternalServerError, "Interner Fehler."},
}

// problemBody ist das Problem-Objekt laut docs/openapi.yaml, ohne interne Details.
type problemBody struct {
	Type   ProblemType `json:"type"`
	Title  string      `json:"title"`
	Status int         `json:"status"`
}

// writeProblem schreibt ein Problem als application/problem+json. Unbekannte Typen werden
// als internal ausgeliefert.
func writeProblem(w http.ResponseWriter, problemType ProblemType) {
	definition, known := problemDefinitions[problemType]
	if !known {
		problemType = ProblemInternal
		definition = problemDefinitions[ProblemInternal]
	}
	// json.Marshal hängt im Gegensatz zu json.Encoder keinen Zeilenumbruch an.
	body, err := json.Marshal(problemBody{Type: problemType, Title: definition.title, Status: definition.status})
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", problemContentTypeHeader)
	w.WriteHeader(definition.status)
	_, _ = w.Write(body)
}
