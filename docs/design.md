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
4. **Hell und dunkel, Palette Gruvbox.** Das Theme folgt standardmäßig dem System (`prefers-color-scheme`); im Kopf lässt sich
   System/Hell/Dunkel wählen (`useTheme`, `data-theme` am `<html>`, gespeichert unter `tip_theme_v1`).
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
`realisedProfit`, `entry`, `tpSl`, `beTarget`. Das Symbol bleibt immer fix links (sticky).
Entfallen: Seite, Menge, Mark, Hebel, Eröffnet, SL/TP einzeln, PnL % (jetzt in Unreal. PnL).

### Positionstabelle (BingX-Stil, flach)
Labels: Symbol, Positionswert, Margin, Unreal. PnL, Real. PnL, Einstieg, TP / SL, TPs (je 33 %).
Optik: keine Zebra-Streifen, nur 1px `border` zwischen Zeilen, Kopfzeile `font-size-x-small`,
`text-muted`, auf `surface` (nicht raised), Zeilen luftig (`--table-row-padding-block` = `space-4`).
Neue Tokens: `--font-size-x-small`, `--table-row-padding-block`, `--size-coin-icon` (20px),
`--size-side-bar` (3px), `--radius-chip`, `--color-chip-surface`, `--color-chip-text` (hell und dunkel, >= 4.5:1), `--color-r-target-surface` (TP3-Pille der TP-Spalte, hell und dunkel).

Zellaufbau (Klassen):
- **Symbol** (`th.cell.symbolCell` + `.barLong`/`.barShort`, Balken links per inset box-shadow):
  `.symbolMain` > `.coinIcon` (img) oder `.coinFallback` (Kreis mit Anfangsbuchstabe) + `.symbolName`
  (nur der Anzeigename, z.B. "ONDO", "Gold"; kein "-USDT") + `.chartButton` (dezent, rechts);
  darunter `.chips` > `.chip.chipLong|chipShort` (Text Long/Short), `.chip` "Isolated"/"Cross", `.chip` "7X".
- **Positionswert / Margin**: Zahl + `<span .unit>USDT</span>`.
- **Unreal. PnL**: `.twoLine` > Betrag (USDT, mit Vorzeichen, Ton `positive`/`negative`) + `.lineSub` "(+x,xx %)" gleicher Ton.
- **TP / SL**: `.twoLine` > TP-Zeile, darunter `.slLine` (SL, rot); "–" wenn nicht gesetzt. Zeilen mit
  Präfix "TP"/"SL" als Text (`.lineMuted`), damit Farbe nicht einziges Merkmal ist.
- **TPs (je 33 %)**: `.levelStack` > drei Tonpillen (TP1 = 1R neutral in `.levelFirst`, TP2 = 2R + BE, TP3 = 3R); früher zwei Tonpillen `.levelPill` (rechtsbündig, `font-size-small`, fett, `radius-chip`): "BE <2R-Preis>" in `.levelBe` (`warning-surface` + `warning`, wie die 2R-Linie im Chart) und "TP3 <3R-Preis>" in `.levelTarget` (`--color-r-target-surface` + `--color-chart-r-target`, wie die 3R-Linie). Preisformat wie Einstieg, tabular-nums. Ohne `levels` "–" mit `aria-label="Risiko unbekannt"`. Pille statt Präfix-Span, weil der Test "BE 1.08" als Text in einem Element erwartet. Kontrast (berechnet): BE hell 5.98:1, dunkel 8.12:1; Ziel hell 5.13:1, dunkel 7.48:1.
- Mobile (<768 px): Karte wie bisher, jede Zelle mit `data-label`; `.twoLine` rechtsbündig neben dem Label.
  Abnahme (Design-Review, gerendert 1440/390 px hell und dunkel): Label (`::before`) hat `margin-inline-end: auto`,
  Wert und Einheit stehen als Gruppe rechtsbündig (`.cell` ohne `gap`, Abstand über `padding-inline-end`).
  Der Long/Short-Balken sitzt mobile am Kartenrand (`.row:has(> .barLong|.barShort)`), desktop an der Symbol-Zelle.
