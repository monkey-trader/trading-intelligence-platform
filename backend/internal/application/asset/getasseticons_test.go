package asset_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	appasset "github.com/monkey-trader/trading-intelligence-platform/backend/internal/application/asset"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/asset"
)

var testNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testNow }

// fakeSource zählt Aufrufe und merkt sich die angefragten BaseAssets.
type fakeSource struct {
	mutex sync.Mutex
	icons map[asset.BaseAsset]asset.IconURL
	err   error
	calls [][]asset.BaseAsset
}

func (source *fakeSource) ResolveIcons(_ context.Context, bases []asset.BaseAsset) (map[asset.BaseAsset]asset.IconURL, error) {
	source.mutex.Lock()
	defer source.mutex.Unlock()
	source.calls = append(source.calls, slices.Clone(bases))
	if source.err != nil {
		return nil, source.err
	}
	return source.icons, nil
}

type cacheEntry struct {
	icon *asset.IconURL
	ttl  time.Duration
}

// fakeCache speichert ohne Ablauf und merkt sich die TTLs.
type fakeCache struct {
	entries map[asset.BaseAsset]cacheEntry
}

func newFakeCache() *fakeCache { return &fakeCache{entries: map[asset.BaseAsset]cacheEntry{}} }

func (cache *fakeCache) Get(base asset.BaseAsset) (*asset.IconURL, bool) {
	entry, found := cache.entries[base]
	return entry.icon, found
}

func (cache *fakeCache) Put(base asset.BaseAsset, icon *asset.IconURL, ttl time.Duration) {
	cache.entries[base] = cacheEntry{icon: icon, ttl: ttl}
}

func mustBase(t *testing.T, symbol string) asset.BaseAsset {
	t.Helper()
	base, err := asset.NewBaseAsset(symbol)
	if err != nil {
		t.Fatal(err)
	}
	return base
}

func mustURL(t *testing.T, raw string) asset.IconURL {
	t.Helper()
	url, err := asset.NewIconURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	return url
}

func mustList(t *testing.T, raw string) asset.SymbolList {
	t.Helper()
	list, err := asset.ParseSymbolList(raw)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

const (
	suiURL = "https://coin-images.coingecko.com/coins/images/26375/large/sui.png"
	btcURL = "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png"
)

func TestExecute_LoestBatchMitEinemSourceAufrufAuf(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{
		mustBase(t, "SUI"): mustURL(t, suiURL), mustBase(t, "BTC"): mustURL(t, btcURL),
	}}
	cache := newFakeCache()
	useCase := appasset.NewGetAssetIconsUseCase(source, cache, fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "SUI-USDT,BTC-USDT,ENA-USDT"))
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(source.calls) != 1 {
		t.Fatalf("Source-Aufrufe = %d, erwartet genau 1 (Batch)", len(source.calls))
	}
	wantBatch := []asset.BaseAsset{mustBase(t, "SUI"), mustBase(t, "BTC"), mustBase(t, "ENA")}
	if !slices.Equal(source.calls[0], wantBatch) {
		t.Errorf("Batch = %v, erwartet %v", source.calls[0], wantBatch)
	}
	if len(icons) != 3 {
		t.Fatalf("Einträge = %d, erwartet 3", len(icons))
	}
	wantSymbols := []string{"SUI-USDT", "BTC-USDT", "ENA-USDT"}
	for index, icon := range icons {
		if icon.Symbol != wantSymbols[index] {
			t.Errorf("Eintrag %d: Symbol = %q, erwartet %q (Reihenfolge der Anfrage)", index, icon.Symbol, wantSymbols[index])
		}
	}
	if icons[0].IconURL == nil || icons[0].IconURL.String() != suiURL || icons[0].BaseAsset.String() != "SUI" {
		t.Errorf("SUI-Eintrag = %+v", icons[0])
	}
	if icons[2].IconURL != nil {
		t.Errorf("ENA unbekannt: IconURL muss nil sein, war %v", icons[2].IconURL)
	}
}

