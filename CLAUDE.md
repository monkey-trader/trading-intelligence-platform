# CLAUDE.md – Trading Intelligence Platform

Leitlinien für Claude Code in diesem Repo. Doku, Kommentare und UI-Texte auf Deutsch.

## Ziel
Persönliches Trading-Journal mit **automatischem Import von BingX** als Kernfunktion
(alle 5 Minuten inkrementell + einmaliger Vollimport der Historie).

Fokus: **zuerst ein eigenes Backend mit sauberer API.** Clients (Web-UI, Skripte, später anderes)
sind austauschbar und sprechen nur über die API mit dem Backend.

## Grundprinzip: API-first, Client-unabhängig
- Das Backend besitzt Import, Datenbank und Fachlogik. Clients greifen **nie** direkt auf die DB zu.
- Kommunikation nur über eine HTTP/JSON-API, beschrieben als OpenAPI-Spezifikation (`docs/openapi.yaml`).
- Die OpenAPI-Spezifikation ist der Vertrag: erst die Spec anpassen, dann implementieren.
- Ein UI kommt erst, wenn Import und API stehen.

## Stack
- Backend: Node.js + TypeScript
- Datenbank: SQLite
- Tests: Vitest
- Paketmanager: **nur npm** (kein pnpm, kein yarn)
- CI: GitHub Actions (Lint + Tests bei jedem Push)

## Ordnerstruktur
```
backend/src/{domain,application,infrastructure,presentation}/<feature>/
docs/openapi.yaml
clients/        (später, z.B. clients/web/)
```

## Architektur – Onion / DDD, feature-sliced
- **Domain (Kern):** Entities, Value Objects, typisierte Errors, Factories, Repository-Ports.
  Keine Abhängigkeiten nach außen.
- **Application:** schlanke Use Cases (z.B. `ImportPositionsUseCase`), bekommen Ports injiziert.
- **Infrastructure:** BingX-Client, SQLite-Repositories, Scheduler.
- **Presentation:** HTTP-API (Routen/Controller, Request-Validierung, Mapping auf DTOs).
  Keine Fachlogik hier.
- Abhängigkeiten zeigen **immer nach innen** zum Domain-Kern.

## Referenz: bestehender Python-Importer `../bingx-journal`
Funktionierender Prototyp (Python, SQLite, Streamlit). Nicht weiterentwickeln, aber als
Referenz nutzen: `bingx.py` (Signatur-Client) und `sync.py` (Import-Logik, Schema).

Daraus übernommene BingX-Erfahrungen:
- Signatur: HMAC-SHA256 über den Query-String, Header `X-BX-APIKEY`, bei **jedem** Retry
  neuer `timestamp` und neue Signatur.
- HTTP 429 → exponentieller Backoff; `code != 0` in der Antwort → Fehler.
- BingX liefert Zahlen meist als **Strings** → defensiv parsen, Feldnamen variieren
  (mehrere Kandidaten prüfen, Rohantwort mitspeichern).
- Income (`/openApi/swap/v2/user/income`): in 7-Tage-Fenstern, max. 1000 pro Seite,
  Deduplizierung über `tranId`. Reicht nur **~90 Tage** zurück. Ältere Fenster liefern
  `data: []` oder `data: null` (gemessen am 2026-10-16). `positionHistory` reicht weiter zurück.
  Symbole, die nur vor über 90 Tagen gehandelt wurden, findet ein Income-Scan deshalb nicht.
  Lösung: Eine einmalige, fortsetzbare Suche fragt `positionHistory` für alle USDT-Kontrakte
  aus `/openApi/swap/v2/quote/contracts` ab (~1'200 Stück, erster Lauf ca. 1 h).
  Lücke: Delistete Kontrakte stehen nicht in dieser Liste und werden nicht gefunden.
- Positionen (`/openApi/swap/v1/trade/positionHistory`), am 2026-10-16 gegen die echte API gemessen:
  - `symbol`, `startTs` und `endTs` sind Pflicht, es geht nur **pro Symbol**.
  - Maximal 90 Tage pro Abfrage, sonst Fehler `109400` ("the query range is more than ninety days").
    Wir nutzen **89-Tage-Fenster**.
  - `pageIndex` zählt **ab 0**, `pageSize` gibt die Seitengröße an.
  - Gefiltert wird nach `updateTime` (Schließzeitpunkt). Teilweise geschlossene Positionen
    erscheinen erst, wenn sie ganz geschlossen sind.
  - `positionId` ist 19-stellig und wird als String behandelt, nie als Number.
- Offene Positionen (`GET /openApi/swap/v2/user/positions`, gemessen am 2026-10-16): `data` ist ein
  Array, Beträge kommen als Strings, aber `liquidationPrice` als **number**; das Eröffnungsfeld heißt `createTime`.
  Gemessen nur im Hedge-Modus (`positionSide` LONG/SHORT). Im One-Way-Modus käme `BOTH`, das lehnen
  wir derzeit ab (502).
- Fortschritt pro Datenquelle in einer State-Tabelle merken, mit 1 h Überlappung.
- Nutzer-Notizen werden vom Import **niemals** überschrieben.

### Bekanntes Problem im Prototyp
Im Prototyp ist die Tabelle `positions` **leer**, obwohl der Sync alle Symbole als erledigt
markiert und `income` 2'137 Einträge hat. Ohne Positionen können Notizen keinem Trade
zugeordnet werden.

**Ursache (bestätigt am 2026-10-16):** Der Prototyp beginnt das Paging mit `pageIndex=1`.
BingX zählt aber ab 0, deshalb wurde jede erste Seite übersprungen und die Antwort war
`code: 0` mit leerer Liste. Die 7-Tage-Fenster waren unnötig, aber nicht die Ursache.

Konsequenz für dieses Projekt:
- Erstes Feature ist der **Positions-Import**, und er muss nachweislich Daten liefern.
- Eine leere API-Antwort bei vorhandenem Income für dasselbe Symbol ist verdächtig und wird
  geloggt, nicht stillschweigend als "erledigt" markiert. Geprüft wird über den ganzen
  Zeitraum, nicht pro Fenster: Ein `REALIZED_PNL` aus einem Teilschluss kann in einem Fenster
  liegen, dessen Position erst später erscheint.
- Tests mit echten (anonymisierten) BingX-Antwortbeispielen als Fixtures.

## Vorgehen
- **TDD:** erst Test, dann Code. Tests liegen neben dem Code (`Foo.ts` → `Foo.test.ts`).
- Kein `any`; `unknown` plus Narrowing verwenden.
- Sprechende Variablennamen.
- Vor Commit: `npm run lint && npm test`.
- Commits nur auf ausdrücklichen Wunsch. Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`),
  Branches `feature/<beschreibung>`.

## Sicherheit
- BingX-API-Key **nur mit Lese-Rechten**; der Code macht nur signierte GET-Requests.
- Secrets (`BX_KEY`, `BX_SECRET`) nur in `.env`, niemals im Code oder Repo. `.env` steht in `.gitignore`.
- Die eigene API gibt niemals Secrets aus.
- Phase 1 ohne Authentifizierung: Der Server bindet nur auf Loopback (`127.0.0.1`), ein anderer
  `HOST` nur mit ausdrücklichem `ALLOW_REMOTE=true`.

## Subagents (`.claude/agents/`)
- `architect` – prüft Schichtengrenzen und den API-Vertrag
- `test-author` – schreibt Tests zuerst
- `programmer` – macht Tests grün, ohne sie zu ändern
- `reviewer` – Qualität und Sicherheit am Schluss

Ablauf pro Feature: architect (Schnitt) → test-author → programmer → reviewer.
