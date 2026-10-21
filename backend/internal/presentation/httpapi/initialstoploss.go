package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
)

const (
	stopLossPathSuffix = "/initial-stop-loss"
	// maxStopLossBodyBytes begrenzt den Request-Body (1 KiB).
	maxStopLossBodyBytes = 1024
	// devServerPort ist der Port des Vite-Dev-Servers, der als Origin erlaubt ist.
	devServerPort = "5173"
)

// stopLossPathSegment liefert die positionId aus /positions/{positionId}/initial-stop-loss.
func stopLossPathSegment(urlPath string) (string, bool) {
	rest, hasPrefix := strings.CutPrefix(urlPath, chartPathPrefix)
	if !hasPrefix {
		return "", false
	}
	segment, hasSuffix := strings.CutSuffix(rest, stopLossPathSuffix)
	if !hasSuffix || segment == "" || strings.Contains(segment, "/") {
		return "", false
	}
	return segment, true
}

type setInitialStopLossRequest struct {
	InitialStopLoss *string `json:"initialStopLoss"`
}

type initialStopLossDto struct {
	PositionID      string `json:"positionId"`
	InitialStopLoss string `json:"initialStopLoss"`
	EntryPrice      string `json:"entryPrice"`
	RiskDistance    string `json:"riskDistance"`
	Source          string `json:"source"`
	SetAt           int64  `json:"setAt"`
}

// newInitialStopLossHandler bedient PUT und DELETE; die Fachlogik liegt in den Use Cases.
func newInitialStopLossHandler(
	set func(ctx context.Context, positionID, rawStopLoss string) (appposition.SetInitialStopLossResult, error),
	clearStopLoss func(ctx context.Context, positionID string) error,
	allowedHosts []string,
	logger *slog.Logger,
) http.HandlerFunc {
	allowedOrigins := originsFor(allowedHosts)
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedOrigin(r.Header, allowedOrigins) {
			writeProblem(w, ProblemForbiddenOrigin)
			return
		}
		if r.Method == http.MethodPut && !hasJSONContentType(r) {
			writeProblem(w, ProblemUnsupportedMediaType)
			return
		}
		positionID, _ := stopLossPathSegment(r.URL.Path)
		if !positionIDPattern.MatchString(positionID) {
			writeProblem(w, ProblemBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			if err := clearStopLoss(r.Context(), positionID); err != nil {
				logger.Error("Ursprungs-SL nicht entfernbar", slog.String("error", truncateForLog(err.Error())))
				writeProblem(w, ProblemInternal)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		rawStopLoss, problem := readStopLossBody(w, r)
		if problem != "" {
			writeProblem(w, problem)
			return
		}
		result, err := set(r.Context(), positionID, rawStopLoss)
		if err != nil {
			problemType := stopLossProblemTypeFor(err)
			logger.Log(r.Context(), slog.LevelWarn, "Ursprungs-SL nicht setzbar",
				slog.String("problem", string(problemType)), slog.String("error", truncateForLog(err.Error())))
			writeProblem(w, problemType)
			return
		}
		body, err := json.Marshal(initialStopLossDto{
			PositionID: result.Override.PositionID, InitialStopLoss: result.Override.InitialStopLoss,
			EntryPrice: result.Override.EntryPrice, RiskDistance: result.RiskDistance,
			Source: string(appposition.RiskSourceManual), SetAt: result.Override.SetAt,
		})
		if err != nil {
			logger.Error("Ursprungs-SL nicht serialisierbar", slog.String("cause", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// readStopLossBody liest den begrenzten Body strikt; leerer Problem-Typ heißt Erfolg.
func readStopLossBody(w http.ResponseWriter, r *http.Request) (string, ProblemType) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxStopLossBodyBytes))
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return "", ProblemPayloadTooLarge
	}
	if err != nil {
		return "", ProblemBadRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request setInitialStopLossRequest
	if err := decoder.Decode(&request); err != nil || request.InitialStopLoss == nil {
		return "", ProblemBadRequest
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", ProblemBadRequest
	}
	return *request.InitialStopLoss, ""
}

// stopLossProblemTypeFor übersetzt Fehler des Set-Use-Cases; alles Unbekannte ist internal.
func stopLossProblemTypeFor(err error) ProblemType {
	var (
		notFound  *chart.PositionNotFoundError
		atEntry   *position.StopLossAtEntryError
		wrongSide *position.StopLossWrongSideError
		invalid   *decimal.InvalidDecimalError
	)
	switch {
	case errors.As(err, &notFound):
		return ProblemNotFound
	case errors.As(err, &atEntry), errors.As(err, &wrongSide), errors.As(err, &invalid):
		return ProblemInvalidStopLoss
	}
	return problemTypeFor(err)
}

func hasJSONContentType(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

// originsFor liefert die erlaubten Origins: alle erlaubten Hosts sowie der Vite-Dev-Server auf Loopback.
func originsFor(allowedHosts []string) map[string]struct{} {
	origins := make(map[string]struct{})
	for _, host := range allowedHosts {
		origins["http://"+strings.ToLower(host)] = struct{}{}
	}
	for _, loopback := range []string{"127.0.0.1", "localhost", "[::1]"} {
		origins["http://"+net.JoinHostPort(strings.Trim(loopback, "[]"), devServerPort)] = struct{}{}
	}
	return origins
}

// isAllowedOrigin: Ohne Origin-Header (kein Browser) ist alles erlaubt, sonst nur bekannte Origins.
func isAllowedOrigin(header http.Header, allowedOrigins map[string]struct{}) bool {
	values, present := header["Origin"]
	if !present {
		return true
	}
	if len(values) != 1 {
		return false
	}
	parsed, err := url.Parse(values[0])
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.Path != "" {
		return false
	}
	_, allowed := allowedOrigins["http://"+strings.ToLower(parsed.Host)]
	return allowed
}
