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

| Token                                   | Zweck                          |
| --------------------------------------- | ------------------------------ |
| `--color-background`                    | Seitenhintergrund              |
| `--color-surface`                       | Panels, Karten, Tabellen       |
| `--color-surface-raised`                | hervorgehobene Flächen         |
| `--color-border`                        | Rahmen, Trennlinien            |
| `--color-text` / `--color-text-muted`   | Text / Labels, Metadaten       |
| `--color-accent`                        | Aktionen, Links, Fokus         |
| `--color-positive` / `--color-negative` | Gewinn / Verlust, Long / Short |
| `--color-warning` (+ `-surface`)        | veraltete Daten, Hinweise      |
| `--color-negative-surface`              | Fehlerpanels                   |

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
  - **Kein `+`** bei Preisen, Mengen, Hebel und Abständen: Das sind keine
    Gewinn/Verlust-Größen. Standard der Formatter bleibt `signDisplay: 'negative'`.
  - Preise und Mengen ohne feste Stellen (so viele, wie BingX liefert, Nullen am Ende entfallen).
  - Fehlender Wert: einheitliches Platzhalterzeichen (`MISSING_VALUE`).
- Tönung über `pnlTone`: Ein Wert, der gerundet 0 ist, ist neutral (kein rotes `0.00`).

## Bausteine

| Baustein         | Aussehen                                                                                                                          |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Panel / Karte    | `surface`, 1px `border`, `radius-medium`, `shadow-card`                                                                           |
| Tabelle          | Panel (s.o.) als Scroll-Container; Kopfzeile `text-muted` + `font-size-small`; Zahlen rechtsbündig; Symbol links, fett und sticky |
| Primär-Button    | `accent`-Hintergrund, Text in `surface`, `radius-small`, `focus-visible`-Rahmen                                                   |
| Fehler           | `negative-surface`, Rahmen und Text `negative`, mit Retry-Button                                                                  |
| Veraltet-Hinweis | `warning-surface`, Rahmen und Text `warning`, `font-size-small`                                                                   |
| Long / Short     | Text "Long"/"Short", fett, in `positive`/`negative`                                                                               |

## Ansicht: Offene Positionen

Standard-Reihenfolge (ColumnKeys): `symbol`, `positionValue`, `margin`, `unrealizedProfit`,
`rMultiple` (nach `unrealizedProfit`), `realisedProfit`, `entry`, `tpSl`. Das Symbol bleibt immer fix links (sticky).
Entfallen: Seite, Menge, Mark, Hebel, Eröffnet, SL/TP einzeln, PnL % (jetzt in Unreal. PnL).

### Positionstabelle (BingX-Stil, flach)
Labels: Symbol, Positionswert, Margin, Unreal. PnL, Real. PnL, Einstieg, TP / SL.
Optik: keine Zebra-Streifen, nur 1px `border` zwischen Zeilen, Kopfzeile `font-size-x-small`,
`text-muted`, auf `surface` (nicht raised), Zeilen luftig (`--table-row-padding-block` = `space-4`).
Neue Tokens: `--font-size-x-small`, `--table-row-padding-block`, `--size-coin-icon` (20px),
`--size-side-bar` (3px), `--radius-chip`, `--color-chip-surface`, `--color-chip-text` (hell und dunkel, >= 4.5:1).

Zellaufbau (Klassen):
- **Symbol** (`th.cell.symbolCell` + `.barLong`/`.barShort`, Balken links per inset box-shadow):
  `.symbolMain` > `.coinIcon` (img) oder `.coinFallback` (Kreis mit Anfangsbuchstabe) + `.symbolName`
  (nur der Anzeigename, z.B. "ONDO", "Gold"; kein "-USDT") + `.chartButton` (dezent, rechts);
  darunter `.chips` > `.chip.chipLong|chipShort` (Text Long/Short), `.chip` "Isolated"/"Cross", `.chip` "7X".
- **Positionswert / Margin**: Zahl + `<span .unit>USDT</span>`.
- **Unreal. PnL**: `.twoLine` > Betrag (USDT, mit Vorzeichen, Ton `positive`/`negative`) + `.lineSub` "(+x,xx %)" gleicher Ton.
- **TP / SL**: `.twoLine` > TP-Zeile, darunter `.slLine` (SL, rot); "–" wenn nicht gesetzt. Zeilen mit
  Präfix "TP"/"SL" als Text (`.lineMuted`), damit Farbe nicht einziges Merkmal ist.
- Mobile (<768 px): Karte wie bisher, jede Zelle mit `data-label`; `.twoLine` rechtsbündig neben dem Label.
  Abnahme (Design-Review, gerendert 1440/390 px hell und dunkel): Label (`::before`) hat `margin-inline-end: auto`,
  Wert und Einheit stehen als Gruppe rechtsbündig (`.cell` ohne `gap`, Abstand über `padding-inline-end`).
  Der Long/Short-Balken sitzt mobile am Kartenrand (`.row:has(> .barLong|.barShort)`), desktop an der Symbol-Zelle.
