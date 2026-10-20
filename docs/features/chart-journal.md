# Feature: Chart im Journal (Entry, SL, TP)

Status: geplant · Umsetzung nach Phase 3 des Migrationsplans (`docs/migration/go-wails.md`)

## Ziel
Zu jeder Position einen Kerzen-Chart anzeigen, auf dem Entry, Stop-Loss und Take-Profit
als Linien und Ein-/Ausstiege als Marker zu sehen sind. Offene Positionen live,
abgeschlossene Trades zur Auswertung im Journal.

**Das Journal handelt nicht.** Alles ist read-only.

## Nicht im Umfang (bewusst)
- SL/TP im Chart verschieben oder Orders ändern/anlegen/stornieren.
  → Bräuchte Handelsrechte am API-Key und widerspricht der Sicherheitsregel "nur lesen".
- Zeichenwerkzeuge (Trendlinien, Fibonacci) und eingebaute Indikatoren. Eigene Analysen zeichnet
  der Nutzer bei TradingView und hinterlegt nur den Link (siehe "TradingView-Link").
- Die lizenzpflichtige TradingView-Charting-Library.

## Technik
- Chart: **TradingView Lightweight Charts** (Open Source, Canvas).
  - Kerzen: Candlestick-Serie
  - Entry / SL / TP: `createPriceLine` (waagrechte Linie mit Label)
  - Ein-/Ausstieg: Serien-Marker (Pfeile auf der Zeitachse)
  - Volumen optional als Histogramm darunter
- Farben über Design-Tokens: Entry neutral, SL rot, TP grün, gestrichelt für offene Orders.
- **API-first:** Das UI holt Kerzen und Orders ausschließlich über die eigene API,
  nie direkt bei BingX.

> Exakte BingX-Endpunkte, Parameter und Feldnamen vor der Umsetzung in der aktuellen
> BingX-API-Doku prüfen und echte Antworten als Fixtures ablegen
> (Lehre aus dem Positions-Bug im Prototyp `../bingx-journal`).

---

## TradingView-Link (Journal-Notiz)
Pro Position kann der Nutzer einen TradingView-Link einfügen (geteilter Chart/Snapshot mit der
eigenen Analyse). Das Journal zeigt ihn als Link, ohne ihn einzubetten oder abzurufen.
- Feld an der Journal-Notiz (`chartUrl`), nur `https://` und Host `tradingview.com`/Subdomains,
  sonst Validierungsfehler (kein `javascript:`), Link mit `rel="noopener noreferrer"`.
- Kommt mit der Notiz-API, nicht mit Stufe 1.

---

## Stufe 1 – Statischer Chart pro Position

**Backend**
- Kerzen: öffentlicher BingX-Klines-Endpunkt (kein API-Key nötig).
  Cache in SQLite (`candles`, Schlüssel: symbol + interval + openTime), damit nicht jedes
  Öffnen eines Charts BingX abfragt.
- Offene Orders: Import der offenen Orders (signierter GET, Lese-Key).
  Relevant sind Stop- und Take-Profit-Orders mit ihrem Auslösepreis.

**Domain**
- `Candle` (symbol, interval, openTime, open, high, low, close, volume)
- `ProtectiveOrder` (orderId, positionId/symbol, kind: STOP_LOSS | TAKE_PROFIT,
  triggerPrice, quantity, status)
- Value Objects für Preis, Menge, Intervall; typisierte Errors
  (z.B. `UnsupportedIntervalError`, `TriggerPriceMustBePositiveError`).

**API** (zuerst in `docs/openapi.yaml`)
- `GET /positions/{id}/chart?interval=15m&from=&to=`
  → `{ candles: [...], entry: {...}, stopLoss: [...], takeProfit: [...], markers: [...] }`
- Mehrere SL/TP pro Position sind möglich (Teil-TPs) → Listen, nicht Einzelwerte.

**UI**
- Chart-Ansicht pro Position, Intervall-Auswahl (5m, 15m, 1h, 4h, 1d).
- Linien mit Label: Preis + Abstand in % zum Entry.

**Akzeptanzkriterien**
- Offene Position mit gesetztem SL und TP zeigt alle drei Linien am richtigen Preis.
- Position ohne SL/TP zeigt nur Entry, ohne Fehler.
- Mehrere Teil-TPs werden alle angezeigt.

---

## Stufe 2 – Live-Update für offene Positionen

**Stand 2026-10-20: Polling umgesetzt** (UI pollt `GET /positions/{id}/chart` alle 5 s ohne `since`, pausiert bei verstecktem Tab, Zoom bleibt erhalten, Chart folgt dem Theme). Orders-Cache 5 s, geteilt mit der Positionsliste. Streaming weiterhin offen.

**Zuerst einfach:** Polling.
- UI fragt alle 5 s `GET /positions/{id}/chart?since=<letzte Kerze>` ab.
- Backend aktualisiert die laufende (noch offene) Kerze und hängt neue an.
- Offene Orders alle 30–60 s neu importieren, damit nachgezogene SL/TP sichtbar werden.

