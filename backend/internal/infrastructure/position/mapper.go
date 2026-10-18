// Package position enthält die Adapter für offene Positionen (BingX, Cache, unkonfiguriert).
package position

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/infrastructure/bingx"
)

// openPositionsEndpoint steht in jeder Feld-Fehlermeldung, damit im Log klar ist, welche Antwort nicht passte.
const openPositionsEndpoint = "user/positions"

// rawEntry ist ein BingX-Eintrag, Feld für Feld roh, damit Typen und null explizit geprüft werden.
type rawEntry map[string]json.RawMessage

// mapOpenPositionsResponse mappt das Feld data von /openApi/swap/v2/user/positions.
// data null → leere Liste; fehlendes data (len 0) oder kein Array → *bingx.UnexpectedResponseError.
func mapOpenPositionsResponse(data json.RawMessage) ([]domainposition.OpenPosition, error) {
	trimmed := bytes.TrimSpace(data)
	if string(trimmed) == "null" {
		return []domainposition.OpenPosition{}, nil
	}
	var rawEntries []json.RawMessage
	if len(trimmed) == 0 || trimmed[0] != '[' || json.Unmarshal(trimmed, &rawEntries) != nil {
		return nil, unexpected("Antwort von " + openPositionsEndpoint + " ist kein Array.")
	}

	positions := make([]domainposition.OpenPosition, 0, len(rawEntries))
	for _, rawPosition := range rawEntries {
		mapped, err := mapRawOpenPosition(rawPosition)
		if err != nil {
			return nil, err
		}
		positions = append(positions, mapped)
	}
	return positions, nil
}

func mapRawOpenPosition(rawPosition json.RawMessage) (domainposition.OpenPosition, error) {
	var entry rawEntry
	if json.Unmarshal(rawPosition, &entry) != nil || entry == nil {
		return domainposition.OpenPosition{}, unexpected(openPositionsEndpoint + " enthält einen Eintrag, der kein Objekt ist.")
	}

	fields := fieldReader{entry: entry, endpoint: openPositionsEndpoint}
	input := domainposition.OpenPositionInput{
		PositionID:       fields.requireString("positionId"),
		Symbol:           fields.requireString("symbol"),
		Side:             fields.requireString("positionSide"),
		Isolated:         fields.requireBoolean("isolated"),
		PositionAmount:   fields.requireString("positionAmt"),
		AvgOpenPrice:     fields.requireString("avgPrice"),
		MarkPrice:        fields.requireString("markPrice"),
		Leverage:         fields.requireFloat("leverage"),
		InitialMargin:    fields.requireString("initialMargin"),
		Margin:           fields.requireString("margin"),
		PositionValue:    fields.requireString("positionValue"),
		UnrealizedProfit: fields.requireString("unrealizedProfit"),
		RealisedProfit:   fields.requireString("realisedProfit"),
		PnlRatio:         fields.requireString("pnlRatio"),
		LiquidationPrice: fields.requireLiquidationPrice("liquidationPrice"),
		OpenTime:         fields.requireInteger("createTime"),
		UpdateTime:       fields.requireInteger("updateTime"),
	}
	if fields.firstError != nil {
		return domainposition.OpenPosition{}, fields.firstError
	}
	// Fehler der Factory (PositionDomainError) gehen unverändert weiter.
	return domainposition.NewOpenPosition(input)
}

// fieldReader liest Felder defensiv und merkt sich den ersten Fehler. Fehlende Felder und null
// führen zu *bingx.UnexpectedResponseError, nie zu stillen Standardwerten.
type fieldReader struct {
	entry      rawEntry
	endpoint   string
	firstError error
}

// lookup liefert den rohen Wert oder nil, wenn das Feld fehlt, null ist oder schon ein Fehler vorliegt.
func (reader *fieldReader) lookup(fieldName string) json.RawMessage {
	if reader.firstError != nil {
		return nil
	}
	rawValue, exists := reader.entry[fieldName]
	if !exists || string(rawValue) == "null" {
		return nil
	}
	return rawValue
}

func (reader *fieldReader) fail(fieldName, expectedType string) {
	if reader.firstError == nil {
		reader.firstError = unexpected(fmt.Sprintf("%s: Feld %s fehlt oder ist kein %s.",
			reader.endpoint, fieldName, expectedType))
	}
}

func (reader *fieldReader) requireString(fieldName string) string {
	rawValue := reader.lookup(fieldName)
	var value string
	if rawValue == nil || json.Unmarshal(rawValue, &value) != nil {
		reader.fail(fieldName, "String")
		return ""
	}
	return value
}

func (reader *fieldReader) requireBoolean(fieldName string) bool {
	rawValue := reader.lookup(fieldName)
	var value bool
	if rawValue == nil || json.Unmarshal(rawValue, &value) != nil {
		reader.fail(fieldName, "Boolean")
		return false
	}
	return value
}

// requireNumber liefert eine JSON-Zahl. Ein String wird abgelehnt, obwohl json.Number ihn annehmen würde.
func (reader *fieldReader) requireNumber(fieldName string) (json.Number, bool) {
	rawValue := reader.lookup(fieldName)
	var value json.Number
	if rawValue == nil || !isJSONNumberLiteral(rawValue) || json.Unmarshal(rawValue, &value) != nil {
		return "", false
	}
	return value, true
}

func (reader *fieldReader) requireFloat(fieldName string) float64 {
	number, ok := reader.requireNumber(fieldName)
	value, err := number.Float64()
	if !ok || err != nil {
		reader.fail(fieldName, "Zahl")
		return 0
	}
	return value
}

// requireInteger akzeptiert Zeiten nur als Ganzzahl ohne Nachkomma und Exponent.
func (reader *fieldReader) requireInteger(fieldName string) int64 {
	number, ok := reader.requireNumber(fieldName)
	value, err := strconv.ParseInt(number.String(), 10, 64)
	if !ok || err != nil {
		reader.fail(fieldName, "Ganzzahl")
		return 0
	}
	return value
}

// requireLiquidationPrice: BingX liefert hier eine number, sonst Strings. Eine endliche JSON-Zahl
// (auch mit Exponent) wird wie String(n) in TS plus toPlainDecimal ohne Exponent geschrieben
// ("-0" → "0"). Ein String wird übernommen; nur ein String mit Exponent ist ein Fehler.
func (reader *fieldReader) requireLiquidationPrice(fieldName string) string {
	rawValue := reader.lookup(fieldName)
	if rawValue != nil && rawValue[0] == '"' {
		return reader.requireString(fieldName)
	}
	number, ok := reader.requireNumber(fieldName)
	value, err := number.Float64()
	if !ok || err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		reader.fail(fieldName, "String oder endliche Zahl")
		return ""
	}
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// isJSONNumberLiteral prüft das erste Zeichen: Zahlen beginnen mit '-' oder einer Ziffer.
func isJSONNumberLiteral(rawValue json.RawMessage) bool {
	first := rawValue[0]
	return first == '-' || (first >= '0' && first <= '9')
}

func unexpected(message string) error {
	return &bingx.UnexpectedResponseError{Message: message}
}
