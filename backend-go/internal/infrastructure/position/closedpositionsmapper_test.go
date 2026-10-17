package position

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	domainposition "github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/infrastructure/bingx"
)

// firstRawClosedPositionEntry liefert den ersten Eintrag der positionHistory Fixture.
func firstRawClosedPositionEntry(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	content, err := os.ReadFile(fixtureDir + "positionHistory.page0.json")
	if err != nil {
		t.Fatalf("Fixture lesen: %v", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("Fixture parsen: %v", err)
	}
	dataRaw := envelope["data"]
	var dataObj map[string]json.RawMessage
	if err := json.Unmarshal(dataRaw, &dataObj); err != nil {
		t.Fatalf("data parsen: %v", err)
	}
	positionHistoryRaw := dataObj["positionHistory"]
	var positionHistory []json.RawMessage
	if err := json.Unmarshal(positionHistoryRaw, &positionHistory); err != nil {
		t.Fatalf("positionHistory parsen: %v", err)
	}
	if len(positionHistory) == 0 {
		t.Fatalf("Fixture hat keine Einträge")
	}
	var entry map[string]json.RawMessage
	if err := json.Unmarshal(positionHistory[0], &entry); err != nil {
		t.Fatalf("Eintrag parsen: %v", err)
	}
	return entry
}

// dataWithClosedEntry baut ein data-Array mit angepasstem Eintrag.
func dataWithClosedEntry(t *testing.T, changes map[string]string) json.RawMessage {
	t.Helper()
	entry := firstRawClosedPositionEntry(t)
	for fieldName, rawValue := range changes {
		if rawValue == "" {
			delete(entry, fieldName)
			continue
		}
		entry[fieldName] = json.RawMessage(rawValue)
	}
	data := map[string][]map[string]json.RawMessage{
		"positionHistory": {entry},
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return encoded
}

// dataWithRawClosedEntry setzt einen Feldwert als rohen JSON-Text ein.
func dataWithRawClosedEntry(t *testing.T, fieldName, rawValue string) json.RawMessage {
	t.Helper()
	encoded := string(dataWithClosedEntry(t, map[string]string{fieldName: `"PLATZHALTER"`}))
	return json.RawMessage(strings.Replace(encoded, `"PLATZHALTER"`, rawValue, 1))
}

func isUnexpectedResponseError(err error) bool {
	var unexpected *bingx.UnexpectedResponseError
	return errors.As(err, &unexpected)
}

func isPositionDomainError(err error) bool {
	var domainError domainposition.PositionDomainError
	return errors.As(err, &domainError)
}

// Tests

func TestMapClosedPositionsResponse_Fixture(t *testing.T) {
	// positionHistory.page0.json hat 3 Positionen.
	data := fixtureDataClosed(t, "positionHistory.page0.json")

	positions, err := mapClosedPositionsResponse(data)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) != 3 {
		t.Errorf("len(positions) = %d, erwartet 3", len(positions))
	}

	// Prüfe erste Position.
	if len(positions) > 0 {
		p := positions[0]
		if p.PositionID != "9000000000000000000" || p.Symbol != "ONDO-USDT" {
			t.Errorf("Position[0] = (%s, %s), erwartet (9000000000000000000, ONDO-USDT)",
				p.PositionID, p.Symbol)
		}
		if p.Side != domainposition.Long {
			t.Errorf("Position[0].Side = %s, erwartet LONG", p.Side)
		}
		if p.Leverage != 7 {
			t.Errorf("Position[0].Leverage = %v, erwartet 7", p.Leverage)
		}
		if p.AvgOpenPrice != "0.4120" {
			t.Errorf("Position[0].AvgOpenPrice = %s, erwartet 0.4120", p.AvgOpenPrice)
		}
		if p.RealisedProfit != "1.7000" {
			t.Errorf("Position[0].RealisedProfit = %s, erwartet 1.7000", p.RealisedProfit)
		}
	}
}

func TestMapClosedPositionsResponse_Short(t *testing.T) {
	// positionHistory.short.json hat 1 SHORT-Position.
	data := fixtureDataClosed(t, "positionHistory.short.json")

	positions, err := mapClosedPositionsResponse(data)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) != 1 {
		t.Errorf("len(positions) = %d, erwartet 1", len(positions))
	}

	if len(positions) > 0 && positions[0].Side != domainposition.Short {
		t.Errorf("Position[0].Side = %s, erwartet SHORT", positions[0].Side)
	}
}

func TestMapClosedPositionsResponse_Leer(t *testing.T) {
	testCases := []struct {
		name string
		data json.RawMessage
	}{
		{"empty-Fixture", fixtureDataClosed(t, "positionHistory.empty.json")},
		{"positionHistory null", json.RawMessage(`{"positionHistory": null}`)},
		{"leeres Array", json.RawMessage(`{"positionHistory": []}`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			positions, err := mapClosedPositionsResponse(testCase.data)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(positions) != 0 {
				t.Errorf("%d Positionen, erwartet 0", len(positions))
			}
		})
	}
}

