package position_test

import (
	"context"
	"testing"
	"time"

	appposition "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/position"
)

func TestResolveImportStart_ErsterStartMerktSichJetztUndGibtEsZurueck(t *testing.T) {
	state := &memoryState{values: map[string]int64{}}

	start, err := appposition.ResolveImportStart(context.Background(), state, time.UnixMilli(nowMillis))

	if err != nil {
		t.Fatal(err)
	}
	if start != nowMillis || state.values[appposition.ImportStartedAtKey] != nowMillis {
		t.Errorf("start = %d, state = %v", start, state.values)
	}
}

func TestResolveImportStart_SpaetereStartsBehaltenDenErstenZeitpunkt(t *testing.T) {
	state := &memoryState{values: map[string]int64{appposition.ImportStartedAtKey: 42}}

	start, err := appposition.ResolveImportStart(context.Background(), state, time.UnixMilli(nowMillis))

	if err != nil {
		t.Fatal(err)
	}
	if start != 42 {
		t.Errorf("start = %d, erwartet 42", start)
	}
}

func TestResolveImportStart_StateFehlerWirdGemeldet(t *testing.T) {
	state := &memoryState{values: map[string]int64{}, setErr: errBoom}

	if _, err := appposition.ResolveImportStart(context.Background(), state, time.UnixMilli(nowMillis)); err == nil {
		t.Fatal("Fehler erwartet")
	}
}
