---
id: TASK-16
title: 'Positionstabelle: aktuellen Kurs (Mark Price) anzeigen'
status: In Progress
assignee: []
created_date: '2026-10-25T15:08:00Z'
updated_date: '2026-10-25T15:33:00Z'
labels:
  - web
dependencies: []
priority: medium
type: enhancement
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Seit TASK-7 zeigt die Tabelle Einstieg, SL, BE und TP1, aber nicht den aktuellen Kurs. markPrice liegt schon in API (docs/openapi.yaml) und Domain (OpenPosition), wird aber nirgends angezeigt; die frühere Mark-Spalte entfiel. Ohne Kurs lässt sich der Abstand zu BE und TP1 nicht erkennen. Heute kommt er nur per Polling der Positionsliste (pollIntervalMs), also nicht live.

Recherche (2026-10-25, gegen die echte BingX-API gemessen, alles öffentlich, ohne Key):
- WebSocket wss://open-api-swap.bingx.com/swap-market: Antworten GZIP-komprimiert, Ping/Pong per Text, Subscribe {id, reqType:'sub', dataType:'BTC-USDT@markPrice'}. In 10 s kamen markPrice ~9, ticker ~10, lastPrice ~23 Nachrichten. markPrice-Payload: {e,E,s,p}; lastPrice: {s,c}.
- WS-Ticker (@ticker) ist für die 24h-Änderung unbrauchbar: p/P (0.02 %) bezog sich auf ein Fenster von wenigen Minuten (openTime O nur 3 min her), REST lieferte 1.30 %. Passt zu einem bekannten ccxt-Befund. Die 24h-Änderung kommt deshalb aus REST.
- REST GET /openApi/swap/v2/quote/ticker (ohne symbol: alle 983 Kontrakte in einem Request, ~350 KB, ~1 s): priceChangePercent, openPrice (rollendes 24h-Fenster, openTime = jetzt minus 24 h), lastPrice. GET /quote/premiumIndex: markPrice. Rate-Limit: 500 Requests / 10 s je IP, geteilt über alle Markt-Endpunkte (Header x-ratelimit-requests-remain).
- Der Icon-Anbieter CoinGecko taugt nicht für Live-Kurse: Free-Plan nur ca. 30 Aufrufe/min, WebSocket erst im bezahlten Plan (Basic+, Beta). BingX ist die richtige Quelle: selbe Börse wie die Positionen, damit passen Kurs und PnL zusammen.
- Farbe wie an den Börsen: nach 24h-Änderung (BingX: rollend 24 h, nicht UTC-Tag), grün positiv, rot negativ; dazu +/−-Vorzeichen, damit es nicht nur an der Farbe hängt.

API-first: Der Browser verbindet sich nie selbst mit BingX. Das Backend holt die Kurse und liefert sie per Stream (SSE) an den Client.

Offene Entscheidung (Quelle im Backend): (1) REST-Polling alle ~2 s mit einem Request für alle Symbole, nur Standardbibliothek, ca. 5 von 500 Requests je 10 s; (2) BingX-WebSocket mit ~1 s Takt, braucht eine WebSocket-Bibliothek (Abweichung von 'nur Standardbibliothek'). Empfehlung: erst (1) hinter einem Port, (2) später austauschbar.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Die Tabelle zeigt den Kurs je Position als eigene Spalte mit reiner Zahl (kopierbar, formatiert wie der Einstieg); die Spalte steht in der Standardreihenfolge direkt nach dem Symbol
- [ ] #2 Der Kurs aktualisiert sich live ohne Neuladen der Seite (Takt höchstens 2 s); Kurse kommen vom Backend per Stream, der Browser spricht nicht mit BingX; docs/openapi.yaml wird zuerst angepasst
- [ ] #3 Farbe und Vorzeichen nach 24h-Änderung wie an den Börsen: grün und '+' bei positiver, rot und '−' bei negativer Änderung (rollendes 24h-Fenster aus BingX REST, nicht aus dem WS-Ticker); die Änderung in Prozent wird mit angezeigt, Berechnung im Backend; Farben nur über Design-Tokens, Kontrast in Hell und Dunkel
- [ ] #4 Fällt der Stream aus oder ist ein Kurs unbekannt, zeigt die Spalte '–' bzw. den zuletzt bekannten Wert mit Hinweis 'veraltet'; die Tabelle bleibt bedienbar; Reconnect mit Backoff, das BingX-Rate-Limit (500 Requests / 10 s) wird nicht überschritten
- [ ] #5 Der Kurs ist sortierbar (Zyklus aufsteigend, absteigend, API-Reihenfolge, aria-sort) und per Alt+Pfeil/Drag verschiebbar; Symbol bleibt links; die Sortierung springt bei Live-Ticks nicht ständig um (Kursstand beim Klick einfrieren oder entprellen, im Plan entscheiden)
- [ ] #6 Bei 1280 px bleibt die Tabelle ohne horizontales Scrollen (lange Symbole, große Zahlen); mobil Karte mit Spaltenname aus data-label; Ticks ändern die Spaltenbreite nicht (tabellarische Ziffern); Screenreader werden nicht bei jedem Tick angesprochen
- [ ] #7 Abstand des Kurses zu BE und TP1 (Prozent oder R) wird im Backend berechnet und von der API geliefert, der Client zeigt ihn nur an
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Entscheidung 2026-10-25 (Nutzer): Quelle Variante 1, REST-Polling alle ~2 s, ein Request /quote/ticker für alle Symbole, nur Standardbibliothek, hinter einem Port (später durch WebSocket austauschbar).
1. architect: Schnitt (Port PriceSource, Use Case, SSE-Route), Rate-Limit-Budget, Sortierung bei Ticks (einfrieren oder entprellen).
2. docs/openapi.yaml zuerst: SSE-Stream der Kurse (symbol, price, change24hPercent) und Abstand zu BE/TP1; Contract-Test.
3. test-author, programmer Backend: BingX-Ticker-Adapter (Fixture aus echter Antwort), Fan-out an Abonnenten, Reconnect/Backoff.
4. ui-prototyping/lead-designer: Kurs-Spalte (Farbe nach 24h, Vorzeichen, tabellarische Ziffern, Stream-Ausfall), Freigabe durch Nutzer.
5. test-author, programmer Web: Stream-Hook, Spalte price direkt nach Symbol, Sortierung, API-Typen per generate:api.
6. designer, reviewer, CI-Liste, unify.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-10-25: Quelle entschieden (Variante 1). Noch nichts umgesetzt; Start mit architect-Schnitt und openapi.yaml.
<!-- SECTION:NOTES:END -->
