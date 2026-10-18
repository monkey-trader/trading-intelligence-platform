package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position/interfaces"
)

var _ interfaces.PositionRepository = (*PositionRepository)(nil)

func samplePosition() position.ClosedPosition {
	return position.ClosedPosition{
		PositionID: "1234567890123456789", Symbol: "SUI-USDT", Side: position.Long, Isolated: true,
		OpenTime: 1_700_000_000_000, CloseTime: 1_700_000_100_000,
		AvgOpenPrice: "1.23456789012345678", AvgClosePrice: "1.30000000", RealisedProfit: "12.50000000",
		NetProfit: "11.99999999", PositionAmount: "100.5", ClosedAmount: "100.5", Leverage: 7.5,
		Commission: "-0.10", TotalFunding: "0.0001", FullyClosed: true, Raw: `{"positionId":"1234567890123456789"}`,
	}
}

type storedRow struct {
	positionID, symbol, side, avgOpen, avgClose, realised, net, amount, closedAmount, commission, funding, raw string
	isolated, fullyClosed                                                                                      int
	openTime, closeTime, importedAt                                                                            int64
	leverage                                                                                                   float64
}

func fixedClock(moment *time.Time) func() time.Time { return func() time.Time { return *moment } }

func TestUpsert_LegtAnUndSpeichertExakt(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	input := samplePosition()
	if err := repo.UpsertClosedPosition(context.Background(), input); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	var row storedRow
	err := db.QueryRow(`SELECT position_id, symbol, side, isolated, open_time, close_time, avg_open_price, avg_close_price,
		realised_profit, net_profit, position_amount, closed_amount, commission, total_funding, leverage, fully_closed, raw, imported_at
		FROM positions`).Scan(&row.positionID, &row.symbol, &row.side, &row.isolated, &row.openTime, &row.closeTime,
		&row.avgOpen, &row.avgClose, &row.realised, &row.net, &row.amount, &row.closedAmount, &row.commission,
		&row.funding, &row.leverage, &row.fullyClosed, &row.raw, &row.importedAt)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	want := storedRow{input.PositionID, input.Symbol, "LONG", input.AvgOpenPrice, input.AvgClosePrice, input.RealisedProfit,
		input.NetProfit, input.PositionAmount, input.ClosedAmount, input.Commission, input.TotalFunding, input.Raw,
		1, 1, input.OpenTime, input.CloseTime, 1_800_000_000_000, 7.5}
	if row != want {
		t.Errorf("gespeichert %+v, erwartet %+v", row, want)
	}
	var idType, priceType string
	_ = db.QueryRow(`SELECT typeof(position_id), typeof(avg_open_price) FROM positions`).Scan(&idType, &priceType)
	if idType != "text" || priceType != "text" {
		t.Errorf("Typen: id=%s preis=%s, beide müssen text sein", idType, priceType)
	}
}

func TestUpsert_ZweiterAufrufAktualisiertOhneDuplikat(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	ctx := context.Background()
	if err := repo.UpsertClosedPosition(ctx, samplePosition()); err != nil {
		t.Fatal(err)
	}
	changed := samplePosition()
	changed.NetProfit = "99.5"
	changed.CloseTime = 1_700_000_200_000
	changed.Raw = `{"neu":true}`
	changed.FullyClosed = false
	now = time.UnixMilli(1_900_000_000_000)
	if err := repo.UpsertClosedPosition(ctx, changed); err != nil {
		t.Fatal(err)
	}
	var count int
	var net, raw string
	var closeTime, importedAt int64
	var fully int
	_ = db.QueryRow(`SELECT COUNT(*) FROM positions`).Scan(&count)
	_ = db.QueryRow(`SELECT net_profit, raw, close_time, imported_at, fully_closed FROM positions`).Scan(&net, &raw, &closeTime, &importedAt, &fully)
	if count != 1 {
		t.Fatalf("Zeilen = %d, erwartet 1", count)
	}
	if net != "99.5" || raw != `{"neu":true}` || closeTime != 1_700_000_200_000 || fully != 0 {
		t.Errorf("Felder nicht aktualisiert: %s %s %d %d", net, raw, closeTime, fully)
	}
	if importedAt != 1_900_000_000_000 {
		t.Errorf("imported_at = %d, erwartet 1900000000000", importedAt)
	}
}

