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

// Fälle portiert aus backend/src/infrastructure/position/bingx/openPositionsMapper.test.ts.

// fixtureDir ist relativ zum Paketordner internal/infrastructure/position (vier Ebenen bis zur Repo-Wurzel).
const fixtureDir = "../../../../fixtures/bingx/"

// fixtureData liest eine Fixture und liefert ihr Feld data roh.
func fixtureData(t *testing.T, name string) json.RawMessage {
	t.Helper()
	content, err := os.ReadFile(fixtureDir + name)
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return envelope["data"]
}

// firstRawEntry liefert den ersten Eintrag der Fixture als Feld-Map zum Abwandeln.
func firstRawEntry(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(fixtureData(t, "openPositions.json"), &entries); err != nil || len(entries) == 0 {
		t.Fatalf("Fixture unvollständig: %v", err)
	}
	return entries[0]
}

// dataWithEntry baut ein data-Array aus dem ersten Fixture-Eintrag mit geänderten oder entfernten Feldern.
// Ein Wert "" entfernt das Feld.
func dataWithEntry(t *testing.T, changes map[string]string) json.RawMessage {
	t.Helper()
	entry := firstRawEntry(t)
	for fieldName, rawValue := range changes {
		if rawValue == "" {
			delete(entry, fieldName)
			continue
		}
		entry[fieldName] = json.RawMessage(rawValue)
	}
	encoded, err := json.Marshal([]map[string]json.RawMessage{entry})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return encoded
}

// dataWithRawEntry setzt einen Feldwert als rohen JSON-Text ein, auch wenn er kein gültiges JSON ist.
func dataWithRawEntry(t *testing.T, fieldName, rawValue string) json.RawMessage {
	t.Helper()
	encoded := string(dataWithEntry(t, map[string]string{fieldName: `"PLATZHALTER"`}))
	return json.RawMessage(strings.Replace(encoded, `"PLATZHALTER"`, rawValue, 1))
}

func isMappingError(err error) bool {
	var unexpected *bingx.UnexpectedResponseError
	var domainError domainposition.PositionDomainError
	return errors.As(err, &unexpected) || errors.As(err, &domainError)
}

func TestMapOpenPositionsResponse_Fixture(t *testing.T) {
	expected := []domainposition.OpenPosition{
		{
			PositionID: "9100000000000000001", Symbol: "SUI-USDT", Side: domainposition.Long, Isolated: true,
			PositionAmount: "500", AvgOpenPrice: "0.9600", MarkPrice: "1.0100", Leverage: 10,
			InitialMargin: "48.0000", Margin: "60.5000", PositionValue: "505", UnrealizedProfit: "25.0000",
			RealisedProfit: "-0.2400", PnlRatio: "0.5208", LiquidationPrice: "0.865",
			OpenTime: 1791500000000, UpdateTime: 1791600000000,
		},
		{
			PositionID: "9100000000000000002", Symbol: "NCCOGOLD2USD-USDT", Side: domainposition.Short, Isolated: true,
			PositionAmount: "0.25", AvgOpenPrice: "4150.00", MarkPrice: "4175.00", Leverage: 15,
			InitialMargin: "69.1667", Margin: "65.0000", PositionValue: "1043.75", UnrealizedProfit: "-6.2500",
			RealisedProfit: "-0.5187", PnlRatio: "-0.0904", LiquidationPrice: "4410.5",
			OpenTime: 1791700000000, UpdateTime: 1791710000000,
		},
	}

	positions, err := mapOpenPositionsResponse(fixtureData(t, "openPositions.json"))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(positions) != len(expected) {
		t.Fatalf("%d Positionen, erwartet %d", len(positions), len(expected))
	}
	for index := range expected {
		if positions[index] != expected[index] {
			t.Errorf("Index %d:\n%+v\nerwartet\n%+v", index, positions[index], expected[index])
		}
	}
}

