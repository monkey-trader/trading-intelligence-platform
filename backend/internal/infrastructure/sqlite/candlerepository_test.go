package sqlite

import (
	"context"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart"
	chartinterfaces "github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/chart/interfaces"
)

var _ chartinterfaces.CandleRepository = (*CandleRepository)(nil)

func sampleCandle(symbol string, interval chart.CandleInterval, openTime int64, closePrice string) chart.Candle {
	return chart.Candle{Symbol: symbol, Interval: interval, OpenTime: openTime,
		Open: "83557.12345678", High: "83570.7", Low: "83464.8", Close: closePrice, Volume: "7.5886"}
}

func countCandles(t *testing.T, repository *CandleRepository) int {
	t.Helper()
	var count int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM candles`).Scan(&count); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	return count
}

func TestCandleRepository_SpeichertExaktUndLiestAufsteigend(t *testing.T) {
	repository := NewCandleRepository(openMemory(t))
	ctx := context.Background()
	// Absichtlich unsortiert eingefügt.
	input := []chart.Candle{
		sampleCandle("BTC-USDT", chart.Interval15m, 3000, "3"),
		sampleCandle("BTC-USDT", chart.Interval15m, 1000, "1"),
		sampleCandle("BTC-USDT", chart.Interval15m, 2000, "2"),
	}
	if err := repository.UpsertCandles(ctx, input); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	candles, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval15m, 10)
	if err != nil || len(candles) != 3 {
		t.Fatalf("candles = %+v, err = %v", candles, err)
	}
	for index, expectedTime := range []int64{1000, 2000, 3000} {
		if candles[index].OpenTime != expectedTime {
			t.Errorf("Index %d: OpenTime = %d, erwartet %d", index, candles[index].OpenTime, expectedTime)
		}
	}
	if candles[0] != sampleCandle("BTC-USDT", chart.Interval15m, 1000, "1") {
		t.Errorf("Dezimalwerte nicht exakt als Text erhalten: %+v", candles[0])
	}
}

func TestCandleRepository_LimitLiefertDieNeuestenKerzenAufsteigend(t *testing.T) {
	repository := NewCandleRepository(openMemory(t))
	ctx := context.Background()
	var input []chart.Candle
	for openTime := int64(1000); openTime <= 5000; openTime += 1000 {
		input = append(input, sampleCandle("BTC-USDT", chart.Interval15m, openTime, "1"))
	}
	if err := repository.UpsertCandles(ctx, input); err != nil {
		t.Fatal(err)
	}

	candles, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval15m, 2)

	if err != nil || len(candles) != 2 || candles[0].OpenTime != 4000 || candles[1].OpenTime != 5000 {
		t.Errorf("candles = %+v, err = %v", candles, err)
	}
	exact, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval15m, 5)
	if err != nil || len(exact) != 5 {
		t.Errorf("limit == Anzahl: %d Kerzen, err = %v", len(exact), err)
	}
}

func TestCandleRepository_UpsertAktualisiertLaufendeKerzeOhneDuplikat(t *testing.T) {
	repository := NewCandleRepository(openMemory(t))
	ctx := context.Background()
	if err := repository.UpsertCandles(ctx, []chart.Candle{sampleCandle("BTC-USDT", chart.Interval15m, 1000, "1.0")}); err != nil {
		t.Fatal(err)
	}

	updated := sampleCandle("BTC-USDT", chart.Interval15m, 1000, "9.9")
	updated.High = "99"
	if err := repository.UpsertCandles(ctx, []chart.Candle{updated}); err != nil {
		t.Fatal(err)
	}

	candles, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval15m, 10)
	if err != nil || len(candles) != 1 || candles[0] != updated || countCandles(t, repository) != 1 {
		t.Errorf("candles = %+v, err = %v", candles, err)
	}
}

func TestCandleRepository_SchluesselIstSymbolIntervallOpenTime(t *testing.T) {
	repository := NewCandleRepository(openMemory(t))
	ctx := context.Background()
	input := []chart.Candle{
		sampleCandle("BTC-USDT", chart.Interval15m, 1000, "a"),
		sampleCandle("BTC-USDT", chart.Interval1h, 1000, "b"),
		sampleCandle("SUI-USDT", chart.Interval15m, 1000, "c"),
	}
	if err := repository.UpsertCandles(ctx, input); err != nil {
		t.Fatal(err)
	}
	if count := countCandles(t, repository); count != 3 {
		t.Fatalf("erwartet 3 getrennte Zeilen, erhalten %d", count)
	}

	candles, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval1h, 10)
	if err != nil || len(candles) != 1 || candles[0].Close != "b" {
		t.Errorf("candles = %+v, err = %v", candles, err)
	}
}

func TestCandleRepository_LeerUndLeereEingabe(t *testing.T) {
	repository := NewCandleRepository(openMemory(t))
	ctx := context.Background()

	candles, err := repository.ListCandles(ctx, "BTC-USDT", chart.Interval15m, 10)
	if err != nil || candles == nil || len(candles) != 0 {
		t.Errorf("erwartet leere nicht-nil Liste: %v, %v", candles, err)
	}
	if err := repository.UpsertCandles(ctx, nil); err != nil {
		t.Errorf("leerer Upsert darf nicht fehlschlagen: %v", err)
	}
}

func TestCandleRepository_Schema(t *testing.T) {
	db := openMemory(t)
	rows, err := db.Query(`SELECT name FROM pragma_table_info('candles') ORDER BY cid`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	for _, required := range []string{"symbol", "interval", "open_time", "open", "high", "low", "close", "volume"} {
		found := false
		for _, column := range columns {
			found = found || column == required
		}
		if !found {
			t.Errorf("Spalte %q fehlt in candles (vorhanden: %v)", required, columns)
		}
	}
}
