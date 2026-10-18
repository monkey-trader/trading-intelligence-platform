package chart

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strconv"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/shared/decimal"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

// klinesPath ist der öffentliche BingX-Endpunkt der Kerzen (v3 liefert absteigend, neueste zuerst).
const klinesPath = "/openApi/swap/v3/quote/klines"

// BingXCandleSource holt Kerzen live bei BingX.
type BingXCandleSource struct {
	client bingx.GetClient
}

// NewBingXCandleSource baut die Source.
func NewBingXCandleSource(client bingx.GetClient) *BingXCandleSource {
	return &BingXCandleSource{client: client}
}

// FetchCandles holt die neuesten limit Kerzen (ohne endTime), aufsteigend sortiert.
func (source *BingXCandleSource) FetchCandles(ctx context.Context, symbol string, interval chart.CandleInterval, limit int) ([]chart.Candle, error) {
	data, err := source.client.Get(ctx, klinesPath, []bingx.QueryParam{
		{Name: "symbol", Value: symbol},
		{Name: "interval", Value: string(interval)},
		{Name: "limit", Value: strconv.Itoa(limit)},
	})
	if err != nil {
		return nil, toUnavailable(err, symbol)
	}
	return MapKlines(data, symbol, interval)
}

// rawKline ist ein BingX-Eintrag; Preise müssen Strings sein, time eine Ganzzahl-Zahl.
type rawKline struct {
	Open   *string          `json:"open"`
	High   *string          `json:"high"`
	Low    *string          `json:"low"`
	Close  *string          `json:"close"`
	Volume *string          `json:"volume"`
	Time   *json.RawMessage `json:"time"`
}

// MapKlines mappt das Feld data der Kerzen-Antwort: aufsteigend, ohne Duplikate (gleiche Zeit),
// leere Antwort ergibt eine leere, nicht-nil Liste.
func MapKlines(data json.RawMessage, symbol string, interval chart.CandleInterval) ([]chart.Candle, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return []chart.Candle{}, nil
	}
	var entries []json.RawMessage
	if trimmed[0] != '[' || json.Unmarshal(trimmed, &entries) != nil {
		return nil, invalidResponse("klines: Antwort ist kein Array.")
	}
	candles := make([]chart.Candle, 0, len(entries))
	for _, entry := range entries {
		candle, err := mapKline(entry, symbol, interval)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}
	slices.SortStableFunc(candles, func(left, right chart.Candle) int {
		switch {
		case left.OpenTime < right.OpenTime:
			return -1
		case left.OpenTime > right.OpenTime:
			return 1
		}
		return 0
	})
	candles = slices.CompactFunc(candles, func(left, right chart.Candle) bool { return left.OpenTime == right.OpenTime })
	return candles, nil
}

func mapKline(entry json.RawMessage, symbol string, interval chart.CandleInterval) (chart.Candle, error) {
	var raw rawKline
	if json.Unmarshal(entry, &raw) != nil {
		return chart.Candle{}, invalidResponse("klines: Eintrag ist kein gültiges Objekt.")
	}
	prices := []*string{raw.Open, raw.High, raw.Low, raw.Close, raw.Volume}
	for _, price := range prices {
		if price == nil || !isDecimal(*price) {
			return chart.Candle{}, invalidResponse("klines: Preis oder Volumen fehlt oder ist keine Zahl.")
		}
	}
	openTime, ok := parseInteger(raw.Time)
	if !ok || openTime < 0 {
		return chart.Candle{}, invalidResponse("klines: time fehlt oder ist keine Ganzzahl.")
	}
	return chart.Candle{
		Symbol: symbol, Interval: interval, OpenTime: openTime,
		Open: *raw.Open, High: *raw.High, Low: *raw.Low, Close: *raw.Close, Volume: *raw.Volume,
	}, nil
}

// isDecimal prüft das Dezimalformat ohne Exponent.
func isDecimal(value string) bool {
	_, err := decimal.Subtract(value, "0")
	return err == nil
}

// parseInteger liest eine JSON-Zahl ohne Nachkomma und Exponent; Strings und null sind unzulässig.
func parseInteger(raw *json.RawMessage) (int64, bool) {
	if raw == nil || len(*raw) == 0 || (*raw)[0] == '"' {
		return 0, false
	}
	value, err := strconv.ParseInt(string(*raw), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}
