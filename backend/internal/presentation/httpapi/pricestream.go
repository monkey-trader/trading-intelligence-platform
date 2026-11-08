package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	appprice "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/price"
)

// priceHeartbeatInterval ist der Abstand der Heartbeat-Kommentare.
const priceHeartbeatInterval = 15 * time.Second

// newPriceStreamHandler beantwortet GET /prices/stream als Server-Sent Events.
func newPriceStreamHandler(
	subscribe func() (<-chan appprice.PriceSnapshot, func()),
	heartbeat time.Duration,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeProblem(w, ProblemInternal)
			return
		}
		// Der Stream läuft länger als der WriteTimeout des Servers; nicht unterstützte Writer ignorieren.
		if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
			logger.Debug("Schreibfrist nicht aufhebbar", slog.String("error", err.Error()))
		}
		snapshots, unsubscribe := subscribe()
		defer unsubscribe()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case snapshot, open := <-snapshots:
				if !open {
					return
				}
				payload, err := json.Marshal(toPriceSnapshotDTO(snapshot))
				if err != nil {
					logger.Error("Kurs-Event nicht serialisierbar", slog.String("error", err.Error()))
					return
				}
				if _, err := fmt.Fprintf(w, "event: prices\ndata: %s\n\n", payload); err != nil {
					return
				}
				flusher.Flush()
				ticker.Reset(heartbeat)
			case <-ticker.C:
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
