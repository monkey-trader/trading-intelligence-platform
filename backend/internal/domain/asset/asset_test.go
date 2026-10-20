package asset_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

func TestNewBaseAsset(t *testing.T) {
	tests := []struct {
		name, symbol, want string
	}{
		{"BingX-Symbol", "SUI-USDT", "SUI"},
		{"Kleinbuchstaben", "sui-usdt", "SUI"},
		{"Ziffern-Präfix bleibt", "1000PEPE-USDT", "1000PEPE"},
		{"mehrere Bindestriche: nur erster Teil", "BTC-USDT-PERP", "BTC"},
		{"ohne Bindestrich", "ETH", "ETH"},
		{"Rohstoff Gold wird XAU", "NCCOGOLD2USD-USDT", "XAU"},
		{"Rohstoff Silber bleibt XAG", "NCCOXAG2USD-USDT", "XAG"},
		{"Rohstoff Nickel", "NCCONICKEL2USD-USDT", "NICKEL"},
		{"Rohstoff Kleinbuchstaben", "nccogold2usd-usdt", "XAU"},
		{"normales XAU", "XAU-USDT", "XAU"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := asset.NewBaseAsset(testCase.symbol)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if got.String() != testCase.want {
				t.Errorf("BaseAsset = %q, erwartet %q", got.String(), testCase.want)
			}
		})
	}
}

func TestNewBaseAsset_Fehler(t *testing.T) {
	for name, symbol := range map[string]string{
		"leer": "", "nur Bindestrich": "-USDT", "Leerzeichen": "  ", "Sonderzeichen": "S$I-USDT",
		"zu lang":         strings.Repeat("A", 21) + "-USDT",
		"NCCO ohne Namen": "NCCO2USD-USDT",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := asset.NewBaseAsset(symbol)
			var invalid *asset.InvalidBaseAssetError
			if !errors.As(err, &invalid) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidBaseAssetError", err, err)
			}
		})
	}
}

func TestIsMetal(t *testing.T) {
	for symbol, want := range map[string]bool{"XAU-USDT": true, "XAG-USDT": true, "xau-usdt": true, "XAUT-USDT": false, "BTC-USDT": false} {
		base, err := asset.NewBaseAsset(symbol)
		if err != nil {
			t.Fatal(err)
		}
		if got := asset.IsMetal(base); got != want {
			t.Errorf("IsMetal(%s) = %v, erwartet %v", symbol, got, want)
		}
	}
}

func TestNewIconURL(t *testing.T) {
	valid := "https://coin-images.coingecko.com/coins/images/26375/large/sui-ocean-square.png?1727791290"
	got, err := asset.NewIconURL(valid)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if got.String() != valid {
		t.Errorf("IconURL = %q, erwartet %q", got.String(), valid)
	}
}

func TestNewIconURL_Fehler(t *testing.T) {
	for name, raw := range map[string]string{
		"leer":                        "",
		"http":                        "http://coin-images.coingecko.com/a.png",
		"anderer Host":                "https://evil.example.com/a.png",
		"Host als Präfix":             "https://coin-images.coingecko.com.evil.example/a.png",
		"Userinfo-Trick":              "https://coin-images.coingecko.com@evil.example/a.png",
		"anderes Schema":              "javascript:alert(1)",
		"kein URL":                    "%%%",
		"Subdomain fremd":             "https://x.coin-images.coingecko.com/a.png",
		"gleicher Host http mit Port": "https://coin-images.coingecko.com:8443/a.png",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := asset.NewIconURL(raw)
			var invalid *asset.InvalidIconURLError
			if !errors.As(err, &invalid) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidIconURLError", err, err)
			}
		})
	}
}

func TestParseSymbolList(t *testing.T) {
	tests := []struct {
		name, raw string
		want      []string
	}{
		{"ein Symbol", "BTC-USDT", []string{"BTC-USDT"}},
		{"Reihenfolge stabil", "SUI-USDT,BTC-USDT,ENA-USDT", []string{"SUI-USDT", "BTC-USDT", "ENA-USDT"}},
		{"Duplikate entfernt, erste Position bleibt", "SUI-USDT,BTC-USDT,SUI-USDT", []string{"SUI-USDT", "BTC-USDT"}},
		{"Ziffern", "1000PEPE-USDT", []string{"1000PEPE-USDT"}},
		{"ohne Bindestrich", "BTC", []string{"BTC"}},
		{"mehrere Bindestriche", "NCCOGOLD2USD-USDT", []string{"NCCOGOLD2USD-USDT"}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			list, err := asset.ParseSymbolList(testCase.raw)
			if err != nil {
				t.Fatalf("unerwarteter Fehler: %v", err)
			}
			if got := list.Symbols(); !slices.Equal(got, testCase.want) {
				t.Errorf("Symbols = %v, erwartet %v", got, testCase.want)
			}
		})
	}
}

