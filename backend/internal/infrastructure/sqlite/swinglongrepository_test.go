package sqlite

import (
	"context"
	"slices"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

var _ interfaces.SwingLongRepository = (*SwingLongRepository)(nil)

func TestSwingLong_SchemaVersionIst2(t *testing.T) {
	if CurrentSchemaVersion != 2 {
		t.Fatalf("CurrentSchemaVersion = %d, erwartet 2 (Tabelle position_swing_long, additiv)", CurrentSchemaVersion)
	}
	db := openMemory(t)
	if got := schemaVersion(t, db); got != 2 {
		t.Errorf("schema_version = %d, erwartet 2", got)
	}
}

func TestSwingLong_UnmarkierteIstNichtGesetzt(t *testing.T) {
	repo := NewSwingLongRepository(openMemory(t))
	isSet, err := repo.IsSet(context.Background(), "1")
	if err != nil || isSet {
		t.Fatalf("isSet=%v err=%v", isSet, err)
	}
}

func TestSwingLong_SetIdempotentMit19stelligerId(t *testing.T) {
	repo := NewSwingLongRepository(openMemory(t))
	ctx := context.Background()
	const positionID = "9100000000000000001"
	for range 2 {
		if err := repo.Set(ctx, positionID, 1_791_800_000_000); err != nil {
			t.Fatal(err)
		}
	}
	if isSet, err := repo.IsSet(ctx, positionID); err != nil || !isSet {
		t.Fatalf("isSet=%v err=%v", isSet, err)
	}
	ids, err := repo.ListIDs(ctx)
	if err != nil || !slices.Equal(ids, []string{positionID}) {
		t.Errorf("ids=%v err=%v", ids, err)
	}
}

func TestSwingLong_UnsetIdempotentAuchOhneFlag(t *testing.T) {
	repo := NewSwingLongRepository(openMemory(t))
	ctx := context.Background()
	if err := repo.Unset(ctx, "1"); err != nil {
		t.Fatalf("Unset ohne Flag: %v", err)
	}
	_ = repo.Set(ctx, "1", 100)
	_ = repo.Set(ctx, "2", 100)
	for range 2 {
		if err := repo.Unset(ctx, "1"); err != nil {
			t.Fatal(err)
		}
	}
	if isSet, _ := repo.IsSet(ctx, "1"); isSet {
		t.Error("Flag 1 noch gesetzt")
	}
	if ids, _ := repo.ListIDs(ctx); !slices.Equal(ids, []string{"2"}) {
		t.Errorf("ids=%v, Flag 2 muss bleiben", ids)
	}
}

func TestSwingLong_ListIDsLeerIstLeereListe(t *testing.T) {
	ids, err := NewSwingLongRepository(openMemory(t)).ListIDs(context.Background())
	if err != nil || len(ids) != 0 {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}