func TestMapOpenPositionsResponse_Leer(t *testing.T) {
	testCases := []struct {
		name string
		data json.RawMessage
	}{
		{"empty-Fixture", fixtureData(t, "openPositions.empty.json")},
		{"data null", json.RawMessage("null")},
		{"leeres Array", json.RawMessage("[]")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			positions, err := mapOpenPositionsResponse(testCase.data)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(positions) != 0 {
				t.Errorf("%d Positionen, erwartet 0", len(positions))
			}
		})
	}
}

func TestMapOpenPositionsResponse_UnerwarteteForm(t *testing.T) {
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
		{"Eintrag null", json.RawMessage(`[null]`)},
		{"Eintrag ist Array", json.RawMessage(`[[]]`)},
		{"Eintrag ist String", json.RawMessage(`["x"]`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			positions, err := mapOpenPositionsResponse(testCase.data)

			var unexpected *bingx.UnexpectedResponseError
			if !errors.As(err, &unexpected) {
				t.Fatalf("Fehler = %v (%T), erwartet *bingx.UnexpectedResponseError (nie still [])", err, err)
			}
			if positions != nil {
				t.Errorf("Positionen = %v, erwartet nil bei Fehler", positions)
			}
		})
	}
}

func TestMapOpenPositionsResponse_GueltigerUndKaputterEintrag(t *testing.T) {
	valid := string(dataWithEntry(t, nil))
	data := json.RawMessage(strings.TrimSuffix(valid, "]") + `,null]`)

	_, err := mapOpenPositionsResponse(data)

	var unexpected *bingx.UnexpectedResponseError
	if !errors.As(err, &unexpected) {
		t.Fatalf("Fehler = %v (%T), erwartet *bingx.UnexpectedResponseError", err, err)
	}
}

func TestMapOpenPositionsResponse_LiquidationPrice(t *testing.T) {
	testCases := []struct {
		name     string
		rawValue string
		expected string
	}{
		{"String unverändert", `"0.86500"`, "0.86500"},
		{"String 0", `"0"`, "0"},
		{"number 0", `0`, "0"},
		{"number -0", `-0`, "0"},
		{"number -0.0", `-0.0`, "0"},
		{"number 4410.5", `4410.5`, "4410.5"},
		{"number 0.8650 kürzeste Darstellung", `0.8650`, "0.865"},
		{"number 0.0000001 ohne Exponent", `0.0000001`, "0.0000001"},
		{"number 0.30000000000000004", `0.30000000000000004`, "0.30000000000000004"},
		{"große number wie String(n)", `123456789012345678901`, "123456789012345680000"},
		{"number 76507", `76507`, "76507"},
		// Parität mit TS (String(n) + toPlainDecimal): Exponentenform wird ohne Rundung ausgeschrieben.
		{"number 1e-7", `1e-7`, "0.0000001"},
		{"number 1.5e-8", `1.5e-8`, "0.000000015"},
		{"number 1E+21", `1E+21`, "1000000000000000000000"},
		{"number -0e5", `-0e5`, "0"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			positions, err := mapOpenPositionsResponse(dataWithEntry(t, map[string]string{"liquidationPrice": testCase.rawValue}))
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if len(positions) != 1 || positions[0].LiquidationPrice != testCase.expected {
				t.Errorf("LiquidationPrice = %v, erwartet %q", positions, testCase.expected)
			}
		})
	}
}

func TestMapOpenPositionsResponse_UngueltigerLiquidationPrice(t *testing.T) {
	testCases := []struct {
		name     string
		rawValue string
	}{
		{"null", `null`},
		{"Boolean", `true`},
		{"Objekt", `{}`},
		{"String mit Exponent", `"1e-7"`},
		{"NaN", `NaN`},
		{"Infinity", `Infinity`},
		{"-Infinity", `-Infinity`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := mapOpenPositionsResponse(dataWithRawEntry(t, "liquidationPrice", testCase.rawValue))

			if !isMappingError(err) {
				t.Fatalf("Fehler = %v (%T), erwartet Mapping-Fehler", err, err)
			}
		})
	}

	t.Run("fehlt", func(t *testing.T) {
		_, err := mapOpenPositionsResponse(dataWithEntry(t, map[string]string{"liquidationPrice": ""}))

		if !isMappingError(err) {
			t.Fatalf("Fehler = %v (%T), erwartet Mapping-Fehler", err, err)
		}
	})
}

