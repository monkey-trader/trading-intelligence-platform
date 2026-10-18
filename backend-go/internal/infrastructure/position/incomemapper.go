package position

import (
	"bytes"
	"encoding/json"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
)

const incomeEndpoint = "user/income"

// mapIncomeResponse mappt data von /openApi/swap/v2/user/income.
// data null → leere Liste; kein Array oder fehlerhafter Eintrag → *bingx.UnexpectedResponseError.
func mapIncomeResponse(data json.RawMessage) ([]domainposition.Income, error) {
	trimmed := bytes.TrimSpace(data)
	if string(trimmed) == "null" {
		return []domainposition.Income{}, nil
	}
	var rawEntries []json.RawMessage
	if len(trimmed) == 0 || trimmed[0] != '[' || json.Unmarshal(trimmed, &rawEntries) != nil {
		return nil, unexpected("Antwort von " + incomeEndpoint + " ist kein Array.")
	}
	entries := make([]domainposition.Income, 0, len(rawEntries))
	for _, rawIncome := range rawEntries {
		mapped, err := mapRawIncome(rawIncome)
		if err != nil {
			return nil, err
		}
		entries = append(entries, mapped)
	}
	return entries, nil
}

// mapRawIncome mappt einen Roheintrag; tradeId darf fehlen oder null sein (Funding).
func mapRawIncome(rawIncome json.RawMessage) (domainposition.Income, error) {
	var entry rawEntry
	if json.Unmarshal(rawIncome, &entry) != nil || entry == nil {
		return domainposition.Income{}, unexpected(incomeEndpoint + " enthält einen Eintrag, der kein Objekt ist.")
	}
	fields := fieldReader{entry: entry, endpoint: incomeEndpoint}
	input := domainposition.IncomeInput{
		TranID:     fields.requireString("tranId"),
		Symbol:     fields.requireString("symbol"),
		IncomeType: fields.requireString("incomeType"),
		Amount:     fields.requireString("income"),
		Time:       fields.requireInteger("time"),
		Asset:      fields.requireString("asset"),
		Raw:        string(rawIncome),
	}
	if fields.lookup("tradeId") != nil {
		input.TradeID = fields.requireString("tradeId")
	}
	if fields.firstError != nil {
		return domainposition.Income{}, fields.firstError
	}
	return domainposition.NewIncome(input)
}
