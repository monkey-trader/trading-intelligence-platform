---
id: TASK-16
title: 'Positionstabelle: aktuellen Kurs (Mark Price) anzeigen'
status: In Progress
assignee: []
created_date: '2026-10-25T15:08:00Z'
updated_date: '2026-10-30T20:24:00Z'
labels:
  - web
dependencies: []
references:
  - docs/design/TASK-16-kurs-spalte/prototype.html
  - docs/design/TASK-16-kurs-spalte/flow.png
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

2026-10-30 architect: Anforderungs-Check ohne Widerspruch. /quote/ticker liefert lastPrice, kein markPrice; Vorschlag Spalte = lastPrice (Feld price), kein 2. Request. Schnitt: domain/price (Quote, Port PriceSource), application/price (StreamPricesUseCase + Broker), infrastructure/bingx TickerSource, presentation GET /prices/stream (SSE, Heartbeat 15 s; WriteTimeout prüfen). AC5: Reihenfolge beim Klick einfrieren. Offen: Nutzer bestätigt Kurs-Quelle und Abstand-Definition für domain.md.

2026-10-30 Nutzer entscheidet: Kurs = lastPrice (Feld price); Abstand = (Level − Kurs)/Kurs in %, vorzeichenbehaftet; AC5 Reihenfolge beim Klick einfrieren.

2026-10-30 Lead: docs/openapi.yaml (GET /prices/stream, PriceSnapshot, PositionPrice, Tag Kurse) und docs/domain.md (Abschnitt Live-Kurs) zuerst angepasst; Spec lädt, generate:api ok. contracts: nur coverage-Tests rot (kein Vertragstest für streamPrices) – Aufgabe des test-author.

2026-10-30 test-author Runde 1: contract prices.contract.test.ts, pricedistance_test.go + Stub, broker_test.go + Stub (rot, erwartet). Nutzer entscheidet Vorzeichen: Gewinnrichtung (positiv = Level liegt in Gewinnrichtung vor dem Kurs, LONG/SHORT gleich); Funktion braucht side, bei SHORT Vorzeichen drehen. pricedistance_test.go ist anzupassen.

2026-10-30 programmer Runde 1: DistanceToLevelPercent(side,…) und generischer Broker[T] grün (go test -race, golangci-lint 0 Befunde, vom Lead nachgeprüft). Lead: ungenutzten Parameter in broker_test.go (revive) zu _ geändert. Prüfreihenfolge: side, Kurs, dann level nil. Offen: Runde 2 (Ticker-Adapter, Use Case/Poller, SSE-Handler) läuft beim test-author.

2026-10-30 test-author Runde 2: Tests für TickerSource, StreamPricesUseCase (Poll/Run, Backoff 1..30 s + Jitter, stale), SSE-Handler (Header, Event prices, Heartbeat 15 s, Disconnect, WriteTimeout-Aufhebung) rot aus richtigem Grund (vom Lead geprüft). Fixture fixtures/bingx/ticker.json: Feldformat entspricht der echten Antwort (am 2026-10-30 gegen /quote/ticker geprüft), Werte erfunden, DEAD-USDT und XYZ-USDT sind synthetische Grenzfälle; im Abschlussbericht ausweisen. Offen: Web-Variante (lead-designer Flow läuft).

2026-10-30 lead-designer Stufe Flow: docs/design/TASK-16-kurs-spalte/flow.excalidraw + flow.png (PNG per Pillow, der Skill-Renderer lief in einen Timeout; kleine Textabschnitte in B und im Flow-Knoten 'Klick Kopf'). Nutzer wählt Variante A (Kurs-Spalte nach Symbol, 24h-% darunter, Abstand als 2. Zeile in BE-/TP1-Zellen) und für Stream-Ausfall: gedämpft + Text 'veraltet' pro Zelle. Abstand nicht einzeln sortierbar. Nächster Schritt: Prototyp (prototype.html).

2026-10-30 lead-designer Stufe Prototyp: docs/design/TASK-16-kurs-spalte/prototype.html fertig, wartet auf Freigabe durch den Nutzer. Playwright: Konsole ohne Fehler, kein horizontales Scrollen bei 1280/390 px, Kontrast alle >= 4.5:1. Nicht geprüft: Drag mit der Maus. Offen: 'veraltet' je Zelle zusätzlich zum Banner; Abstand in Prozent (nicht R).

2026-10-30 programmer Runde 2 + Lead-Nachzieher: TickerSource, StreamPricesUseCase, SSE-Handler GET /prices/stream, Verdrahtung in cmd/ (Route nur mit BingX-Keys). Lead-Korrekturen: TestPriceStreamSurvivesServerWriteTimeout hatte einen Testfehler (Leseschleife brach am ersten Event ab; reproduziert), Schleife korrigiert; Lint in Testdateien behoben (errcheck, Builtin real, ungenutzte Funktion text); BackoffMin/Max unexportiert und kommentiert; PriceSnapshot mit nolint:revive (Name laut Vertrag). Gate grün: gofmt, go vet, golangci-lint 0, go test -race ./... Offen: Contract-Test gegen laufenden Server nicht gelaufen (Route braucht BX_KEY); Heartbeat-Zeile nicht gegen die Spec geprüft. Web wartet auf Freigabe des Prototyps.

2026-10-30 Nutzer gibt Prototyp (Variante A) frei. Entscheidungen im Prototyp: 'veraltet' je Zelle (gedämpft) plus Banner, Abstand in Prozent. Web-Umsetzung: test-author → programmer → designer → reviewer.
<!-- SECTION:NOTES:END -->