func TestMapOpenPositionsResponse_FalscheFeldtypen(t *testing.T) {
	testCases := []struct {
		name      string
		fieldName string
		rawValue  string // "" = Feld fehlt
	}{
		{"markPrice fehlt", "markPrice", ""},
		{"markPrice null", "markPrice", `null`},
		{"markPrice als number", "markPrice", `1.01`},
		{"positionId als number", "positionId", `9100000000000000000`},
		{"positionId null", "positionId", `null`},
		{"symbol fehlt", "symbol", ""},
		{"positionSide null", "positionSide", `null`},
		{"isolated als String", "isolated", `"true"`},
		{"isolated null", "isolated", `null`},
		{"leverage als String", "leverage", `"10"`},
		{"leverage null", "leverage", `null`},
		{"createTime als String", "createTime", `"1791500000000"`},
		{"createTime null", "createTime", `null`},
		{"createTime mit Nachkomma", "createTime", `1791500000000.5`},
		{"createTime mit Exponent", "createTime", `1.7915e12`},
		{"updateTime fehlt", "updateTime", ""},
		{"unrealizedProfit als number", "unrealizedProfit", `25`},
		{"pnlRatio null", "pnlRatio", `null`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := mapOpenPositionsResponse(dataWithEntry(t, map[string]string{testCase.fieldName: testCase.rawValue}))

			var unexpected *bingx.UnexpectedResponseError
			if !errors.As(err, &unexpected) {
				t.Fatalf("Fehler = %v (%T), erwartet *bingx.UnexpectedResponseError", err, err)
			}
		})
	}
}

func TestMapOpenPositionsResponse_FehlertextNenntEndpunktUndFeld(t *testing.T) {
	_, err := mapOpenPositionsResponse(dataWithEntry(t, map[string]string{"markPrice": ""}))
	if err == nil {
		t.Fatal("Fehler erwartet")
	}
	if text := err.Error(); !strings.Contains(text, "user/positions") || !strings.Contains(text, "markPrice") {
		t.Errorf("Fehlertext %q nennt Endpunkt oder Feld nicht", text)
	}
}

func TestMapOpenPositionsResponse_DomainFehlerDerFactory(t *testing.T) {
	testCases := []struct {
		name      string
		fieldName string
		rawValue  string
		assert    func(*testing.T, error)
	}{
		{"Seite BOTH", "positionSide", `"BOTH"`, assertAs[*domainposition.InvalidPositionSideError]},
		{"positionId abc", "positionId", `"abc"`, assertAs[*domainposition.InvalidPositionIDError]},
		{"createTime 0", "createTime", `0`, assertAs[*domainposition.InvalidTimestampError]},
		{"leverage 0", "leverage", `0`, assertAs[*domainposition.InvalidLeverageError]},
		{"markPrice 1e3 als String", "markPrice", `"1e3"`, assertAs[*domainposition.InvalidDecimalAmountError]},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := mapOpenPositionsResponse(dataWithEntry(t, map[string]string{testCase.fieldName: testCase.rawValue}))

			testCase.assert(t, err)
			var domainError domainposition.PositionDomainError
			if !errors.As(err, &domainError) {
				t.Errorf("Fehler %T ist kein PositionDomainError", err)
			}
		})
	}
}

func assertAs[Target error](t *testing.T, err error) {
	t.Helper()
	var target Target
	if !errors.As(err, &target) {
		t.Fatalf("Fehler = %v (%T), erwartet %T", err, err, target)
	}
}
