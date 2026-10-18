package position

import (
	"bytes"
	"encoding/json"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// closedPositionsEndpoint steht in jeder Feld-Fehlermeldung, damit im Log klar ist, welche Antwort nicht passte.
const closedPositionsEndpoint = "trade/positionHistory"

// mapClosedPositionsResponse mappt das Feld data von /openApi/swap/v1/trade/positionHistory.
// positionHistory null → leere Liste; fehlendes data, fehlendes Feld oder kein Array → *bingx.UnexpectedResponseError.
func mapClosedPositionsResponse(data json.RawMessage) ([]domainposition.ClosedPosition, error) {
	var envelope rawEntry
	if json.Unmarshal(bytes.TrimSpace(data), &envelope) != nil || envelope == nil {
		return nil, unexpected("Antwort von " + closedPositionsEndpoint + " ist kein Objekt.")
	}
	rawHistory, exists := envelope["positionHistory"]
	if !exists {
		return nil, unexpected("Antwort von " + closedPositionsEndpoint + " enthält kein Feld positionHistory.")
	}
	if string(bytes.TrimSpace(rawHistory)) == "null" {
		return []domainposition.ClosedPosition{}, nil
	}
	var rawEntries []json.RawMessage
	if json.Unmarshal(rawHistory, &rawEntries) != nil {
		return nil, unexpected("positionHistory von " + closedPositionsEndpoint + " ist kein Array.")
	}

	positions := make([]domainposition.ClosedPosition, 0, len(rawEntries))
	for _, rawPosition := range rawEntries {
		mapped, err := mapRawClosedPosition(rawPosition)
		if err != nil {
			return nil, err
		}
		positions = append(positions, mapped)
	}
	return positions, nil
}

// mapRawClosedPosition mappt einen einzelnen Roheintrag von positionHistory auf ein ClosedPosition.
// updateTime ist der Schließzeitpunkt; der Rohtext wird als Raw mitgespeichert.
func mapRawClosedPosition(rawPosition json.RawMessage) (domainposition.ClosedPosition, error) {
	var entry rawEntry
	if json.Unmarshal(rawPosition, &entry) != nil || entry == nil {
		return domainposition.ClosedPosition{}, unexpected(closedPositionsEndpoint + " enthält einen Eintrag, der kein Objekt ist.")
	}

	fields := fieldReader{entry: entry, endpoint: closedPositionsEndpoint}
	input := domainposition.ClosedPositionInput{
		PositionID:     fields.requireString("positionId"),
		Symbol:         fields.requireString("symbol"),
		Side:           fields.requireString("positionSide"),
		Isolated:       fields.requireBoolean("isolated"),
		OpenTime:       fields.requireInteger("openTime"),
		CloseTime:      fields.requireInteger("updateTime"),
		AvgOpenPrice:   fields.requireString("avgPrice"),
		AvgClosePrice:  fields.requireString("avgClosePrice"),
		RealisedProfit: fields.requireString("realisedProfit"),
		NetProfit:      fields.requireString("netProfit"),
		PositionAmount: fields.requireString("positionAmt"),
		ClosedAmount:   fields.requireString("closePositionAmt"),
		Leverage:       fields.requireFloat("leverage"),
		Commission:     fields.requireString("positionCommission"),
		TotalFunding:   fields.requireString("totalFunding"),
		FullyClosed:    fields.requireBoolean("closeAllPositions"),
		Raw:            string(rawPosition),
	}
	if fields.firstError != nil {
		return domainposition.ClosedPosition{}, fields.firstError
	}
	return domainposition.NewClosedPosition(input)
}
