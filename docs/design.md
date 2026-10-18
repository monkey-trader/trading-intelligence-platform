# Design – Trading Intelligence Platform

Quelle der Wahrheit für das UI. Der `designer`-Agent hält dieses Dokument aktuell.
Stand: beschreibt den **bestehenden** Web-Client (`clients/web`) plus die Zielrichtung.

## Leitidee
Ein ruhiges, dichtes Trading-Journal: viele Zahlen auf einen Blick, klare Hierarchie,
Gewinn/Verlust sofort erkennbar. Inspiration: professionelle Trading-Apps (dunkel, aufgeräumt,
panelbasiert) – eigener Look, keine Kopie fremder Marken oder Logos.

## Grundregeln
1. **Nur Design-Tokens.** Komponenten verwenden ausschließlich Variablen aus
   `clients/web/src/styles/design-tokens.css`. Erlaubte Ausnahmen: `1px`-Rahmen, `2px`-Fokusrahmen,
   Breakpoints in Media Queries (CSS erlaubt dort keine Variablen).
2. **CSS Modules**, keine Inline-Styles.
3. **Mobile-first.** Unter 768 px werden Tabellen zu Karten (Spaltenname aus `data-label`).
   Ab 768 px ist das Tabellen-Panel ein horizontaler Scroll-Container: Ist die Tabelle breiter
   als der Platz (bei offenen Positionen bis ca. 1'280 px Fensterbreite), scrollt sie im Panel,
   nie die ganze Seite. Die Symbol-Spalte bleibt dabei links stehen (`position: sticky`).
   Der Container ist per Tastatur fokussierbar (`tabIndex={0}`, `role="region"`, `aria-label`).
4. **Hell und dunkel.** Das Theme folgt dem System (`prefers-color-scheme`).
   Dunkel ist die primäre Zielansicht; hell muss trotzdem sauber funktionieren.

## Farben (Tokens)
| Token | Zweck |
|---|---|
| `--color-background` | Seitenhintergrund |
| `--color-surface` | Panels, Karten, Tabellen |
| `--color-surface-raised` | hervorgehobene Flächen |
| `--color-border` | Rahmen, Trennlinien |
| `--color-text` / `--color-text-muted` | Text / Labels, Metadaten |
| `--color-accent` | Aktionen, Links, Fokus |
| `--color-positive` / `--color-negative` | Gewinn / Verlust, Long / Short |
| `--color-warning` (+ `-surface`) | veraltete Daten, Hinweise |
| `--color-negative-surface` | Fehlerpanels |

Kontrast: Text mindestens 4.5:1 (WCAG AA) in beiden Themes.

## Abstände, Radien, Schatten
- Abstände: `--space-1` (4) bis `--space-6` (32), 4er-Raster.
- Radien: `--radius-small` (Buttons, Hinweise), `--radius-medium` (Panels, Karten).
- Schatten: nur `--shadow-card`.

## Typografie
- Schrift: System-Schrift (`--font-family-base`).
- Größen: `--font-size-small` (Labels), `-base` (Text), `-large` (Titel, Symbol in Karten),
  `-x-large` (Kennzahl im Header, z.B. Gesamt-PnL).
- Gewichte: `regular` und `strong`, keine weiteren.

## Zahlen
- Immer `font-variant-numeric: tabular-nums`, in Tabellen rechtsbündig.
- Formatierung nur über die Formatter in `presentation/*/format/`:
  - Locale **de-CH** (`1'234.5`), exakt mit Dezimal-Strings, nie über `Number`.
  - Kaufmännisch gerundet, kein `-0.00`.
  - PnL mit 2 Nachkommastellen, Prozente mit 2 Nachkommastellen.
  - **Vorzeichen bei PnL:** PnL-Werte und PnL-Prozente zeigen positive Werte mit `+`
    (`signDisplay: 'exceptZero'` in `formatDecimal` bzw. `formatRatioAsPercent`). Gewinn hängt so
    nicht nur an der Farbe. Ein Wert, der gerundet 0 ist, steht ohne Vorzeichen (`0.00`, nie
    `+0.00` oder `-0.00`). Das Vorzeichen richtet sich nach dem gerundeten Wert (`0.005` → `+0.01`).
  - **Kein `+`** bei Preisen, Mengen, Hebel und Abständen (z.B. Liq.-Abstand): Das sind keine
    Gewinn/Verlust-Größen. Standard der Formatter bleibt `signDisplay: 'negative'`.
  - Preise und Mengen ohne feste Stellen (so viele, wie BingX liefert, Nullen am Ende entfallen).
  - Fehlender Wert: einheitliches Platzhalterzeichen (`MISSING_VALUE`).
- Tönung über `pnlTone`: Ein Wert, der gerundet 0 ist, ist neutral (kein rotes `0.00`).

## Bausteine
| Baustein | Aussehen |
|---|---|
| Panel / Karte | `surface`, 1px `border`, `radius-medium`, `shadow-card` |
| Tabelle | Panel (s.o.) als Scroll-Container; Kopfzeile `text-muted` + `font-size-small`; Zahlen rechtsbündig; Symbol links, fett und sticky |
| Primär-Button | `accent`-Hintergrund, Text in `surface`, `radius-small`, `focus-visible`-Rahmen |
| Fehler | `negative-surface`, Rahmen und Text `negative`, mit Retry-Button |
| Veraltet-Hinweis | `warning-surface`, Rahmen und Text `warning`, `font-size-small` |
| Long / Short | Text "Long"/"Short", fett, in `positive`/`negative` |

## Ansicht: Offene Positionen
Spalten in dieser Reihenfolge: Symbol, Seite, Menge, Einstieg, Mark, Hebel, Unreal. PnL, PnL %,
Real. PnL, Liq.-Preis, Liq.-Abstand, Eröffnet.
- **Unreal. PnL**, **PnL %**, **Real. PnL**: mit `+`/`-`, getönt über `pnlTone` (PnL auf 2 Stellen,
  PnL % auf 4 Stellen des Verhältnisses = 2 Prozent-Stellen).
- **Real. PnL** (`realisedProfit`): bereits realisierter Anteil der offenen Position. Nach den echten Daten
  vermutlich vor allem Gebühren und Funding, daher oft leicht negativ. 2 Nachkommastellen.
- Header: Summe des unrealisierten PnL in `font-size-x-large`, mit `+`/`-` und Tönung.

## Chart (Stufe 1)
Chart-Tokens angelegt in `design-tokens.css` (hell und dunkel):
- `--color-chart-up`, `--color-chart-down` (Kerzen, abgeleitet von positive/negative)
- `--color-chart-grid`, `--color-chart-crosshair`
- `--color-chart-entry` (neutral, accent), `--color-chart-stop-loss` (negative), `--color-chart-take-profit` (positive)
- Lightweight Charts liest Farben zur Laufzeit via `readToken()` aus CSS-Variablen.
- **Bekannte Limitation:** Theme-Wechsel zur Laufzeit (ohne Neuladen) aktualisiert Chart-Farben nicht;
  wird in Stufe 2 gelöst (Listener auf `prefers-color-scheme`-Wechsel).
- Read-only Chart mit Entry-Marker, SL/TP-Linien (gestrichelt). Linien-Labels (Preis + %): Stufe 2+.

## Offene Punkte
- **Seitenbreite `max-width: 1280px`:** Für eine Desktop-App mit mehreren Panels (Chart,
  Positionen, Historie) eher volle Breite mit Panel-Layout. Entscheiden, wenn der Chart kommt.
- **Breakpoint 768 px** steht in drei Media Queries (`OpenPositionsHeader`, `OpenPositionsPage`,
  `OpenPositionsTable`); `--breakpoint-table` ist nur Referenz, jede Stelle trägt einen Kommentar
  mit Verweis darauf (per `grep breakpoint-table` auffindbar). Eine echte einzige Quelle ginge nur
  über `@custom-media` (PostCSS-Plugin oder Lightning CSS als Vite-Transformer). Das ändert die
  CSS-Pipeline und lohnt sich bei drei Stellen noch nicht. Neu bewerten, wenn mehr Ansichten
  (Chart, Historie) dazukommen oder ein zweiter Breakpoint nötig wird.
- **Scroll-Hinweis in der Tabelle:** Zwischen 768 px und ca. 1'280 px sind die rechten Spalten
  (Liq.-Abstand, Eröffnet) erst nach horizontalem Scrollen sichtbar. Unter macOS sind
  Scrollbalken standardmäßig ausgeblendet, der Hinweis fehlt also. Optionen: Schatten am rechten
  Rand als Scroll-Andeutung, oder den Tabellen-Breakpoint anheben und dazwischen Karten im
  Raster zeigen. Mit dem Panel-Layout (Seitenbreite) zusammen entscheiden.
- **Scroll-Container der Tabelle (Review-Befunde, klein, im Browser noch nicht geprüft):**
  - Unter 768 px (Kartenansicht) scrollt der Container nicht, hat aber `role="region"` und
    `tabIndex={0}`: ein überflüssiger Tab-Stopp. Lösung wäre `tabIndex` nur ab 768 px (`matchMedia`).
  - Die feste Symbol-Spalte hat keinen Trenner zum darunter scrollenden Inhalt. Vorschlag:
    `box-shadow: inset -1px 0 0 var(--color-border)`. Bei `border-collapse: collapse` können die
    Zeilenlinien an festen Zellen in Chrome und Safari beim Scrollen springen; dann auf
    `border-collapse: separate; border-spacing: 0` umstellen.
  - Das `aria-label` des Containers ("Offene Positionen, horizontal scrollbar") und das der Tabelle
    ("Offene Positionen") sind fast gleich; Screenreader nennen es doppelt. Container besser
    "Tabelle offene Positionen, scrollbar" nennen.
  - Die Grenze "ca. 1'280 px", ab der die Tabelle ohne Scrollen passt, ist nicht gemessen.
