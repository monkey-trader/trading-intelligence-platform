package httpapi

import (
	"encoding/json"
	"testing"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// firstEntry serialisiert v und liefert die Felder des ersten Eintrags unter key.
func firstEntry(t *testing.T, v any, key string) map[string]any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	if err := json.Unmarshal(parsed[key], &entries); err != nil || len(entries) == 0 {
		t.Fatalf("%s = %s (%v)", key, parsed[key], err)
	}
	return entries[0]
}

func TestOpenPositionDtoEnthaeltBaseAssetDisplayNameUndSwingLong(t *testing.T) {
	overview := appposition.OpenPositionsOverview{Positions: []appposition.OpenPositionEntry{{
		OpenPosition: position.OpenPosition{PositionID: "1", Symbol: "NCCOGOLD2USD-USDT"},
		SwingLong:    true, BaseAsset: "XAU", DisplayName: "Gold",
	}}}
	entry := firstEntry(t, toOpenPositionsSnapshotDto(overview), "positions")
	if entry["baseAsset"] != "XAU" || entry["displayName"] != "Gold" || entry["swingLong"] != true {
		t.Fatalf("entry = %v", entry)
	}
}

func TestClosedPositionDtoEnthaeltBaseAssetDisplayNameUndSwingLong(t *testing.T) {
	views := []appposition.ClosedPositionView{{
		ClosedPosition: position.ClosedPosition{PositionID: "1", Symbol: "BTC-USDT"},
		SwingLong:      true, BaseAsset: "BTC", DisplayName: "BTC",
	}}
	entry := firstEntry(t, toClosedPositionsDto(views), "positions")
	if entry["baseAsset"] != "BTC" || entry["displayName"] != "BTC" || entry["swingLong"] != true {
		t.Fatalf("entry = %v", entry)
	}
}
