package sqlite

import (
	"context"
	"reflect"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

var _ interfaces.ImportStateRepository = (*ImportStateRepository)(nil)

func TestImportState_GetFehlendeLiefertNichtGefunden(t *testing.T) {
	repo := NewImportStateRepository(openMemory(t))
	value, found, err := repo.Get(context.Background(), "gibt-es-nicht")
	if err != nil || found || value != 0 {
		t.Fatalf("value=%d found=%v err=%v", value, found, err)
	}
}

func TestImportState_SetGetUndUeberschreiben(t *testing.T) {
	repo := NewImportStateRepository(openMemory(t))
	ctx := context.Background()
	key := "position_history_until:SUI-USDT"
	if err := repo.Set(ctx, key, 1_700_000_000_000); err != nil {
		t.Fatal(err)
	}
	if value, found, err := repo.Get(ctx, key); err != nil || !found || value != 1_700_000_000_000 {
		t.Fatalf("value=%d found=%v err=%v", value, found, err)
	}
	if err := repo.Set(ctx, key, 1_800_000_000_000); err != nil {
		t.Fatal(err)
	}
	if value, _, _ := repo.Get(ctx, key); value != 1_800_000_000_000 {
		t.Errorf("nicht überschrieben: %d", value)
	}
}

func TestImportState_Delete(t *testing.T) {
	repo := NewImportStateRepository(openMemory(t))
	ctx := context.Background()
	_ = repo.Set(ctx, "a", 1)
	if err := repo.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := repo.Get(ctx, "a"); found {
		t.Error("Schlüssel noch vorhanden")
	}
	if err := repo.Delete(ctx, "a"); err != nil {
		t.Errorf("Delete eines fehlenden Schlüssels darf kein Fehler sein: %v", err)
	}
}

func TestImportState_ListKeys(t *testing.T) {
	const prefix = "position_history_until:"
	tests := []struct {
		name   string
		keys   []string
		prefix string
		want   []string
	}{
		{"sortiert, nur Präfix", []string{prefix + "SUI-USDT", prefix + "BTC-USDT", "income_scan_until", "position_suspicious:BTC-USDT"}, prefix,
			[]string{prefix + "BTC-USDT", prefix + "SUI-USDT"}},
		{"Unterstrich ist kein Platzhalter", []string{"aXb", "a_b"}, "a_", []string{"a_b"}},
		{"Prozent ist kein Platzhalter", []string{"abc", "a%c", "a%"}, "a%", []string{"a%", "a%c"}},
		{"Präfix mit Unterstrich im Schlüssel", []string{prefix + "A_B-USDT", "positionXhistory_until:Z"}, prefix, []string{prefix + "A_B-USDT"}},
		{"keine Treffer", []string{"x"}, "y", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewImportStateRepository(openMemory(t))
			ctx := context.Background()
			for _, key := range tc.keys {
				if err := repo.Set(ctx, key, 1); err != nil {
					t.Fatal(err)
				}
			}
			got, err := repo.ListKeys(ctx, tc.prefix)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestImportState_AbgebrochenerKontextLiefertFehler(t *testing.T) {
	repo := NewImportStateRepository(openMemory(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := repo.Get(ctx, "k"); err == nil {
		t.Error("Get: Fehler erwartet")
	}
	if err := repo.Set(ctx, "k", 1); err == nil {
		t.Error("Set: Fehler erwartet")
	}
	if err := repo.Delete(ctx, "k"); err == nil {
		t.Error("Delete: Fehler erwartet")
	}
	if _, err := repo.ListKeys(ctx, "k"); err == nil {
		t.Error("ListKeys: Fehler erwartet")
	}
}
