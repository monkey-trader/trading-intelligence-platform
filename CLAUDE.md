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
- Das Web-UI (`clients/web/`) zeigt nur, was die API liefert; Fachlogik und Berechnungen
  (z.B. Summen, Liquidationsabstand) gehören ins Backend.

## Stack
- Backend: Go (`backend/`, nur Standardbibliothek; das frühere TypeScript-Backend ist entfernt,
  Migration laut `docs/migration/go-wails.md`)
- Web-Client: Vite + React + TypeScript, CSS Modules
- Datenbank: SQLite
- Tests: Vitest
- Paketmanager: **nur npm** (kein pnpm, kein yarn)
- CI: GitHub Actions (Lint + Tests bei jedem Push)

## Ordnerstruktur
```
backend/                Go-Backend: cmd/trading-intelligence/, internal/{domain,application,
                        infrastructure,presentation}/<feature>/; Befehle: `go test -race ./...`,
                        `golangci-lint run ./...`, `gofmt -l .` (muss leer sein)
docs/openapi.yaml       API-Vertrag (Quelle der Wahrheit, es gibt nur diese eine Spec)
docs/database/schema.md SQLite-Schema und State-Keys
contracts/         Vertragstest gegen eine laufende API (`CONTRACT_BASE_URL=… npm test`),
                        backend-unabhängig (läuft in der CI gegen das Go-Backend)
fixtures/bingx/         anonymisierte BingX-Antworten, sprachneutral (Go-Tests: Pfad relativ zum Repo-Root)
clients/desktop/ Wails-App (macOS, Go-Modul): Fenster auf die lokale HTTP-API; bauen mit
                `wails build -skipbindings -s`, Tests `go test ./...` in diesem Ordner
clients/web/    Web-Client (Vite + React), spricht nur über die HTTP-API
                API-Typen per `npm run generate:api` aus docs/openapi.yaml (openapi-typescript 7.13.0
                verlangt TS ^5, läuft mit TS 6 über `overrides` in package.json, getestet)
```

## Lokal starten
Einmalig installieren und die Keys hinterlegen:
```bash
(cd clients/web && npm ci)
export BX_KEY=… BX_SECRET=…   # Key nur mit Lese-Rechten; das Backend liest keine .env-Datei
```
Dann zwei Terminals:
```bash
cd backend && go run ./cmd/trading-intelligence   # Terminal 1: API auf http://127.0.0.1:3000
cd clients/web && npm run dev    # Terminal 2: Web-UI, Proxy /api → Backend
```
Danach http://127.0.0.1:5173 öffnen (Vite bindet nur auf IPv4-Loopback; `localhost` kann auf `::1` zeigen).

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
- Tests mit echten (anonymisierten) BingX-Antwortbeispielen als Fixtures (`fixtures/bingx/`).