- Desktop: Sticky Symbol-Spalte (auch Kopfzelle) hat ab 768 px eine 1px-`border`-Trennlinie rechts, damit gescrollte Werte nicht abgeschnitten wirken (Abnahme: 768 px gescrollt). Symbol-Zelle `width: 1%`, damit der Chart-Button nah am Namen steht und nicht am Spaltenende schwebt.
- Kontrast (berechnet, hell und dunkel): Text und Chips alle >= 4.5:1 (kleinster Wert: Short-Chip hell 4.88:1,
  Muted hell/dunkel 6.87:1, Long-Chip dunkel 6.69:1). Die Farbtöne `positive`/`negative` brauchen als Text auf
  `surface` dunkel 6.2:1 bis 8.2:1.
- Kontrast R-Elemente (berechnet, Review): R-Linie 3R (`chart-r-target`) hell 5.96:1 auf `surface`, dunkel 8.70:1;
  2R-Linie (`warning`) hell 6.56:1, dunkel 9.22:1; Chip "BE ziehen" hell 5.98:1, dunkel 8.12:1; 3R auf
  `surface-raised` hell 5.00:1. Chart-Titel sind die Linienfarbe auf `surface`, Achsenlabel setzt lightweight-charts
  selbst kontrastreich. Die 3R-Linie (Teal, Hue ~195°) und die TP-Linie (Grün, Hue ~152°) unterscheiden sich
  zusätzlich durch Linienstil (3R gepunktet, TP gestrichelt) und Titel.
- Fehlender SL: "SL –" ohne Klasse `.slLine`, also gedämpft; rot (`.slLine`) nur bei gesetztem SL.
- **R-Spalte entfernt** (2026-10-21): Die Spalte „TPs (je 33 %)“ mit den Preisen und der Chip ersetzen sie; `currentRMultiple` bleibt in der API (Journal, Stufe 3).
- **Aktionschip "BE ziehen"** (`.chipAction`, `warning-surface` mit Text `warning`) in der Chip-Zeile der
  Symbolzelle, nur bei `breakEvenTriggerReached: true` und `breakEven` nicht `true`; bleibt auch mobil.
- **R-Linien im Chart** (seit TASK-14): 2R `warning` gestrichelt ("BE"), 3R neues Token
  `--color-chart-r-target` gepunktet ("TP1"); 1R entfällt. Liegt 3R innerhalb 0,1 % des Entry an einem TP
  (`coincidesWithTakeProfit` vom Backend), eine Linie "TP = 3R". Neues Token `--color-chart-r-target` (hell/dunkel); 2R nutzt `--color-warning`.
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

- **Ladezustand** (Status `loading` ohne Daten, Prototyp `prototype.html`, Zustand "Lädt"): Die Tabelle
  bleibt mit Kopf stehen, die Sortier-Knöpfe sind `disabled` (Cursor `default`, Schrift wie im Kopf), die
  Tabelle trägt `aria-busy`. Es gibt drei Platzhalterzeilen (`position-placeholder-row`, `aria-hidden`):
  Desktop wie Datenzeilen mit 40 px Höhe und einem Balken (1em, `surface-raised`, Radius `radius-small`) je
  Spalte; mobil eine Karte mit einer Zeile je Spalte (Balken halb so breit). Der Ladetext bleibt
  `role="status"`. Beim ersten Laden wächst der Inhalt beim Wechsel zu Daten (Kopf der Ansicht kommt dazu,
  ca. 24 px am Desktop, ca. 60 px mobil). Beim Neuladen mit Daten bleibt alles stehen (kein Platzhalter).
  Platzhalter sind dekorativ; ihr Kontrast (ca. 1.1:1 bzw. 1.3:1) ist bewusst keine Anforderung.
- **CSS-Reihenfolge (Gotcha):** Mobile Basisregeln (Karten, Platzhalter, Mobil-Sortierung) stehen vor dem
  `@media (min-width: 768px)`-Block. Stünden sie danach, überschrieben sie dort gleich spezifische
  Desktop-Regeln (so war die Mobil-Sortierung am Desktop sichtbar und die Platzhalter als Kästen).
- **Mobile Karte**: Das Symbol ist der Kartentitel (links, ohne Spaltennamen), der Chart-Button
  steht rechts (`margin-left: auto`). Ab 768 px steht der Chart-Button direkt neben dem Symbol.
  Auf Touch-Geräten (`pointer: coarse`) mindestens `--size-touch-target` (44 px) hoch; ohne Regel waren es 21 px (TASK-7).
  Der Long/Short-Balken der Mobilkarte sitzt am Kartenrand (`.row:has(...)`); die Desktop-Regel (ab 768 px, an der Symbol-Zelle) steht nach der Basisregel, sonst gewinnt die Basis.

