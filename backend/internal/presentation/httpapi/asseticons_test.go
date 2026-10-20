package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

// Erwartete API (siehe docs/openapi.yaml, /assets/icons):
// Settings.GetAssetIcons func(ctx, asset.SymbolList) ([]asset.CoinIcon, error); nil = Route nicht registriert.

type iconsCall struct {
	symbols []string
	count   int
}

func settingsWithIcons(t *testing.T, call *iconsCall, icons func(list asset.SymbolList) []asset.CoinIcon, failure error) Settings {
	t.Helper()
	settings := testSettings
	settings.GetAssetIcons = func(_ context.Context, list asset.SymbolList) ([]asset.CoinIcon, error) {
		call.count++
		call.symbols = list.Symbols()
		if failure != nil {
			return nil, failure
		}
		return icons(list), nil
	}
	return settings
}

func coinIcon(t *testing.T, symbol, rawURL string) asset.CoinIcon {
	t.Helper()
	base, err := asset.NewBaseAsset(symbol)
	if err != nil {
		t.Fatal(err)
	}
	icon := asset.CoinIcon{Symbol: symbol, BaseAsset: base, DisplayName: asset.DisplayName(base)}
	if rawURL != "" {
		parsed, err := asset.NewIconURL(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		icon.IconURL = &parsed
	}
	return icon
}

func TestAssetIcons_200MitGenauDenSpecFeldern(t *testing.T) {
	call := &iconsCall{}
	settings := settingsWithIcons(t, call, func(asset.SymbolList) []asset.CoinIcon {
		return []asset.CoinIcon{
			coinIcon(t, "SUI-USDT", "https://coin-images.coingecko.com/coins/images/26375/large/sui.png"),
			coinIcon(t, "XAU-USDT", ""),
		}
	}, nil)

	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=SUI-USDT,XAU-USDT,SUI-USDT", allowedTestHost))

	if recorder.Code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q", contentType)
	}
	if strings.Join(call.symbols, ",") != "SUI-USDT,XAU-USDT" {
		t.Errorf("Use Case bekam %v, erwartet deduplizierte Liste", call.symbols)
	}
	var body struct {
		Icons []map[string]any `json:"icons"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Icons) != 2 {
		t.Fatalf("icons = %v", body.Icons)
	}
	for index, entry := range body.Icons {
		if len(entry) != 4 {
			t.Errorf("Eintrag %d hat Felder %v, erwartet genau symbol, baseAsset, displayName, iconUrl", index, entry)
		}
	}
	first, second := body.Icons[0], body.Icons[1]
	if first["symbol"] != "SUI-USDT" || first["baseAsset"] != "SUI" ||
		first["displayName"] != "SUI" ||
		first["iconUrl"] != "https://coin-images.coingecko.com/coins/images/26375/large/sui.png" {
		t.Errorf("Eintrag 0 = %v", first)
	}
	if value, present := second["iconUrl"]; !present || value != nil {
		t.Errorf("Eintrag 1 iconUrl = %v (vorhanden=%v), erwartet null", value, present)
	}
}

func TestAssetIcons_RohstoffKontraktMitDisplayNameOhneIcon(t *testing.T) {
	call := &iconsCall{}
	settings := settingsWithIcons(t, call, func(asset.SymbolList) []asset.CoinIcon {
		return []asset.CoinIcon{coinIcon(t, "NCCOGOLD2USD-USDT", "")}
	}, nil)

	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=NCCOGOLD2USD-USDT", allowedTestHost))

	var body struct {
		Icons []map[string]any `json:"icons"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || len(body.Icons) != 1 {
		t.Fatalf("Body = %s, Fehler = %v", recorder.Body.String(), err)
	}
	entry := body.Icons[0]
	value, present := entry["iconUrl"]
	if entry["symbol"] != "NCCOGOLD2USD-USDT" || entry["baseAsset"] != "XAU" || entry["displayName"] != "Gold" || !present || value != nil {
		t.Errorf("Eintrag = %v", entry)
	}
}

func TestAssetIcons_QuellenfehlerImUseCaseErgibtWeiterhin200MitNull(t *testing.T) {
	// Der Use Case schluckt Quellenfehler selbst und liefert Einträge ohne Icon.
	call := &iconsCall{}
	settings := settingsWithIcons(t, call, func(_ asset.SymbolList) []asset.CoinIcon {
		return []asset.CoinIcon{coinIcon(t, "BTC-USDT", "")}
	}, nil)

	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=BTC-USDT", allowedTestHost))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"iconUrl":null`) {
		t.Errorf("Status = %d, Body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAssetIcons_UnerwarteterFehlerAusUseCaseIst500OhneDetails(t *testing.T) {
	call := &iconsCall{}
	settings := settingsWithIcons(t, call, nil, errors.New("geheimes internes Detail"))

	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=BTC-USDT", allowedTestHost))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, erwartet 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "geheimes") || !strings.Contains(recorder.Body.String(), `"internal"`) {
		t.Errorf("Body = %s", recorder.Body.String())
	}
}

func TestAssetIcons_400BeiUngueltigemSymbols(t *testing.T) {
	tooMany := make([]string, 51)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("C%d-USDT", index)
	}
	tests := map[string]string{
		"Parameter fehlt": "/assets/icons",
		"leer":            "/assets/icons?symbols=",
		"Kleinbuchstaben": "/assets/icons?symbols=btc-usdt",
		"leerer Eintrag":  "/assets/icons?symbols=BTC-USDT,,SUI-USDT",
		"Sonderzeichen":   "/assets/icons?symbols=" + url.QueryEscape("BTC/USDT"),
		"mehr als 50":     "/assets/icons?symbols=" + strings.Join(tooMany, ","),
	}
	for name, target := range tests {
		t.Run(name, func(t *testing.T) {
			call := &iconsCall{}
			settings := settingsWithIcons(t, call, func(asset.SymbolList) []asset.CoinIcon { return nil }, nil)

			recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, target, allowedTestHost))

			assertProblem(t, recorder, http.StatusBadRequest, badRequestBody)
			if call.count != 0 {
				t.Error("Use Case darf bei 400 nicht laufen")
			}
		})
	}
}

func TestAssetIcons_NurGetErlaubt(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			call := &iconsCall{}
			settings := settingsWithIcons(t, call, func(asset.SymbolList) []asset.CoinIcon { return nil }, nil)

			recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(method, "/assets/icons?symbols=BTC-USDT", allowedTestHost))

			if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), `"not-found"`) {
				t.Errorf("Status = %d, Body = %s, erwartet 404 not-found", recorder.Code, recorder.Body.String())
			}
			if call.count != 0 {
				t.Error("Use Case darf nicht laufen")
			}
		})
	}
}

func TestAssetIcons_RouteNurMitGesetztemUseCaseRegistriert(t *testing.T) {
	recorder := serve(NewHandler(testSettings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=BTC-USDT", allowedTestHost))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404 ohne GetAssetIcons", recorder.Code)
	}
}

func TestAssetIcons_HostGuardGiltAuch(t *testing.T) {
	call := &iconsCall{}
	settings := settingsWithIcons(t, call, func(asset.SymbolList) []asset.CoinIcon { return nil }, nil)
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/assets/icons?symbols=BTC-USDT", "evil.example:3000"))
	if recorder.Code != http.StatusForbidden || call.count != 0 {
		t.Errorf("Status = %d, Aufrufe = %d", recorder.Code, call.count)
	}
}