func TestExecute_CacheTTLs(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{mustBase(t, "SUI"): mustURL(t, suiURL)}}
	cache := newFakeCache()
	useCase := appasset.NewGetAssetIconsUseCase(source, cache, fixedClock{})

	if _, err := useCase.Execute(context.Background(), mustList(t, "SUI-USDT,ENA-USDT")); err != nil {
		t.Fatal(err)
	}

	if got := cache.entries[mustBase(t, "SUI")]; got.icon == nil || got.ttl != 24*time.Hour {
		t.Errorf("Treffer: %+v, erwartet Icon und TTL 24h", got)
	}
	got, found := cache.entries[mustBase(t, "ENA")]
	if !found || got.icon != nil || got.ttl != 10*time.Minute {
		t.Errorf("Unbekannt: %+v (gespeichert=%v), erwartet Negativ-Eintrag mit TTL 10min", got, found)
	}
}

func TestExecute_CacheTrefferOhneSourceAufruf(t *testing.T) {
	source := &fakeSource{}
	cache := newFakeCache()
	suiIcon := mustURL(t, suiURL)
	cache.Put(mustBase(t, "SUI"), &suiIcon, 24*time.Hour)
	cache.Put(mustBase(t, "ENA"), nil, 10*time.Minute)
	useCase := appasset.NewGetAssetIconsUseCase(source, cache, fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "SUI-USDT,ENA-USDT"))
	if err != nil {
		t.Fatal(err)
	}

	if len(source.calls) != 0 {
		t.Errorf("Source-Aufrufe = %d, erwartet 0", len(source.calls))
	}
	if icons[0].IconURL == nil || icons[0].IconURL.String() != suiURL || icons[1].IconURL != nil {
		t.Errorf("Ergebnis = %+v", icons)
	}
}

func TestExecute_FragtNurFehlendeBaseAssetsAn(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{mustBase(t, "BTC"): mustURL(t, btcURL)}}
	cache := newFakeCache()
	suiIcon := mustURL(t, suiURL)
	cache.Put(mustBase(t, "SUI"), &suiIcon, time.Hour)
	useCase := appasset.NewGetAssetIconsUseCase(source, cache, fixedClock{})

	if _, err := useCase.Execute(context.Background(), mustList(t, "SUI-USDT,BTC-USDT")); err != nil {
		t.Fatal(err)
	}

	if len(source.calls) != 1 || !slices.Equal(source.calls[0], []asset.BaseAsset{mustBase(t, "BTC")}) {
		t.Errorf("Source-Aufrufe = %v, erwartet [[BTC]]", source.calls)
	}
}

func TestExecute_MetalleWerdenNieAngefragt(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{
		mustBase(t, "XAU"): mustURL(t, suiURL), // Quelle würde (fälschlich) liefern
		mustBase(t, "BTC"): mustURL(t, btcURL),
	}}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "XAU-USDT,BTC-USDT,XAG-USDT"))
	if err != nil {
		t.Fatal(err)
	}

	if len(source.calls) != 1 || !slices.Equal(source.calls[0], []asset.BaseAsset{mustBase(t, "BTC")}) {
		t.Errorf("Batch = %v, erwartet [[BTC]]", source.calls)
	}
	if icons[0].IconURL != nil || icons[2].IconURL != nil {
		t.Errorf("Metalle müssen nil liefern: %+v", icons)
	}
	if icons[0].Symbol != "XAU-USDT" || icons[0].BaseAsset.String() != "XAU" || len(icons) != 3 {
		t.Errorf("Metall-Eintrag = %+v", icons[0])
	}
}

func TestExecute_NurMetalleRuftSourceNichtAuf(t *testing.T) {
	source := &fakeSource{}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})

	if _, err := useCase.Execute(context.Background(), mustList(t, "XAU-USDT")); err != nil {
		t.Fatal(err)
	}
	if len(source.calls) != 0 {
		t.Errorf("Source-Aufrufe = %d, erwartet 0", len(source.calls))
	}
}

func TestExecute_QuellenfehlerLiefertNilOhneFehlerUndOhneCache(t *testing.T) {
	for name, sourceErr := range map[string]error{
		"Netzfehler": errors.New("connection refused"),
		"Rate-Limit": errors.New("HTTP 429"),
	} {
		t.Run(name, func(t *testing.T) {
			source := &fakeSource{err: sourceErr}
			cache := newFakeCache()
			suiIcon := mustURL(t, suiURL)
			cache.Put(mustBase(t, "SUI"), &suiIcon, time.Hour)
			useCase := appasset.NewGetAssetIconsUseCase(source, cache, fixedClock{})

			icons, err := useCase.Execute(context.Background(), mustList(t, "SUI-USDT,BTC-USDT"))
			if err != nil {
				t.Fatalf("Quellenfehler darf nicht nach oben: %v", err)
			}
			if icons[0].IconURL == nil || icons[1].IconURL != nil || len(icons) != 2 {
				t.Errorf("Ergebnis = %+v, erwartet SUI aus Cache, BTC nil", icons)
			}
			if _, cached := cache.Get(mustBase(t, "BTC")); cached {
				t.Error("Quellenfehler darf nicht (auch nicht negativ) gecacht werden")
			}
		})
	}
}