- **Unreal. PnL**, **Real. PnL**: mit `+`/`-`, getönt über `pnlTone` (PnL auf 2 Stellen).
- **Real. PnL** (`realisedProfit`): bereits realisierter Anteil der offenen Position. Nach den echten Daten
  vermutlich vor allem Gebühren und Funding, daher oft leicht negativ. 2 Nachkommastellen.
- Header: Summe des unrealisierten PnL in `font-size-large` (fett, Zeilenhöhe `line-height-tight`), mit `+`/`-` und Tönung. Kontostand und Perioden-PnL in `font-size-base`, Labels in `font-size-small`. Der Kennzahlen-Kopf ist kompakt: Innenabstand `space-2`/`space-3`, Zeitraum-Knöpfe mindestens `size-control-compact` (26 px), bei `pointer: coarse` mindestens `size-touch-target` (44 px). Desktop ca. 80 px hoch (vorher 120 px), Mobile ca. 167 px (vorher 236 px).

### Ursprungs-SL von Hand (Zelle "TPs (je 33 %)")
- Bei unbekanntem Risiko (keine `levels`) erscheint der Button "SL eintragen" (accessible name
  "Ursprünglichen SL eintragen für <Symbol>"). Klick öffnet inline ein Textfeld (`inputmode="decimal"`,
  Label "Ursprünglicher SL") mit Speichern/Abbrechen; Fokus springt ins Feld, Escape bricht ab.
- Fehler stehen als `role="alert"` unter dem Feld (Validierung vor dem Senden, Serverfehler gemappt).
- Bei `initialStopLossSource: MANUAL` zeigt die Zelle das Badge "manuell" und den Button "Zurücksetzen".
- Touch-Ziele mindestens 44 px (`--size-touch-target`), Farben und Abstände nur über Tokens, auch in der mobilen Karte.
- Button und Eingabefeld: Fläche `surface-raised` (Button) bzw. `surface` (Feld), Rahmen `--color-control-border`
  (≥ 3:1 auf surface und surface-raised; berechnet hell 3.87:1 auf surface, 3.55:1 auf raised; dunkel 4.17:1 auf surface, 3.17:1 auf raised).
  Dunkel war `#928374` (2.40:1 auf raised, Befund TASK-7) und ist jetzt `#a89984`. Text `text` (hell 14.2:1, dunkel 13.1:1).
  Ohne diesen Rahmen verschwindet der Button weiß auf weiß bzw. dunkel auf dunkel.
- Fokus: `:focus-visible` mit `--border-width-emphasis` und `--color-accent` (Ring, 2px Abstand) an Button und Feld; das Feld erhält den Fokus beim Öffnen, Escape bricht ab.
- Feld-Breite `--size-sl-input-width`, Zahlen `tabular-nums`, rechtsbündig wie die Preise.
- Fehlertext `role="alert"`: `negative` auf `surface` hell 5.75:1, dunkel 6.21:1; steht unter dem Feld, nicht als Overlay.

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
- **Scroll-Verhalten der Tabelle (gemessen 2026-10-25, TASK-7):** Ab 1100 px passt die Tabelle ohne
  Scrollen, auch mit langen Symbolen und sechsstelligen Werten (Tabelle 1050 px bei 1100 px Fenster).
  Bei 1280 px mit Extremwerten (36-stelliges Symbol, 7-stellige Beträge, Preise mit 11 Nachkommastellen)
  bleiben ca. 30 px Reserve (Tabelle mindestens 1200 px, Panel 1230 px). Darunter scrollt der Container
  innerhalb des Panels (Symbol bleibt sticky): 1024 px mit langen Werten 75 px, 900 px 199 px, 768 px
  331 px (normale Werte 768 px: 97 px). Unter macOS sind Scrollbalken ausgeblendet, der Hinweis fehlt also.
  Optionen: Schatten am rechten Rand als Scroll-Andeutung, oder den Tabellen-Breakpoint anheben und
  dazwischen Karten im Raster zeigen. Mit dem Panel-Layout (Seitenbreite) zusammen entscheiden.