func TestUpsert_LaesstNotesUnberuehrt(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	ctx := context.Background()
	input := samplePosition()
	if err := repo.UpsertClosedPosition(ctx, input); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes (position_id, setup, rating, mistakes, notes, updated_at) VALUES (?, 'Breakout', 4, 'zu früh', 'Text', 42)`, input.PositionID); err != nil {
		t.Fatalf("notes einfügen: %v", err)
	}
	input.NetProfit = "5"
	if err := repo.UpsertClosedPosition(ctx, input); err != nil {
		t.Fatalf("zweiter Upsert mit notes-Zeile: %v", err)
	}
	var setup, mistakes, text string
	var rating int
	var updatedAt int64
	err := db.QueryRow(`SELECT setup, rating, mistakes, notes, updated_at FROM notes WHERE position_id = ?`, input.PositionID).
		Scan(&setup, &rating, &mistakes, &text, &updatedAt)
	if err != nil {
		t.Fatalf("notes-Zeile weg: %v", err)
	}
	if setup != "Breakout" || rating != 4 || mistakes != "zu früh" || text != "Text" || updatedAt != 42 {
		t.Errorf("notes verändert: %s %d %s %s %d", setup, rating, mistakes, text, updatedAt)
	}
}

func TestUpsert_AbgebrochenerKontextLiefertFehler(t *testing.T) {
	db := openMemory(t)
	repo := NewPositionRepository(db, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repo.UpsertClosedPosition(ctx, samplePosition()); err == nil {
		t.Fatal("Fehler erwartet")
	}
}

func TestUpsertBatch_SpeichertAlleUndIstIdempotent(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	first, second := samplePosition(), samplePosition()
	second.PositionID = "1234567890123456790"
	batch := []position.ClosedPosition{first, second, first} // Duplikat im Batch
	for run := 1; run <= 2; run++ {
		if err := repo.UpsertClosedPositionBatch(context.Background(), batch); err != nil {
			t.Fatalf("Lauf %d: %v", run, err)
		}
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM positions`).Scan(&count)
	if count != 2 {
		t.Errorf("Zeilen = %d, erwartet 2", count)
	}
}

func TestUpsertBatch_LeerIstOk(t *testing.T) {
	now := time.UnixMilli(1)
	repo := NewPositionRepository(openMemory(t), fixedClock(&now))
	if err := repo.UpsertClosedPositionBatch(context.Background(), nil); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestUpsertBatch_RollbackBeiFehler(t *testing.T) {
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TRIGGER fail_bad BEFORE INSERT ON positions WHEN NEW.position_id = '999' BEGIN SELECT RAISE(ABORT, 'bad'); END`); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(1)
	repo := NewPositionRepository(db, fixedClock(&now))
	bad := samplePosition()
	bad.PositionID = "999"
	err := repo.UpsertClosedPositionBatch(context.Background(), []position.ClosedPosition{samplePosition(), bad})
	if err == nil {
		t.Fatal("Fehler erwartet")
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM positions`).Scan(&count)
	if count != 0 {
		t.Errorf("%d Zeilen übrig, Rollback erwartet", count)
	}
}

