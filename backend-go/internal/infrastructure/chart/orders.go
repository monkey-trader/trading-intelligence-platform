package chart

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/chart"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/shared/interfaces"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// openOrdersPath ist der signierte BingX-Endpunkt der offenen Orders.
const openOrdersPath = "/openApi/swap/v2/trade/openOrders"

// idPattern erlaubt IDs nur als Ganzzahl-Literal (number oder String), nie als Float.
var idPattern = regexp.MustCompile(`^\d+$`)

// BingXProtectiveOrderSource holt die offenen SL-/TP-Orders live bei BingX.
type BingXProtectiveOrderSource struct {
	client bingx.GetClient
	clock  interfaces.Clock
}

// NewBingXProtectiveOrderSource baut die Source.
func NewBingXProtectiveOrderSource(client bingx.GetClient, clock interfaces.Clock) *BingXProtectiveOrderSource {
	return &BingXProtectiveOrderSource{client: client, clock: clock}
}

// FetchProtectiveOrders ruft GET /openApi/swap/v2/trade/openOrders auf und übersetzt Fehler.
func (source *BingXProtectiveOrderSource) FetchProtectiveOrders(ctx context.Context) (chart.ProtectiveOrdersSnapshot, error) {
	data, err := source.client.Get(ctx, openOrdersPath, nil)
	if err != nil {
		return chart.ProtectiveOrdersSnapshot{}, toUnavailable(err, "")
	}
	orders, err := MapProtectiveOrders(data)
	if err != nil {
		return chart.ProtectiveOrdersSnapshot{}, err
	}
	// Zeitpunkt erst nach der erfolgreichen Antwort lesen.
	return chart.ProtectiveOrdersSnapshot{Orders: orders, FetchedAt: source.clock.Now().UnixMilli()}, nil
}

// rawOrder hält die Felder roh, damit IDs verlustfrei (ohne float64) gelesen werden.
type rawOrder struct {
	Symbol     *string         `json:"symbol"`
	OrderID    json.RawMessage `json:"orderId"`
	PositionID json.RawMessage `json:"positionID"`
	Type       string          `json:"type"`
	OrigQty    *string         `json:"origQty"`
	StopPrice  *string         `json:"stopPrice"`
	Status     string          `json:"status"`
}

// MapProtectiveOrders mappt das Feld data von openOrders. Nur STOP_MARKET (SL) und
// TAKE_PROFIT_MARKET (TP) mit positionID != 0 zählen; alles andere wird ignoriert.
func MapProtectiveOrders(data json.RawMessage) ([]chart.ProtectiveOrder, error) {
	orders := []chart.ProtectiveOrder{}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return orders, nil
	}
	var envelope struct {
		Orders json.RawMessage `json:"orders"`
	}
	if trimmed[0] != '{' || json.Unmarshal(trimmed, &envelope) != nil {
		return nil, invalidResponse("openOrders: Antwort ist kein Objekt.")
	}
	ordersTrimmed := bytes.TrimSpace(envelope.Orders)
	if len(ordersTrimmed) == 0 || string(ordersTrimmed) == "null" {
		return orders, nil
	}
	var entries []json.RawMessage
	if ordersTrimmed[0] != '[' || json.Unmarshal(ordersTrimmed, &entries) != nil {
		return nil, invalidResponse("openOrders: orders ist kein Array.")
	}
	for _, entry := range entries {
		order, relevant, err := mapProtectiveOrder(entry)
		if err != nil {
			return nil, err
		}
		if relevant {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func mapProtectiveOrder(entry json.RawMessage) (order chart.ProtectiveOrder, relevant bool, err error) {
	var raw rawOrder
	if json.Unmarshal(entry, &raw) != nil {
		return chart.ProtectiveOrder{}, false, invalidResponse("openOrders: Eintrag ist kein gültiges Objekt.")
	}
	positionID, ok := readID(raw.PositionID, true)
	if !ok {
		return chart.ProtectiveOrder{}, false, invalidResponse("openOrders: positionID ist keine Ganzzahl.")
	}
	var kind chart.ProtectiveKind
	switch raw.Type {
	case "STOP_MARKET":
		kind = chart.StopLoss
	case "TAKE_PROFIT_MARKET":
		kind = chart.TakeProfit
	default:
		return chart.ProtectiveOrder{}, false, nil
	}
	if positionID == "" || positionID == "0" {
		return chart.ProtectiveOrder{}, false, nil
	}

	orderID, idOK := readID(raw.OrderID, false)
	if !idOK || raw.Symbol == nil || raw.OrigQty == nil || raw.StopPrice == nil ||
		!isDecimal(*raw.OrigQty) || !isDecimal(*raw.StopPrice) {
		return chart.ProtectiveOrder{}, false, invalidResponse("openOrders: Pflichtfeld fehlt oder ist ungültig.")
	}
	triggerPrice, err := chart.ParseTriggerPrice(*raw.StopPrice)
	if err != nil {
		return chart.ProtectiveOrder{}, false, err
	}
	return chart.ProtectiveOrder{
		OrderID: orderID, PositionID: positionID, Symbol: *raw.Symbol, Kind: kind,
		TriggerPrice: triggerPrice, Quantity: *raw.OrigQty, Status: raw.Status,
	}, true, nil
}

// readID liest eine ID als number oder String verlustfrei. optional: fehlend/null ergibt "" (ok).
func readID(raw json.RawMessage, optional bool) (string, bool) {
	text := string(bytes.TrimSpace(raw))
	if text == "" || text == "null" {
		return "", optional
	}
	if text[0] == '"' {
		var quoted string
		if json.Unmarshal(raw, &quoted) != nil {
			return "", false
		}
		text = quoted
	}
	if !idPattern.MatchString(text) {
		return "", false
	}
	return text, true
}
