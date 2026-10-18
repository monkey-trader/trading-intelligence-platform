package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position"
	"github.com/monkey-trader/trading-intelligence-platform/backend-go/internal/domain/position/interfaces"
)

var (
	_ interfaces.IncomeRepository = (*IncomeRepository)(nil)
	_ interfaces.IncomeLookup     = (*IncomeLookup)(nil)
)

func sampleIncome(tranID, symbol string, at int64) position.Income {
	return position.Income{TranID: tranID, Symbol: symbol, IncomeType: "REALIZED_PNL", Amount: "1.70000000000000001",
		Time: at, Asset: "USDT", TradeID: "7000000000000000002", Raw: `{"tranId":"` + tranID + `"}`}
}

func TestIncomeRepository_SpeichertExaktMitTypen(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewIncomeRepository(db, fixedClock(&now))
	input := sampleIncome("8000000000000000002", "ONDO-USDT", 1_780_018_017_000)
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{input}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	var tranID, symbol, incomeType, amount, asset, tradeID, raw string
	var at, importedAt int64
	err := db.QueryRow(`SELECT transaction_id, symbol, income_type, amount, time, asset, trade_id, raw, imported_at FROM income`).
		Scan(&tranID, &symbol, &incomeType, &amount, &at, &asset, &tradeID, &raw, &importedAt)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if tranID != input.TranID || symbol != input.Symbol || incomeType != input.IncomeType || amount != input.Amount ||
		at != input.Time || asset != input.Asset || tradeID != input.TradeID || raw != input.Raw || importedAt != 1_800_000_000_000 {
		t.Errorf("gespeichert falsch: %s %s %s %s %d %s %s %s %d", tranID, symbol, incomeType, amount, at, asset, tradeID, raw, importedAt)
	}
	var idType, amountType string
	_ = db.QueryRow(`SELECT typeof(transaction_id), typeof(amount) FROM income`).Scan(&idType, &amountType)
	if idType != "text" || amountType != "text" {
		t.Errorf("Typen id=%s amount=%s, beide text", idType, amountType)
	}
}

func TestIncomeRepository_DedupUeberTranId(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1_800_000_000_000)
	repo := NewIncomeRepository(db, fixedClock(&now))
	first := sampleIncome("1", "ONDO-USDT", 1000)
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{first, sampleIncome("2", "ONDO-USDT", 2000)}); err != nil {
		t.Fatal(err)
	}
	changed := first
	changed.Amount = "9.5"
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{changed, sampleIncome("2", "ONDO-USDT", 2000)}); err != nil {
		t.Fatalf("zweiter Upsert (Überlapp): %v", err)
	}
	var rows int
	var amount string
	_ = db.QueryRow(`SELECT COUNT(*) FROM income`).Scan(&rows)
	_ = db.QueryRow(`SELECT amount FROM income WHERE transaction_id = '1'`).Scan(&amount)
	if rows != 2 || amount != "9.5" {
		t.Errorf("Zeilen=%d amount=%s, erwartet 2 und 9.5", rows, amount)
	}
}

func TestIncomeRepository_LeererBatchIstOk(t *testing.T) {
	now := time.UnixMilli(1)
	if err := NewIncomeRepository(openMemory(t), fixedClock(&now)).UpsertIncomeBatch(context.Background(), nil); err != nil {
		t.Errorf("Fehler: %v", err)
	}
}

func TestIncomeRepository_BatchIstAtomar(t *testing.T) {
	db := openMemory(t)
	if _, err := db.Exec(`CREATE TRIGGER fail_bad BEFORE INSERT ON income WHEN NEW.transaction_id = 'bad' BEGIN SELECT RAISE(ABORT, 'bad'); END`); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	now := time.UnixMilli(1)
	repo := NewIncomeRepository(db, fixedClock(&now))
	err := repo.UpsertIncomeBatch(context.Background(), []position.Income{sampleIncome("1", "ONDO-USDT", 1000), sampleIncome("bad", "ONDO-USDT", 2000)})
	if err == nil {
		t.Fatal("Fehler erwartet")
	}
	var rows int
	_ = db.QueryRow(`SELECT COUNT(*) FROM income`).Scan(&rows)
	if rows != 0 {
		t.Errorf("%d Zeilen übrig, Rollback erwartet", rows)
	}
}

func TestIncomeRepository_AbgebrochenerKontext(t *testing.T) {
	now := time.UnixMilli(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewIncomeRepository(openMemory(t), fixedClock(&now)).UpsertIncomeBatch(ctx, []position.Income{sampleIncome("1", "ONDO-USDT", 1000)})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err=%v", err)
	}
}