## Vorgehen
- **TDD:** erst Test, dann Code. Tests liegen neben dem Code (`Foo.ts` → `Foo.test.ts`).
- Kein `any`; `unknown` plus Narrowing verwenden.
- Sprechende Variablennamen.
- Vor Commit: `npm run lint && npm test` (in `contracts/` ohne laufenden Server `npm run test:unit`).
- Commits nur auf ausdrücklichen Wunsch. Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`),
  Branches `feature/<beschreibung>`.
- **Releases sind automatisch:** Nur ein Push auf `main` löst über `.github/workflows/release.yml` (semantic-release, `.releaserc.json`) einen Release aus; die Version ergibt sich aus den Commit-Präfixen (`feat` → minor, `fix`/`perf` → patch, `!` oder `BREAKING CHANGE:` bei jedem Typ → major; `docs`/`chore`/`test`/`ci`/`refactor`/`style` lösen **keinen** Release aus). Tags nie von Hand setzen. Squash-Merge-Titel müssen Conventional Commits sein.

## Sicherheit
- BingX-API-Key **nur mit Lese-Rechten**; der Code macht nur signierte GET-Requests.
  „Nur GET“ gilt für BingX. Die eigene API hat einen Schreib-Endpoint (`PUT`/`DELETE /positions/{positionId}/initial-stop-loss`,
  nur Loopback, ohne Auth; mit `ALLOW_REMOTE=true` nicht registriert) und schreibt nur in die eigene Datenbank.
- Secrets (`BX_KEY`, `BX_SECRET`) nur in `.env`, niemals im Code oder Repo. `.env` steht in `.gitignore`.
- Die eigene API gibt niemals Secrets aus.
- Phase 1 ohne Authentifizierung: Der Server bindet nur auf Loopback (`127.0.0.1`), ein anderer
  `HOST` nur mit ausdrücklichem `ALLOW_REMOTE=true`.

## Subagents (`.claude/agents.txt`)
Generische Agents (Onion, DDD, TDD) aus ai-os, verlinkt mit `ai-link`. Projektspezifisches für die Agents
steht in `.claude/agent-rules.md` (BingX, Pfade, Befehle, Testschwerpunkte), nicht in den Agents selbst.
- `architect` – prüft Schichtengrenzen und den API-Vertrag
- `test-author` – schreibt Tests zuerst
- `programmer` – macht Tests grün, ohne sie zu ändern
- `designer` – Routine-Checks für Gestaltung und Konsistenz des UI in `clients/` (Quelle: `docs/design.md`)
- `lead-designer` – neue Designs, Varianten, Umgestaltung (setzt die designer-Regeln voraus)
- `reviewer` – Qualität und Sicherheit am Schluss

Alle Agents laden den Skill `token-saving` vor.

Ablauf pro Feature: architect (Schnitt) → test-author → programmer → reviewer.
Bei sichtbaren UI-Änderungen: … → programmer → designer → reviewer.

## Skills (`.claude/skills.txt`)
Skills (und Agents) liegen zentral in ai-os (`~/git/ai-c3smonkey/ai-os/ai/skills`). `.claude/skills.txt` listet,
welche dieses Repo nutzt; `ai-link` legt die Links in `.claude/skills/` an (lokal, nicht versioniert).
Neuer Skill: in ai-os anlegen, Zeile in `skills.txt` ergänzen, `ai-link` ausführen.

## Planung (vor größeren Änderungen lesen)
- `docs/migration/go-wails.md` – Migration des Backends auf Go, Desktop-App mit Wails,
  Auslieferung über Homebrew. Das Go-Backend liegt in `backend/`.
- `docs/features/chart-journal.md` – Feature: Chart mit Entry, SL, TP (Stufen 1–3, read-only).


## Code finden (repo-rag)
Bevor du Ordner durchsuchst oder Dateien breit liest:
`repo-rag search --files-only "<konkrete englische Frage mit Code-Begriffen>"`
Beispiel: `repo-rag search --files-only "fetch open positions from bingx api"`
Lies danach nur die gefundenen Dateien/Zeilen. Optional `--path backend` zum Eingrenzen.
Nach grösseren Änderungen: `repo-rag index`.


- **ask-local nur für EINE Quelle** (ein Log, ein Diff, eine Datei). Nicht für Vergleiche mehrerer Quellen oder Status-Fragen (z. B. Plan vs. Commits) – das machst du selbst, `git log --oneline -20` ist billig.
- `ask-local` nutzt standardmässig das schnelle deepseek-coder-v2:16b (MoE) (gut im Herauslesen: Logs, Diffs, Fixtures, Commit-Messages). Nur wenn Verständnis/Gewichtung nötig ist: `ask-local --deep` (14b, ~2× langsamer).
- Erscheint `[ask-local] WARNUNG: Eingabe zu lang`, Ergebnis nicht trauen: Eingabe eingrenzen und neu fragen.

## Agenten-Orchestrierung (Token-Regeln für den Lead)

**Starten und Übergeben**
- Subagenten **immer neu starten, nie resumen** (ein Resume schleppt den ganzen Verlauf mit).
- Jedem Agenten eine kurze Übergabe mitgeben: betroffene Dateien/Pakete, betroffene Tests, genaue Aufgabe,
  was er NICHT anfassen darf. Er soll nicht selbst das Repo durchsuchen müssen.
- Agenten berichten in höchstens 15 Zeilen: geänderte Dateien, Ergebnis (grün/rot), offene Punkte. Keine Logs.

**Wann welcher Agent**
- `architect`: nur bei neuen Features/Phasen oder Schichtfragen, nicht bei Nachziehern oder Bugfixes.
- `test-author` → `programmer`: je eine Runde. Bleiben Tests rot, meldet der programmer den Grund,
  statt dass die Kette erneut läuft.
- `reviewer`: **genau einmal, ganz am Ende**, vor dem Commit.
- `designer`: nur bei sichtbaren UI-Änderungen. `lead-designer`: nur für neue Designs oder Varianten.
- Kleine Nachzieher (Review-Punkte, Kommentare, Konstanten, einzelne Tests) erledigt der Lead bzw.
  ein einzelner programmer direkt, ohne die ganze Kette.

**Vor jedem Commit (Pflicht)**
- Go: `cd backend && test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...`
- `golangci-lint` läuft lokal (go install, built with go1.27). „Läuft erst in der CI“ ist kein gültiger Grund mehr.
  Schlägt Lint fehl, wird vor dem Commit behoben, nicht nachgezogen.

**Session klein halten**
- Vor dem reviewer `/compact` vorschlagen, wenn die Session lang ist.
- Nach jedem Commit den Abschnitt „Aktueller Stand“ in `docs/migration/go-wails.md` aktualisieren
  und dem Nutzer `/clear` empfehlen.