func symbolsOf(count int) string {
	parts := make([]string, count)
	for index := range parts {
		parts[index] = fmt.Sprintf("C%d-USDT", index)
	}
	return strings.Join(parts, ",")
}

func TestParseSymbolList_Grenzen(t *testing.T) {
	if list, err := asset.ParseSymbolList(symbolsOf(50)); err != nil || len(list.Symbols()) != 50 {
		t.Fatalf("genau 50 muss gehen, Fehler = %v", err)
	}
	// 50 eindeutige + Duplikat: zählt nach Deduplizierung 50, trotzdem erlaubt.
	if _, err := asset.ParseSymbolList(symbolsOf(50) + ",C0-USDT"); err != nil {
		t.Errorf("50 eindeutige plus Duplikat muss gehen: %v", err)
	}
	if _, err := asset.ParseSymbolList(symbolsOf(51)); err == nil {
		t.Error("51 eindeutige Symbole müssen scheitern")
	}
	if _, err := asset.ParseSymbolList(strings.Repeat("A", 20) + "-USDT"); err != nil {
		t.Errorf("Basis mit 20 Zeichen muss gehen: %v", err)
	}
}

func TestParseSymbolList_Fehler(t *testing.T) {
	for name, raw := range map[string]string{
		"leer": "", "nur Komma": ",", "leerer Eintrag": "BTC-USDT,,SUI-USDT", "Endkomma": "BTC-USDT,",
		"Kleinbuchstaben": "btc-usdt", "Leerzeichen": "BTC-USDT, SUI-USDT", "Sonderzeichen": "BTC/USDT",
		"zu lang": strings.Repeat("A", 21) + "-USDT", "Bindestrich am Anfang": "-USDT", "Bindestrich am Ende": "BTC-",
		"51 Symbole": symbolsOf(51),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := asset.ParseSymbolList(raw)
			var invalid *asset.InvalidSymbolListError
			if !errors.As(err, &invalid) {
				t.Fatalf("Fehler = %v (%T), erwartet *InvalidSymbolListError", err, err)
			}
		})
	}
}

func TestIsCommodity(t *testing.T) {
	for symbol, want := range map[string]bool{
		"NCCOGOLD2USD-USDT": true, "NCCOXAG2USD-USDT": true, "NCCONICKEL2USD-USDT": true,
		"XAU-USDT": true, "XAG-USDT": true, "BTC-USDT": false, "XAUT-USDT": false, "1000PEPE-USDT": false,
	} {
		t.Run(symbol, func(t *testing.T) {
			base, err := asset.NewBaseAsset(symbol)
			if err != nil {
				t.Fatal(err)
			}
			if got := asset.IsCommodity(base); got != want {
				t.Errorf("IsCommodity(%s) = %v, erwartet %v", symbol, got, want)
			}
		})
	}
}

func TestIsMetal_RohstoffKontrakte(t *testing.T) {
	for symbol, want := range map[string]bool{"NCCOGOLD2USD-USDT": true, "NCCOXAG2USD-USDT": true, "NCCONICKEL2USD-USDT": false} {
		base, err := asset.NewBaseAsset(symbol)
		if err != nil {
			t.Fatal(err)
		}
		if got := asset.IsMetal(base); got != want {
			t.Errorf("IsMetal(%s) = %v, erwartet %v", symbol, got, want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	tests := map[string]string{
		"XAU-USDT": "Gold", "XAG-USDT": "Silber",
		"NCCOGOLD2USD-USDT": "Gold", "NCCOXAG2USD-USDT": "Silber", "NCCONICKEL2USD-USDT": "Nickel",
		"ONDO-USDT": "ONDO", "BTC-USDT": "BTC", "btc-usdt": "BTC", "1000PEPE-USDT": "1000PEPE",
	}
	for symbol, want := range tests {
		t.Run(symbol, func(t *testing.T) {
			base, err := asset.NewBaseAsset(symbol)
			if err != nil {
				t.Fatal(err)
			}
			if got := asset.DisplayName(base); got != want {
				t.Errorf("DisplayName(%s) = %q, erwartet %q", symbol, got, want)
			}
		})
	}
}