func TestIncomeSchema_IndexSymbolTime(t *testing.T) {
	db := openMemory(t)
	var columns string
	rows, err := db.Query(`SELECT name FROM pragma_index_info((SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='income' AND name NOT LIKE 'sqlite_autoindex%'))`)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		columns += name + ","
	}
	if columns != "symbol,time," {
		t.Errorf("Index-Spalten %q, erwartet symbol,time,", columns)
	}
}

func TestIncomeLookup(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1)
	repo := NewIncomeRepository(db, fixedClock(&now))
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{
		sampleIncome("1", "SUI-USDT", 1000), sampleIncome("2", "1000SUI-USDT", 5000),
	}); err != nil {
		t.Fatal(err)
	}
	lookup := NewIncomeLookup(db)
	tests := []struct {
		name     string
		symbol   string
		from, to int64
		want     bool
	}{
		{"Treffer", "SUI-USDT", 0, 2000, true},
		{"untere Grenze inklusive", "SUI-USDT", 1000, 2000, true},
		{"obere Grenze inklusive", "SUI-USDT", 0, 1000, true},
		{"1 ms vor dem Eintrag", "SUI-USDT", 0, 999, false},
		{"1 ms nach dem Eintrag", "SUI-USDT", 1001, 2000, false},
		{"Symbol exakt, kein Suffix-Treffer", "SUI-USDT", 4000, 6000, false},
		{"anderes Symbol", "XRP-USDT", 0, 9000, false},
		{"Symbol mit Präfix existiert", "1000SUI-USDT", 4000, 6000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lookup.HasIncomeForSymbol(context.Background(), tt.symbol, tt.from, tt.to)
			if err != nil || got != tt.want {
				t.Errorf("got=%v err=%v, erwartet %v", got, err, tt.want)
			}
		})
	}
}

func TestIncomeLookup_LeereTabelleUndFehler(t *testing.T) {
	db := openMemory(t)
	lookup := NewIncomeLookup(db)
	if got, err := lookup.HasIncomeForSymbol(context.Background(), "SUI-USDT", 0, 9); err != nil || got {
		t.Errorf("leer: got=%v err=%v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := lookup.HasIncomeForSymbol(ctx, "SUI-USDT", 0, 9); !errors.Is(err, context.Canceled) {
		t.Errorf("err=%v", err)
	}
}

var _ interfaces.IncomeSymbolLister = (*IncomeRepository)(nil)

func TestListIncomeSymbols(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1)
	repo := NewIncomeRepository(db, fixedClock(&now))
	ctx := context.Background()

	empty, err := repo.ListIncomeSymbols(ctx)
	if err != nil || len(empty) != 0 {
		t.Fatalf("leere Tabelle: %v, %v", empty, err)
	}
	funding := sampleIncome("4", "FUND-USDT", 4000)
	funding.IncomeType = "FUNDING_FEE"
	entries := []position.Income{
		sampleIncome("1", "ONDO-USDT", 1000), sampleIncome("2", "BTC-USDT", 2000),
		sampleIncome("3", "ONDO-USDT", 3000), funding,
	}
	if err := repo.UpsertIncomeBatch(ctx, entries); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListIncomeSymbols(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// nur REALIZED_PNL, eindeutig, sortiert
	if len(got) != 2 || got[0] != "BTC-USDT" || got[1] != "ONDO-USDT" {
		t.Errorf("Symbole = %v, erwartet [BTC-USDT ONDO-USDT]", got)
	}
}

func TestIncomeLookup_NurRealizedPnlGiltAlsBeleg(t *testing.T) {
	db := openMemory(t)
	now := time.UnixMilli(1)
	repo := NewIncomeRepository(db, fixedClock(&now))
	fee := sampleIncome("1", "ALGO-USDT", 1000)
	fee.IncomeType = "TRADING_FEE"
	funding := sampleIncome("2", "ALGO-USDT", 2000)
	funding.IncomeType = "FUNDING_FEE"
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{fee, funding}); err != nil {
		t.Fatal(err)
	}
	lookup := NewIncomeLookup(db)
	if got, err := lookup.HasIncomeForSymbol(context.Background(), "ALGO-USDT", 0, 9000); err != nil || got {
		t.Errorf("Gebühren und Funding offener Positionen sind kein Beleg: got=%v err=%v", got, err)
	}
	if err := repo.UpsertIncomeBatch(context.Background(), []position.Income{sampleIncome("3", "ALGO-USDT", 3000)}); err != nil {
		t.Fatal(err)
	}
	if got, err := lookup.HasIncomeForSymbol(context.Background(), "ALGO-USDT", 0, 9000); err != nil || !got {
		t.Errorf("REALIZED_PNL ist ein Beleg: got=%v err=%v", got, err)
	}
}
