package chart

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

const (
	allOrdersPath       = "/openApi/swap/v2/trade/allOrders"
	allOrdersWindowMs   = int64(7 * 24 * 60 * 60 * 1000)
	allOrdersLimit      = "1000"
	allOrdersLimitCount = 1000
	stopMarketOrderType = "STOP_MARKET"
	stopLimitOrderType  = "STOP"
	noPositionIDLiteral = "0"
)

// BingXStopOrderHistorySource liest historische STOP-Orders über allOrders (nur signierte GETs).
type BingXStopOrderHistorySource struct {
	client bingx.GetClient
}

// NewBingXStopOrderHistorySource baut die Source.
func NewBingXStopOrderHistorySource(client bingx.GetClient) *BingXStopOrderHistorySource {
	return &BingXStopOrderHistorySource{client: client}
}

// FetchStopOrders fragt den Zeitraum in Fenstern von höchstens 7 Tagen ab.
func (source *BingXStopOrderHistorySource) FetchStopOrders(ctx context.Context, symbol string, fromMs, toMs int64) ([]position.StopOrderRecord, error) {
	records := []position.StopOrderRecord{}
	for start := fromMs; start <= toMs; {
		end := min(start+allOrdersWindowMs, toMs)
		data, err := source.client.Get(ctx, allOrdersPath, []bingx.QueryParam{
			{Name: "symbol", Value: symbol},
			{Name: "startTime", Value: strconv.FormatInt(start, 10)},
			{Name: "endTime", Value: strconv.FormatInt(end, 10)},
			{Name: "limit", Value: allOrdersLimit},
		})
		if err != nil {
			var rateLimit *bingx.RateLimitError
			if errors.As(err, &rateLimit) {
				err = &interfaces.RateLimitedError{Err: err}
			}
			return nil, fmt.Errorf("allOrders %s: %w", symbol, err)
		}
		windowRecords, stats, err := mapStopOrderRecords(data)
		if err != nil {
			return nil, err
		}
		if stats.read >= allOrdersLimitCount {
			slog.WarnContext(ctx, "allOrders lieferte genau das Limit, Historie evtl. abgeschnitten",
				slog.String("symbol", symbol), slog.Int("limit", allOrdersLimitCount))
		}
		slog.DebugContext(ctx, "allOrders gelesen", slog.String("symbol", symbol),
			slog.Int("orders", stats.read), slog.Int("stop", stats.stop), slog.Int("mitPositionId", len(windowRecords)))
		records = append(records, windowRecords...)
		start = end + 1
	}
	return records, nil
}

type rawHistoryOrder struct {
	PositionID json.RawMessage `json:"positionID"`
	Type       string          `json:"type"`
	StopPrice  string          `json:"stopPrice"`
	Time       int64           `json:"time"`
}

// windowStats zählt gelesene Orders und davon STOP-Orders (für das Debug-Log).
type windowStats struct {
	read int
	stop int
}

func mapStopOrderRecords(data json.RawMessage) ([]position.StopOrderRecord, windowStats, error) {
	records := []position.StopOrderRecord{}
	var stats windowStats
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return records, stats, nil
	}
	var envelope struct {
		Orders []rawHistoryOrder `json:"orders"`
	}
	if trimmed[0] != '{' || json.Unmarshal(trimmed, &envelope) != nil {
		return nil, stats, invalidResponse("allOrders: Antwort ist kein Objekt.")
	}
	stats.read = len(envelope.Orders)
	for _, order := range envelope.Orders {
		if order.Type != stopMarketOrderType && order.Type != stopLimitOrderType {
			continue
		}
		stats.stop++
		positionID, ok := readID(order.PositionID, true)
		if !ok {
			return nil, stats, invalidResponse("allOrders: positionID ist keine Ganzzahl.")
		}
		if positionID == "" || positionID == noPositionIDLiteral || !isDecimal(order.StopPrice) {
			continue
		}
		records = append(records, position.StopOrderRecord{PositionID: positionID, TriggerPrice: order.StopPrice, Time: order.Time})
	}
	return records, stats, nil
}