func TestUpsertBatch_LaesstNotesUnberuehrt(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	input := samplePosition()
	if err := repo.UpsertClosedPositionBatch(context.Background(), []position.ClosedPosition{input}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes (position_id, setup, rating, mistakes, notes, updated_at) VALUES (?, 'Breakout', 4, 'zu früh', 'Text', 42)`, input.PositionID); err != nil {
		t.Fatal(err)
	}
	input.NetProfit = "5"
	if err := repo.UpsertClosedPositionBatch(context.Background(), []position.ClosedPosition{input}); err != nil {
		t.Fatal(err)
	}
	var setup, text string
	var rating, updatedAt int
	if err := db.QueryRow(`SELECT setup, rating, notes, updated_at FROM notes`).Scan(&setup, &rating, &text, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if setup != "Breakout" || rating != 4 || text != "Text" || updatedAt != 42 {
		t.Errorf("notes verändert: %s %d %s %d", setup, rating, text, updatedAt)
	}
}

func TestUpsertBatch_AbgebrochenerKontext(t *testing.T) {
	now := time.UnixMilli(1)
	repo := NewPositionRepository(openMemory(t), fixedClock(&now))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repo.UpsertClosedPositionBatch(ctx, []position.ClosedPosition{samplePosition()}); err == nil {
		t.Error("Fehler erwartet")
	}
}

var _ interfaces.OpenPositionRecorder = (*PositionRepository)(nil)

func sampleOpenPosition() position.OpenPosition {
	return position.OpenPosition{
		PositionID: "9876543210987654321", Symbol: "ONDO-USDT", Side: position.Short, Isolated: false,
		PositionAmount: "250.5", AvgOpenPrice: "0.98765432109876", Leverage: 5, OpenTime: 1_790_000_000_000,
	}
}

func TestRecordOpenPositions_LegtOffeneZeileOhneSchlusszeitAn(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	input := sampleOpenPosition()

	if err := repo.RecordOpenPositions(context.Background(), []position.OpenPosition{input}); err != nil {
		t.Fatal(err)
	}

	var symbol, side, avgOpen, amount, raw string
	var closeTime, openTime, importedAt int64
	var fullyClosed int
	err := db.QueryRow(`SELECT symbol, side, avg_open_price, position_amount, close_time, open_time, fully_closed, raw, imported_at
		FROM positions WHERE position_id = ?`, input.PositionID).
		Scan(&symbol, &side, &avgOpen, &amount, &closeTime, &openTime, &fullyClosed, &raw, &importedAt)
	if err != nil {
		t.Fatal(err)
	}
	if symbol != "ONDO-USDT" || side != "SHORT" || avgOpen != input.AvgOpenPrice || amount != "250.5" ||
		closeTime != 0 || openTime != input.OpenTime || fullyClosed != 0 || importedAt != 1_800_000_000_000 {
		t.Errorf("Zeile = %s %s %s %s close=%d open=%d fully=%d imported=%d", symbol, side, avgOpen, amount, closeTime, openTime, fullyClosed, importedAt)
	}
	if !strings.Contains(raw, input.PositionID) {
		t.Errorf("raw enthält die Position nicht: %s", raw)
	}
}

func TestRecordOpenPositions_UeberschreibtKeineBereitsGeschlosseneZeile(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	closed := samplePosition()
	if err := repo.UpsertClosedPosition(context.Background(), closed); err != nil {
		t.Fatal(err)
	}
	stillOpen := sampleOpenPosition()
	stillOpen.PositionID = closed.PositionID

	if err := repo.RecordOpenPositions(context.Background(), []position.OpenPosition{stillOpen}); err != nil {
		t.Fatal(err)
	}

	var closeTime int64
	_ = db.QueryRow(`SELECT close_time FROM positions WHERE position_id = ?`, closed.PositionID).Scan(&closeTime)
	if closeTime != closed.CloseTime {
		t.Errorf("close_time = %d, geschlossene Zeile darf nicht überschrieben werden", closeTime)
	}
}

func TestRecordOpenPositions_SchliessenAktualisiertDieselbeZeileUndBehaeltNotizen(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewPositionRepository(db, fixedClock(&now))
	open := sampleOpenPosition()
	if err := repo.RecordOpenPositions(context.Background(), []position.OpenPosition{open}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes (position_id, notes) VALUES (?, 'mein Setup')`, open.PositionID); err != nil {
		t.Fatal(err)
	}
	closed := samplePosition()
	closed.PositionID = open.PositionID

	if err := repo.UpsertClosedPosition(context.Background(), closed); err != nil {
		t.Fatal(err)
	}

	var rows, fullyClosed int
	_ = db.QueryRow(`SELECT COUNT(*), MAX(fully_closed) FROM positions`).Scan(&rows, &fullyClosed)
	var note string
	_ = db.QueryRow(`SELECT notes FROM notes WHERE position_id = ?`, open.PositionID).Scan(&note)
	if rows != 1 || fullyClosed != 1 || note != "mein Setup" {
		t.Errorf("rows=%d fullyClosed=%d note=%q", rows, fullyClosed, note)
	}
}