- **Leerer Raum über der Tabelle am Desktop:** Die reservierte Toolbar (32 px) plus Abstand ergeben
  ca. 90 px Luft zwischen Panel-Kopf und Tabelle (gemessen 1280 px, Fenster mit Daten). Der Prototyp
  hat diesen Raum nicht. Vorschlag: Toolbar nur reservieren, wenn der Reset-Knopf sichtbar ist, oder
  den Abstand kleiner setzen (Entscheidung offen).
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

### Erklärung der TP-Spalte (i)
- Kleiner (i)-Knopf (`.infoButton`, Kreis, „i“ per `::before`) im Kopf der Spalte „TPs (je 33 %)“; auf Mobile (kein Kopf) in der Toolbar (`.infoButtonToolbar`). Er öffnet den modalen `InfoDialog` (natives `<dialog>`, Esc/Hintergrund/„Schließen“) mit der Erklärung `BreakEvenTargetInfo`: 1R = ursprüngliches Risiko, BE ab 2R (SL auf Break-even), TP1 bei 3R (33 % nur Richtwert); 1R, TP2, TP3 entfallen (TASK-14, Tabellenumbau in TASK-7).

## Wails-Desktop-App (macOS)

Die Desktop-Hülle (`clients/desktop`, Wails v2) zeigt denselben Web-Client mit `mac.TitleBarHiddenInset()`:
Titelleiste unsichtbar, Ampel-Buttons (Schließen/Minimieren/Vollbild, ca. 80 px breit, 28 px hoch) liegen oben links
über dem Inhalt. Im Browser ändert sich nichts.

- **Erkennung**: `markWailsApp()` (`presentation/shared/platform/wailsApp.ts`) setzt `data-desktop="wails"` am `<html>`,
  wenn `window.runtime` existiert oder der User-Agent "wails.io" enthält (Wails v2 setzt das auf macOS). Alle App-Regeln
  hängen an diesem Attribut, nie am Browser-Standard.
- **Titelleiste mit Titel**: In der App ist der `AppHeader` genau so hoch wie die Titelleiste (`--size-titlebar-height`,
  32 px, Gesamthöhe inkl. Rahmen 33 px) und trägt selbst den Titel. Der Titel steht mittig im Fenster, in
  `--font-size-small`, `regular` und `color-text-muted` (Kontrast 5.2:1 hell, 6.0:1 dunkel). Ein Raster mit zwei
  gleich breiten Randspalten (`--size-titlebar-side`, 10 rem) hält ihn mittig: links bleibt die Ampel frei, rechts steht
  die Design-Wahl. Unter 700 px kürzt sich der Titel mit "…" (`ellipsis`), er bricht nie um und rutscht nie unter die
  Ampel. Der Header trägt `--wails-draggable: drag`. Die Ampel-Zone ist also kein eigener Balken mehr, keine zweite Zeile.
  Im Browser bleibt der Header unverändert (Titel links, Höhe 51 px).
- **Drag-Zonen**: `AppHeader` und die Kennzahlen-Kopfzeile (`OpenPositionsHeader`). Interaktive Elemente
  (`a, button, input, select, textarea, label`) setzen `--wails-draggable: no-drag`, sonst zieht der Klick das Fenster.
  Wails prüft nur das Computed-Style des Klick-Ziels, `--wails-draggable` vererbt sich aber. Deshalb explizit setzen.
- **Text und Zeiger**: Alle Texte sind markierbar, damit BE-Preis, SL, TP, Zahlen und Positions-IDs
  kopierbar sind. Nur `button`, `summary`, `select` und `nav` haben `user-select: none`. Zeiger: `cursor: default`, Links und Eingabefelder
  behalten `pointer` bzw. `text`. Die Klasse `.dragHandle` (grab) bleibt unverändert.
- **Scrollen**: `overscroll-behavior: none` am `<html>`, dünne Scrollbars (`scrollbar-width: thin`) in
  `--color-control-border` auf transparent.
- **Tokens**: `--size-titlebar-height` (32 px), `--size-titlebar-side` (10 rem) und `--line-height-tight` (1.2) in `design-tokens.css`. Keine Farbe ist app-spezifisch.
- **Offen**: Das Fenster ist translucent (`WindowIsTranslucent`), aber der Hintergrund bleibt deckend
  (`--color-background`). Ein transparenter Hintergrund würde einen Materialeffekt brauchen und ist nicht umgesetzt.
