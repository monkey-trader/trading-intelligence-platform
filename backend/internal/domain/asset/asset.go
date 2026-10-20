// Package asset enthält den Domain-Kern für Coin-Icons.
package asset

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// maxSymbols begrenzt die Zahl eindeutiger Symbole je Anfrage.
const maxSymbols = 50

// allowedIconHost ist der einzige erlaubte Host für Icon-URLs.
const allowedIconHost = "coin-images.coingecko.com"

var (
	baseAssetPattern = regexp.MustCompile(`^[A-Z0-9]{1,20}$`)
	symbolPattern    = regexp.MustCompile(`^[A-Z0-9]{1,20}(-[A-Z0-9]+)*$`)
)

// InvalidBaseAssetError meldet: Aus dem Symbol lässt sich kein Basis-Asset ableiten.
type InvalidBaseAssetError struct{ Value string }

func (err *InvalidBaseAssetError) Error() string {
	return fmt.Sprintf("ungültiges Basis-Asset aus Symbol %q", err.Value)
}

// InvalidIconURLError meldet: Die URL ist keine erlaubte Icon-URL.
type InvalidIconURLError struct{ Value string }

func (err *InvalidIconURLError) Error() string {
	return fmt.Sprintf("ungültige Icon-URL %q", err.Value)
}

// InvalidSymbolListError meldet: Die Symbolliste ist ungültig.
type InvalidSymbolListError struct{ Reason string }

func (err *InvalidSymbolListError) Error() string {
	return "ungültige Symbolliste: " + err.Reason
}

// commodityPrefix und commoditySuffix rahmen den Namen bei Rohstoff-Kontrakten (NCCO<NAME>2USD).
const (
	commodityPrefix = "NCCO"
	commoditySuffix = "2USD"
)

// BaseAsset ist der Basis-Teil eines Handelssymbols, z.B. SUI aus SUI-USDT.
type BaseAsset struct {
	value     string
	commodity bool
}

// NewBaseAsset leitet das Basis-Asset aus dem Symbol ab (Teil vor dem ersten Bindestrich, groß).
// Rohstoff-Kontrakte NCCO<NAME>2USD werden normalisiert (GOLD wird XAU).
func NewBaseAsset(symbol string) (BaseAsset, error) {
	first, _, _ := strings.Cut(strings.ToUpper(symbol), "-")
	if !baseAssetPattern.MatchString(first) {
		return BaseAsset{}, &InvalidBaseAssetError{Value: symbol}
	}
	if strings.HasPrefix(first, commodityPrefix) && strings.HasSuffix(first, commoditySuffix) {
		name := strings.TrimSuffix(strings.TrimPrefix(first, commodityPrefix), commoditySuffix)
		if name == "" {
			return BaseAsset{}, &InvalidBaseAssetError{Value: symbol}
		}
		if name == "GOLD" {
			name = "XAU"
		}
		return BaseAsset{value: name, commodity: true}, nil
	}
	return BaseAsset{value: first, commodity: first == "XAU" || first == "XAG"}, nil
}

// String liefert das Basis-Asset in Großbuchstaben.
func (base BaseAsset) String() string { return base.value }

// IsMetal meldet, ob das Asset Gold oder Silber (XAU, XAG) ist; dafür gibt es keine Coin-Icons.
func IsMetal(base BaseAsset) bool {
	return base.commodity && (base.value == "XAU" || base.value == "XAG")
}

// IsCommodity meldet Rohstoffe (NCCO-Kontrakte, XAU, XAG).
func IsCommodity(base BaseAsset) bool { return base.commodity }

// DisplayName liefert den Anzeigenamen: Gold, Silber, Rohstoffe mit großem Anfangsbuchstaben, sonst den Ticker.
func DisplayName(base BaseAsset) string {
	switch {
	case base.value == "XAU" && base.commodity:
		return "Gold"
	case base.value == "XAG" && base.commodity:
		return "Silber"
	case base.commodity:
		return base.value[:1] + strings.ToLower(base.value[1:])
	default:
		return base.value
	}
}

// IconURL ist eine geprüfte https-URL auf den CoinGecko-Image-Host.
type IconURL struct{ value string }

// NewIconURL akzeptiert nur https, Host coin-images.coingecko.com, ohne Port und Userinfo.
func NewIconURL(raw string) (IconURL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host != allowedIconHost ||
		parsed.User != nil || parsed.Port() != "" {
		return IconURL{}, &InvalidIconURLError{Value: raw}
	}
	return IconURL{value: raw}, nil
}

// String liefert die URL.
func (icon IconURL) String() string { return icon.value }

// SymbolList ist eine geprüfte, deduplizierte Liste von Handelssymbolen.
type SymbolList struct{ symbols []string }

// ParseSymbolList zerlegt eine kommagetrennte Liste (1 bis 50 eindeutige Symbole).
func ParseSymbolList(raw string) (SymbolList, error) {
	seen := map[string]bool{}
	symbols := []string{}
	for _, symbol := range strings.Split(raw, ",") {
		if !symbolPattern.MatchString(symbol) {
			return SymbolList{}, &InvalidSymbolListError{Reason: fmt.Sprintf("ungültiges Symbol %q", symbol)}
		}
		if !seen[symbol] {
			seen[symbol] = true
			symbols = append(symbols, symbol)
		}
	}
	if len(symbols) > maxSymbols {
		return SymbolList{}, &InvalidSymbolListError{Reason: "zu viele Symbole"}
	}
	return SymbolList{symbols: symbols}, nil
}

// Symbols liefert eine Kopie der Symbole in Eingabereihenfolge.
func (list SymbolList) Symbols() []string { return append([]string(nil), list.symbols...) }

// CoinIcon ordnet einem Symbol sein Basis-Asset, den Anzeigenamen und (falls bekannt) das Icon zu.
type CoinIcon struct {
	Symbol      string
	BaseAsset   BaseAsset
	DisplayName string
	IconURL     *IconURL
}