**Später, nur wenn Polling stört:** Streaming.
- Backend abonniert den BingX-WebSocket für Kline-Updates der Symbole mit offenen Positionen.
- Backend reicht Updates per Server-Sent Events weiter: `GET /stream/candles?symbol=&interval=`.
- Reconnect mit Backoff, nach Reconnect Lücke über REST nachladen.

**Akzeptanzkriterien**
- Laufende Kerze bewegt sich ohne Neuladen der Seite.
- SL, der bei BingX nachgezogen wurde, erscheint spätestens nach 60 s an neuer Stelle.
- Netzwerkabbruch führt nicht zu doppelten oder fehlenden Kerzen.

---

## CRV-Level (umgesetzt 2026-10-20, offene Positionen)

Aus dem ursprünglichen Risiko (Entry→SL) berechnet das Backend 1R, 2R (Break-even-Trigger) und 3R (CRV-Ziel 1:3).
- Chart: `PositionChart.levels` als gepunktete (1R, 3R) bzw. gestrichelte (2R, „2R · BE“) Linien; liegt 3R innerhalb von 0,1 % des Entry am TP, zeigt eine Linie „TP = 3R“.
- Tabelle: Chip „▲ 2R · BE ziehen“, sobald 2R erreicht und der SL noch nicht auf Break-even ist. Eine eigene R-Spalte gibt es nicht mehr (ersetzt durch „BE / Ziel“); `currentRMultiple` bleibt in der API.
- Das ursprüngliche SL wird beim ersten Sehen gespeichert (`position_risk`); ohne bekanntes Risiko gibt es keine Level.
- Grenze: Die 2R-Prüfung nutzt die letzten 1000 gecachten 5-Minuten-Kerzen (ca. 3,5 Tage).
- Das `rMultiple` bei Schließung (Stufe 3) ist davon getrennt.
- Tabelle: Spalte „BE / Ziel“ mit 2R-Preis („BE …“, dort den SL bei BingX auf Break-even setzen) und 3R-Preis („Ziel …“, TP).
- Stand die Position beim ersten Sehen schon auf Break-even, wird der ursprüngliche SL aus der Order-Historie (`allOrders`) rekonstruiert. Das klappt nur, wenn der SL dort als eigene Stop-Order steht; sonst bleibt das Risiko unbekannt („–“).

---

## Stufe 3 – Abgeschlossene Trades im Journal

Der wertvollste Teil für die Auswertung: "Wo lag mein SL, wo wäre er besser gewesen?"

**Backend**
- Import der Order-Historie (alle Orders inkl. ausgelöster und stornierter Stop-/TP-Orders),
  zugeordnet zur Position.
- Pro abgeschlossener Position speichern: Entry(s), Exit(s), zuletzt gültiger SL und TP
  zum Zeitpunkt des Schließens, und ob SL oder TP ausgelöst hat (oder manuell geschlossen).

**Domain**
- `TradeOutcome`: STOP_LOSS_HIT | TAKE_PROFIT_HIT | MANUAL_CLOSE | LIQUIDATION
- Kennzahlen pro Trade: R-Multiple (Gewinn ÷ ursprüngliches Risiko Entry→SL),
  MAE/MFE (größter Verlust/Gewinn während der Haltedauer) aus den Kerzen.

**API**
- `GET /positions/{id}/chart` funktioniert auch für geschlossene Positionen
  (Zeitraum automatisch: Open − Puffer bis Close + Puffer).
- Felder `outcome`, `rMultiple`, `mae`, `mfe` an der Position.

**UI**
- Chart im Journal-Detail neben den Notizen.
- Ein-/Ausstiege als Marker, SL/TP als Linien nur über die Haltedauer.

**Akzeptanzkriterien**
- Für einen per SL geschlossenen Trade: Outcome = STOP_LOSS_HIT, Exit-Marker liegt am SL.
- R-Multiple stimmt mit Handrechnung an einem Fixture-Trade überein.
- Notizen bleiben vom Import unberührt.

---

## Tests (test-author)
- Mapping der BingX-Antworten für Klines und Orders mit echten Fixtures
  (Zahlen als Strings, fehlende Felder, abweichende Feldnamen).
- Zuordnung Order → Position (Symbol + Seite + Zeitraum).
- Mehrere Teil-TPs, SL nachgezogen, SL storniert.
- Laufende Kerze wird aktualisiert statt dupliziert.
- R-Multiple, MAE, MFE an konstruierten Kerzenreihen.
- Leere Antworten werden geloggt und nicht still als "erledigt" markiert.

## Ablauf mit den Agenten
Pro Stufe: architect (Schnitt + OpenAPI-Änderung) → test-author → programmer → reviewer.
Jede Stufe ist ein eigener Branch `feature/chart-stufe-<n>` und ein eigener PR.