func TestExecute_GleicheBaseAssetsLoesenEinmalAufJedesSymbolBekommtEintrag(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{mustBase(t, "BTC"): mustURL(t, btcURL)}}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "BTC-USDT,BTC-USDC,BTC-USDT"))
	if err != nil {
		t.Fatal(err)
	}

	if len(source.calls) != 1 || !slices.Equal(source.calls[0], []asset.BaseAsset{mustBase(t, "BTC")}) {
		t.Errorf("Batch = %v, erwartet [[BTC]] ohne Duplikat", source.calls)
	}
	if len(icons) != 2 || icons[0].Symbol != "BTC-USDT" || icons[1].Symbol != "BTC-USDC" {
		t.Fatalf("Einträge = %+v", icons)
	}
	for _, icon := range icons {
		if icon.IconURL == nil || icon.IconURL.String() != btcURL {
			t.Errorf("%s ohne Icon: %+v", icon.Symbol, icon)
		}
	}
}

func TestExecute_ZweiterAufrufNutztCache(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{mustBase(t, "SUI"): mustURL(t, suiURL)}}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})
	list := mustList(t, "SUI-USDT,ENA-USDT")

	for range 2 {
		if _, err := useCase.Execute(context.Background(), list); err != nil {
			t.Fatal(err)
		}
	}
	if len(source.calls) != 1 {
		t.Errorf("Source-Aufrufe = %d, erwartet 1 (Positiv und Negativ gecacht)", len(source.calls))
	}
}

func TestExecute_RohstoffeWerdenNieAngefragtAuchImGemischtenBatch(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{
		mustBase(t, "NCCONICKEL2USD-USDT"): mustURL(t, suiURL), // Quelle würde (fälschlich) liefern
		mustBase(t, "ONDO"):                mustURL(t, btcURL),
	}}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "NCCOGOLD2USD-USDT,ONDO-USDT,NCCONICKEL2USD-USDT"))
	if err != nil {
		t.Fatal(err)
	}

	if len(source.calls) != 1 || !slices.Equal(source.calls[0], []asset.BaseAsset{mustBase(t, "ONDO")}) {
		t.Errorf("Batch = %v, erwartet [[ONDO]]", source.calls)
	}
	if len(icons) != 3 || icons[0].IconURL != nil || icons[2].IconURL != nil || icons[1].IconURL == nil {
		t.Fatalf("Ergebnis = %+v", icons)
	}
}

func TestExecute_NurRohstoffeRuftSourceNichtAuf(t *testing.T) {
	source := &fakeSource{}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})
	if _, err := useCase.Execute(context.Background(), mustList(t, "NCCONICKEL2USD-USDT")); err != nil {
		t.Fatal(err)
	}
	if len(source.calls) != 0 {
		t.Errorf("Source-Aufrufe = %d, erwartet 0", len(source.calls))
	}
}

func TestExecute_JederEintragHatDisplayName(t *testing.T) {
	source := &fakeSource{icons: map[asset.BaseAsset]asset.IconURL{mustBase(t, "ONDO"): mustURL(t, suiURL)}}
	useCase := appasset.NewGetAssetIconsUseCase(source, newFakeCache(), fixedClock{})

	icons, err := useCase.Execute(context.Background(), mustList(t, "ONDO-USDT,NCCOGOLD2USD-USDT,NCCOXAG2USD-USDT,NCCONICKEL2USD-USDT,ENA-USDT"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ONDO", "Gold", "Silber", "Nickel", "ENA"}
	if len(icons) != len(want) {
		t.Fatalf("Einträge = %+v", icons)
	}
	for index, icon := range icons {
		if icon.DisplayName != want[index] {
			t.Errorf("Eintrag %d (%s): DisplayName = %q, erwartet %q", index, icon.Symbol, icon.DisplayName, want[index])
		}
	}
}