- Desktop: Sticky Symbol-Spalte (auch Kopfzelle) hat ab 768 px eine 1px-`border`-Trennlinie rechts, damit gescrollte Werte nicht abgeschnitten wirken (Abnahme: 768 px gescrollt). Symbol-Zelle `width: 1%`, damit der Chart-Button nah am Namen steht und nicht am Spaltenende schwebt.
- Kontrast (berechnet, hell und dunkel): Text und Chips alle >= 4.5:1 (kleinster Wert: Short-Chip hell 4.88:1,
  Muted hell/dunkel 6.87:1, Long-Chip dunkel 6.69:1). Die Farbtöne `positive`/`negative` brauchen als Text auf
  `surface` dunkel 6.2:1 bis 8.2:1.
- Kontrast R-Elemente (berechnet, Review): R-Linie 3R (`chart-r-target`) hell 5.96:1 auf `surface`, dunkel 8.70:1;
  2R-Linie (`warning`) hell 6.56:1, dunkel 9.22:1; Chip "2R · BE ziehen" hell 5.98:1, dunkel 8.12:1; 3R auf
  `surface-raised` hell 5.00:1. Chart-Titel sind die Linienfarbe auf `surface`, Achsenlabel setzt lightweight-charts
  selbst kontrastreich. Die 3R-Linie (Teal, Hue ~195°) und die TP-Linie (Grün, Hue ~152°) unterscheiden sich
  zusätzlich durch Linienstil (3R gepunktet, TP gestrichelt) und Titel.
- Fehlender SL: "SL –" ohne Klasse `.slLine`, also gedämpft; rot (`.slLine`) nur bei gesetztem SL.
- **R-Spalte** (`rMultiple`, Label "R"): aktueller Gewinn in R, eine Nachkommastelle, Vorzeichen ("+1.4R"),
  Ton über `pnlTone`; `null` = "–" mit `aria-label="Risiko unbekannt"`. Bleibt auch mobil (Karte) sichtbar.
- **Aktionschip "2R · BE ziehen"** (`.chipAction`, `warning-surface` mit Text `warning`) in der Chip-Zeile der
  Symbolzelle, nur bei `breakEvenTriggerReached: true` und `breakEven` nicht `true`; bleibt auch mobil.
- **R-Linien im Chart**: 1R `text-muted` gepunktet ("1R"), 2R `warning` gestrichelt ("2R · BE"), 3R neues Token
  `--color-chart-r-target` gepunktet ("3R", ohne TP "3R Ziel"); liegt 3R innerhalb 0,1 % des Entry am TP, eine
  TP-Linie "TP = 3R". Neues Token `--color-chart-r-target` (hell/dunkel); 2R nutzt `--color-warning`.
  Keine Legendeneinträge für R-Linien, der Titel an der Linie genügt.
- **BE-Badge und "kein SL"** in der SL-Zeile der TP/SL-Zelle: **Badge "BE"** (`.breakEvenBadge`, nur bei
  `breakEven: true`): `positive-surface`-Hintergrund, Text `positive`, `font-size-small`, fett,
  `radius-small`, mit `aria-label="Break-even"` und `role="img"`. Hinweis **"kein SL"** (`.noStopLossHint`, nur bei
  `breakEven: false` und fehlendem SL): `text-muted`, `font-size-small`. Beide sind Text, nicht
  nur Farbe. Kontrast BE: hell ca. 5.3:1, dunkel ca. 6.7:1; "kein SL" hell ca. 6.9:1, dunkel ca. 6.9:1.
- **Coin-Icons**: `CoinIcon` zeigt die `iconUrl` der API (CoinGecko), für Gold/Silber (XAU/XAG) lokale SVGs
  (`src/assets/commodities/`); ohne URL oder bei Ladefehler einen Kreis mit dem Anfangsbuchstaben des Anzeigenamens.
  Icons (20 px) haben einen 1px-`border`-Innenrand (`box-shadow inset`), damit dunkle Logos im dunklen Theme sichtbar bleiben.
  Gold/Silber: Münze (Kreis mit dunklem Rand, ca. 1.5 px) mit Barren; der Rand sorgt auf hellem und dunklem Grund für die Kontur.
- **App-Kopfzeile** (`AppHeader`): `<header>` mit "Trading Intelligence Platform" über der Seite, `surface` mit 1px `border` unten, `font-size-base` fett. Der Titel steht bündig zur Seiteninhalt-Kante (gleiche Formel wie `.page`: `max(space-3, (100% - layout-max-width) / 2 + space-3)`, ab 768 px `space-5` statt `space-3`). Abnahme gerendert 390/768/1440 px: Titel und Überschrift haben dieselbe linke Kante.