func TestMapClosedPositionsResponse_UnerwarteteForm(t *testing.T) {
	testCases := []struct {
		name string
		data json.RawMessage
	}{
		{"data fehlt (nil)", nil},
		{"leeres Objekt", json.RawMessage(`{}`)},
		{"String", json.RawMessage(`"positions"`)},
		{"Zahl", json.RawMessage(`0`)},
		{"Boolean", json.RawMessage(`true`)},
		{"kein JSON", json.RawMessage(`[{`)},
		{"Eintrag null", json.RawMessage(`{"positionHistory": [null]}`)},
		{"Eintrag ist Array", json.RawMessage(`{"positionHistory": [[]]}`)},
		{"Eintrag ist String", json.RawMessage(`{"positionHistory": ["x"]}`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			positions, err := mapClosedPositionsResponse(testCase.data)

			if err == nil {
				t.Fatalf("Fehler erwartet, aber keiner geliefert")
			}
			if !isUnexpectedResponseError(err) || isPositionDomainError(err) {
				t.Errorf("Fehler sollte UnexpectedResponseError sein, aber: %T", err)
			}
			if positions != nil {
				t.Errorf("positions sollte nil sein, aber: %v", positions)
			}
		})
	}
}

func TestMapClosedPositionsResponse_FehlendeFelderZeroBehavior(t *testing.T) {
	// Ein Eintrag mit fehlenden Pflichtfeldern sollte einen Fehler geben.
	data := dataWithClosedEntry(t, map[string]string{
		"symbol": "", // Feld entfernen
	})

	positions, err := mapClosedPositionsResponse(data)

	if err == nil {
		t.Fatalf("Fehler erwartet (fehlendes symbol), aber keiner geliefert")
	}
	if !isUnexpectedResponseError(err) {
		t.Errorf("Fehler sollte UnexpectedResponseError sein, aber: %T", err)
	}
	if positions != nil {
		t.Errorf("positions sollte nil sein")
	}
}

func TestMapClosedPositionsResponse_InvalidLeverage_Error(t *testing.T) {
	// Leverage muss eine valide Zahl sein.
	data := dataWithRawClosedEntry(t, "leverage", `"not-a-number"`)

	positions, err := mapClosedPositionsResponse(data)

	if err == nil {
		t.Fatalf("Fehler erwartet (leverage ist String), aber keiner geliefert")
	}
	if !isUnexpectedResponseError(err) {
		t.Errorf("Fehler sollte UnexpectedResponseError sein (leverage ist kein Zahlenwert), aber: %T", err)
	}
	if positions != nil {
		t.Errorf("positions sollte nil sein")
	}
}

func TestMapClosedPositionsResponse_InvalidTimestamp_Error(t *testing.T) {
	// openTime oder updateTime ungültig.
	data := dataWithRawClosedEntry(t, "openTime", `0`)

	positions, err := mapClosedPositionsResponse(data)

	if err == nil {
		t.Fatalf("Fehler erwartet (openTime=0), aber keiner geliefert")
	}
	if !isPositionDomainError(err) || isUnexpectedResponseError(err) {
		t.Errorf("Fehler sollte PositionDomainError sein, aber: %T", err)
	}
	if positions != nil {
		t.Errorf("positions sollte nil sein")
	}
}

func TestMapClosedPositionsResponse_RawFieldStored(t *testing.T) {
	// Das Raw-Feld sollte die Original-JSON mitspeichern.
	data := fixtureDataClosed(t, "positionHistory.page0.json")

	positions, err := mapClosedPositionsResponse(data)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) > 0 && positions[0].Raw == "" {
		t.Errorf("Position[0].Raw sollte nicht leer sein (Raw JSON sollte gespeichert werden)")
	}
}

func TestMapClosedPositionsResponse_UpdateTimeAsCloseTime(t *testing.T) {
	// updateTime sollte als CloseTime verwendet werden.
	data := fixtureDataClosed(t, "positionHistory.page0.json")

	positions, err := mapClosedPositionsResponse(data)

	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) > 0 {
		p := positions[0]
		// Fixture hat updateTime 1780018017000.
		if p.CloseTime != 1780018017000 {
			t.Errorf("Position[0].CloseTime = %d, erwartet 1780018017000 (from updateTime)", p.CloseTime)
		}
	}
}

func TestMapClosedPositionsResponse_FieldError_NamesPositionHistoryEndpoint(t *testing.T) {
	data := dataWithClosedEntry(t, map[string]string{"symbol": ""})

	_, err := mapClosedPositionsResponse(data)

	if err == nil {
		t.Fatalf("Fehler erwartet, aber keiner geliefert")
	}
	if !strings.Contains(err.Error(), "positionHistory") {
		t.Errorf("Fehler = %q, sollte positionHistory nennen", err.Error())
	}
	if strings.Contains(err.Error(), "user/positions") {
		t.Errorf("Fehler = %q, darf den Endpunkt der offenen Positionen nicht nennen", err.Error())
	}
}
