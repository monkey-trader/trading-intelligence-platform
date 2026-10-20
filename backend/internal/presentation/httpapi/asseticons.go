package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

type coinIconDto struct {
	Symbol      string  `json:"symbol"`
	BaseAsset   string  `json:"baseAsset"`
	DisplayName string  `json:"displayName"`
	IconURL     *string `json:"iconUrl"`
}

type assetIconsDto struct {
	Icons []coinIconDto `json:"icons"`
}

// newAssetIconsHandler beantwortet GET /assets/icons?symbols=…; Fehlerdetails landen nur im Log.
func newAssetIconsHandler(
	getAssetIcons func(ctx context.Context, list asset.SymbolList) ([]asset.CoinIcon, error),
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := asset.ParseSymbolList(r.URL.Query().Get("symbols"))
		if err != nil {
			writeProblem(w, ProblemBadRequest)
			return
		}
		icons, err := getAssetIcons(r.Context(), list)
		if err != nil {
			logger.Error("Coin-Icons nicht verfügbar", slog.String("error", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		dto := assetIconsDto{Icons: make([]coinIconDto, 0, len(icons))}
		for _, icon := range icons {
			entry := coinIconDto{Symbol: icon.Symbol, BaseAsset: icon.BaseAsset.String(), DisplayName: icon.DisplayName}
			if icon.IconURL != nil {
				text := icon.IconURL.String()
				entry.IconURL = &text
			}
			dto.Icons = append(dto.Icons, entry)
		}
		body, err := json.Marshal(dto)
		if err != nil {
			logger.Error("Coin-Icons nicht serialisierbar", slog.String("error", err.Error()))
			writeProblem(w, ProblemInternal)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