- **Spalten umordnen (ab 768 px)**: Jeder Spaltenkopf (außer Symbol) hat einen Griff `.dragHandle`
  (Glyphe ⠇, `cursor: grab`, `text-muted`, 16px-Box = `--space-4`, ohne Abstand zum Label). Hover:
  `accent`-Glyphe, `surface`-Hintergrund, `border`-Rahmen. Fokus: 2px-Ring (`--border-width-emphasis`)
  innerhalb der Box. Die ganze Kopfzelle ist Drag-Quelle; Tastatur: Griff fokussieren, Alt + Pfeiltaste.
  Drop-Ziel: `.dropTarget` mit 2px-Indikator links (`accent`). Auf Mobile gibt es keinen Kopf und
  keinen Griff (`display: none`).
- **Toolbar** über der Tabelle (`.toolbar`): Ab 768 px reserviert sie immer `--space-6` Höhe, damit der
  Tabellenkopf nicht springt, wenn der Button **"Spalten zurücksetzen"** erscheint (nur bei
  nicht-Standard-Reihenfolge). Rechtsbündig, `surface-raised`, 1px `border`, `radius-small`,
  `font-size-small`, Fokusrahmen 2px. Auf Mobile ohne Inhalt ausgeblendet.

- **Mobile Karte**: Das Symbol ist der Kartentitel (links, ohne Spaltennamen), der Chart-Button
  steht rechts (`margin-left: auto`). Ab 768 px steht der Chart-Button direkt neben dem Symbol.

- **Unreal. PnL**, **Real. PnL**: mit `+`/`-`, getönt über `pnlTone` (PnL auf 2 Stellen).
- **Real. PnL** (`realisedProfit`): bereits realisierter Anteil der offenen Position. Nach den echten Daten
  vermutlich vor allem Gebühren und Funding, daher oft leicht negativ. 2 Nachkommastellen.
- Header: Summe des unrealisierten PnL in `font-size-x-large`, mit `+`/`-` und Tönung.

## Chart (Stufe 1)
Chart-Tokens angelegt in `design-tokens.css` (hell und dunkel):
- `--color-chart-up`, `--color-chart-down` (Kerzen, abgeleitet von positive/negative)
- `--color-chart-grid`, `--color-chart-crosshair`
- `--color-chart-entry` (neutral, accent), `--color-chart-stop-loss` (negative), `--color-chart-take-profit` (positive)
- Lightweight Charts liest Farben zur Laufzeit via `readToken()` aus CSS-Variablen.
- Theme-Wechsel zur Laufzeit: Das Panel hört auf `prefers-color-scheme` und ruft `applyTheme()` des Renderers auf (liest die Tokens neu).
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
- **Scroll-Hinweis in der Tabelle:** Bei 1440 px passt die Tabelle ohne Scrollen (gemessen: 1310 px
  Panelbreite, Tabelle 1310 px nach Reduktion von `--table-cell-padding-inline` auf `space-2` und
  Griff ohne Abstand). Unter ca. 1'306 px (gemessen 1280 px: Tabelle 1306 px, Panel 1230 px) scrollt
  sie wieder; dort sind die rechten Spalten (SL, TP, Eröffnet) erst nach horizontalem Scrollen sichtbar. Unter macOS sind
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
  - Die Grenze für das Scrollen ist jetzt gemessen (siehe oben), nicht mehr geschätzt.

## Design: Pro (Kraken-Stil)

Seriös, tiefes Indigo/Violett (dunkel bevorzugt, hell unterstützt), gedämpfter Violett-Akzent,
Radien 6/8 px, Borders statt Schatten, Seitenbreite 1360 px. Tabellenkopf auf
`--color-surface-raised`, Sticky-Spalte opak auf `--color-surface`.

- Tokens: siehe `design-tokens.css`; zusätzlich `--layout-max-width`, `--panel-radius`,
  `--table-cell-padding-block/-inline` (inline jetzt `space-2`, dichter), `--font-family-numeric`,
  `--color-on-accent`, `--color-positive-surface`, `--border-width-emphasis` (2px: Fokusrahmen,
  Drop-Indikator; theme-unabhängig).
- Kontrast: alle Text-Paare rechnerisch >= 4.5:1, hell (min. 4.82) und dunkel (min. 5.6).
  Griff-Glyphe (nur Icon, Nicht-Text, Ziel 3:1): `text-muted` auf `surface-raised` hell 5.76:1, dunkel 6.20:1.
- Die früheren Design-Lab-Varianten (Terminal, Clean, Exchange) wurden verworfen.
