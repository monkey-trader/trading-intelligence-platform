# Migrationsplan: Backend auf Go, Desktop-App mit Wails, Auslieferung über Homebrew

Status: Phase 0 bis 5 und 7 (Server) erledigt, Release automatisch über semantic-release; offen: Phase 6 (Wails)

## Aktueller Stand (2026-10-20)
- **Erledigt:** Phase 0 bis 2d (Vertrag, Gerüst, offene Positionen, Positions-Import, SQLite/State, Income-Import).
- **2e:** Scheduler (5 Minuten) und einmaliger Vollimport sind in `main.go` verdrahtet, nur mit gültigen Keys.
- **3:** Vertragstest läuft grün gegen das Go-Backend, der UI-Proxy ist umschaltbar.
- **4:** `schema_version` (Version 1), Sicherung `<datei>.bak-v<alt>-<UTC>` vor Migration (0600, `O_EXCL`),
  Ablehnung zu neuer DBs (`SchemaTooNewError`), Standardpfad `<UserConfigDir>/tip/journal.db` über
  `config.ReadDatabasePath` (`JOURNAL_DB` überschreibt). Details in `docs/database/schema.md`.
- **Phase 7 (Server):** `.goreleaser.yaml` + `.github/workflows/release.yml` (Push auf `main` → semantic-release berechnet die Version aus Conventional Commits, `feat` → minor, `fix`/`perf`/`chore`/`docs`/`refactor`/`test`/`ci`/`style` → patch, `!` oder `BREAKING CHANGE:` (bei jedem Typ, auch `chore!:`) → major; legt Tag + GitHub Release an, danach Build im selben Workflow + Formula im Tap `monkey-trader/homebrew-tap`, Secret `TAP_GITHUB_TOKEN`; Konfiguration `.releaserc.json`; Build eines bestehenden Tags erneut per `workflow_dispatch`). Keys für den Dienst in `~/.config/tip/env`. Cask `tip-app` folgt mit Phase 6.
- **Web-UI:** wird per `go:embed` im Go-Server unter `/` ausgeliefert (Release-Build baut `clients/web` mit `VITE_API_BASE=''`); `/health`, `/positions`, `/api` bleiben API.
- **Release (2026-10-20):** Aktuell `v1.0.0` (Stand Tags; die Zeile unten beschreibt die Einrichtung bei `v0.1.5`), installiert per `brew upgrade tip-server && brew services restart tip-server` (Health-Check `http://127.0.0.1:3000/health`). Releases entstehen nur noch automatisch auf `main` (semantic-release per `npx`, Node aus `.nvmrc`, Preset `conventional-changelog-conventionalcommits@8`, weil neuere Presets Writer 9 brauchen). Regeln in `.releaserc.json`: Breaking → major, `feat` → minor, `fix`/`perf`/`chore`/`docs`/`refactor`/`test`/`ci`/`style` → patch. Tag, Build und Tap-Update laufen im selben Workflow (Tags vom `GITHUB_TOKEN` lösen keinen zweiten aus); Runner `ubuntu-24.04`. Fehlgeschlagenen Build eines Tags wiederholen: `workflow_dispatch` mit Tag.
- **Chart Stufe 1 (2026-10-18, in `main` per #12):** `GET /positions/{positionId}/chart` (nur offene Positionen) liefert Kerzen
  (öffentlicher Klines-Endpunkt v3, absteigend geliefert, im Backend aufsteigend sortiert; Cache-Tabelle `candles`), Entry sowie SL/TP aus
  `openOrders` (Zuordnung über `positionID`, 19-stellig, verlustfrei gelesen) mit `distanceRatio`. Web-UI: Chart-Button je Zeile,
  `PositionChartPanel` mit Lightweight Charts. Gemessen: `allOrders` erlaubt nur 7-Tage-Fenster (für Stufe 3). Offen: Stufe 2 (Polling,
  Theme-Wechsel im Chart-Adapter), Stufe 3 (geschlossene Trades, Order-Historie).
- **Stufe 2 + SL/TP/BE (2026-10-20, v0.4.0):** Chart-Polling alle 5 s (Pause bei hidden, Theme-Wechsel). `GET /positions/open` liefert `stopLoss`, `takeProfit`, `breakEven` (nullable, `null` bei Orders-Fehler); BE = SL nicht im Verlust (LONG SL>=Entry, SHORT SL<=Entry), Regel in `domain/position/breakeven.go`. Orders-Source mit 5-s-Cache (`infrastructure/chart/caching.go`) für Liste und Chart. Tabelle: Spalten SL/TP, Badge „BE“, Hinweis „kein SL“.
- **v1.0.0 (2026-10-20, Breaking):** `liquidationPrice` und `liquidationDistanceRatio` sind aus `GET /positions/open` entfernt (Rohfeld `LiquidationPrice` bleibt im Domain-Entity/BingX-Mapper). Tabelle: Spalten per Drag & Drop oder Alt+Pfeil umordnen, Reihenfolge in `localStorage` (`tip_open_positions_column_order_v1`), „Spalten zurücksetzen“, Symbol-Spalte fest vorn. Offen: Horizontal-Scroll bei 1280 px, Negativ-Cache für Orders-Fehler, Zeilen-Sortierung per Klick, Chart Stufe 3.
- **Positionstabelle im BingX-Stil + Coin-Icons (2026-10-20, Branch `feature/bingx-positionstabelle`):** Tabelle mit 7 Spalten (Symbol, Positionswert, Margin, Unreal. PnL mit Prozent, Real. PnL, Einstieg, TP / SL), Symbol-Zelle mit Icon, Anzeigename und Chips (Long/Short, Isolated/Cross, Hebel), BE-Badge und „kein SL“ in der SL-Zeile; Spaltenreihenfolge neu unter `tip_open_positions_column_order_v2`; Kopfzeile „Trading Intelligence Platform“. Neu: `GET /assets/icons?symbols=…` (Vertrag 0.5.0): Basis-Asset und `displayName` je Symbol, Icon-URL über CoinGecko `coins/markets` (ein Abruf je Batch, bester `market_cap_rank`, nur https + Host `coin-images.coingecko.com`), In-Memory-Cache (24 h Treffer, 10 min kein Treffer, Quellenfehler werden nicht gecacht und ergeben 200 mit `iconUrl: null`). Rohstoff-Kontrakte `NCCO<Name>2USD` werden normalisiert (`NCCOGOLD2USD` → XAU „Gold“, `NCCOXAG2USD` → XAG „Silber“, sonst z.B. „Nickel“), gehen nie an CoinGecko; Gold und Silber zeigt der Client mit lokalen SVGs. Schichten: `domain/asset`, `application/asset`, `infrastructure/{coingecko,asset}`, `presentation/httpapi/asseticons.go`. Fixtures: `fixtures/coingecko/`. Offen: Cache ohne Obergrenze (Eingabe ist auf 50 Symbole je Anfrage begrenzt), Test für Backslash-Hosts in `NewIconURL`, `1000PEPE`-artige Präfixe werden bewusst nicht umgedeutet.
- **UI (2026-10-18):** Margin-Spalte in der Positionstabelle; neues Standard-Design „Pro“ im Kraken-Stil (#11). Agents `lead-designer` (neue Designs) und `designer` (Checks).
- **Offen:** Binary-Größe messen, Sortierung der Income-Antwort gegen die echte API belegen, Homebrew/Wails (Phase 5+).
- **Importbeginn:** Der Import startet ab dem ersten Serverstart (State `import_started_at`, `ResolveImportStart`), keine Historie, keine Vollsuche über alle Kontrakte. Bei jedem Lauf (also auch beim Start) werden die aktuell offenen Positionen als Zeile in `positions` angelegt (`close_time` 0, `fully_closed` 0), damit sie im Journal verwaltet werden können; ihre Symbole gehen in den Closed-Import. Schließt eine Position, überschreibt der Upsert dieselbe Zeile, Notizen bleiben.
- **Zurückgestellt:** `RunFullSearch`/`RunFullSearchUntilDone` (Historie nachladen) existieren, sind aber nicht verdrahtet; später als Nutzeraktion.
- **Phase 5 erledigt (2026-10-18):** Das TS-Backend ist gelöscht, `backend-go/` heißt jetzt `backend/` (Go-Modul `…/trading-intelligence-platform/backend`). Pfade `backend-go/` in den Phasen unten sind historisch. CI: ein Job `Go-Backend`, `Vertragstest` ohne Matrix (plus OpenAPI-Lint). Die alte TS-SQLite (`data/journal.sqlite`) wurde ausserhalb des Repos gesichert.

## Zielbild

```
brew install monkey-trader/tap/tip-server        # Go-Backend (Formula)
brew services start tip-server                   # launchd-Dienst: Import alle 5 Min, HTTP-API auf localhost
brew install --cask monkey-trader/tap/tip-app    # Wails-App (Cask): Fenster auf die API
```

- **tip-server (Go):** Import, SQLite, Fachlogik, HTTP-API gemäß `docs/openapi.yaml`.
  Läuft im Hintergrund, auch wenn die App geschlossen ist.
- **tip-app (Wails):** natives macOS-Fenster mit dem bestehenden Web-UI.
  Reiner Client der HTTP-API (keine Wails-Direktaufrufe in Fachlogik → API-first bleibt).
- **Kein Hosting, keine Cloud-DB.** Alles lokal auf dem Mac.

Warum Go: eine einzige Binary ohne Runtime, SQLite in reinem Go (`modernc.org/sqlite`,
kein CGO), sparsam als Hintergrunddienst, Tests eingebaut.

## Branch-Strategie

**Kein langlebiger Migrations-Branch.** Stattdessen läuft das Go-Backend eine Zeit lang
**parallel** im Ordner `backend-go/` auf `main`, bis es den gleichen Stand hat wie das
TypeScript-Backend (Strangler-Pattern).

- Pro Phase ein kurzer Branch `feature/go-<phase>` und ein PR.
- `main` bleibt jederzeit lauffähig, das UI funktioniert durchgehend.
- Umschalten zwischen den Backends über die Basis-URL im UI (`API_BASE_URL`).
- Erst wenn beide Backends den Vertragstest bestehen, wird das TS-Backend gelöscht.

Vorteil gegenüber einem großen Branch: keine Merge-Hölle, kleine Reviews, jederzeit zurück.

---

## Phase 0 – Bestandsaufnahme und Vertrag einfrieren
- Alle bestehenden Endpunkte des TS-Backends in `docs/openapi.yaml` vollständig beschreiben
  (inkl. Fehlerantworten).
- **Vertragstest** schreiben, der gegen eine laufende API-Basis-URL läuft und die
  Antworten gegen die OpenAPI-Spec prüft. Läuft zuerst gegen das TS-Backend → grün.
- Echte BingX-Antworten (anonymisiert) als Fixtures unter `fixtures/bingx/` ablegen,
  sprachneutral als JSON, damit beide Backends sie nutzen.
- Aktuelles SQLite-Schema dokumentieren.

Ergebnis: ein Vertrag, an dem das Go-Backend gemessen wird.

Umsetzung (2026-10-17):
- Spec 0.4.0: allgemeine Regeln in `info.description` (Host-Prüfung vor Routing, 404 `not-found`
  auch für `HEAD` und fremde Methoden, Fehler immer `problem+json`, `Retry-After` nur bei rate-limited).
  Das TS-Backend liefert 404/500 jetzt ebenfalls als Problem.
- `contract-tests/`: eigenes npm-Paket, validiert Antworten mit Ajv gegen die Spec; statischer Check,
  dass jede `operationId` einen Vertragstest hat. CI-Job `contract` startet das TS-Backend ohne Keys.
- `fixtures/bingx/` mit README, `docs/database/schema.md`.
- Offen für Phase 3: Socket-Fehler vor dem HTTP-Parsing (z.B. 431, Header zu groß) liefert Node
  ohne Problem-Format; nicht im Vertrag.
- Offen für Phase 3: Der 200-Pfad von `/positions/open` sowie 502/504/rate-limited werden in CI nicht
  live geprüft. Idee: `BX_BASE` auf einen lokalen Fixture-Server zeigen lassen (BingX-Stub).
- Bekannt: Die Fastify-Response-Schemas sind eine Handkopie der Spec; Fastify entfernt zusätzliche
  Felder, der Vertragstest ist das Sicherheitsnetz.

## Phase 1 – Go-Gerüst
- `backend-go/` mit Go-Modul, Onion-Struktur:
  ```
  backend-go/
  ├── cmd/tip-server/main.go
  └── internal/{domain,application,infrastructure,presentation}/<feature>/
  ```
- Werkzeuge: `go test`, `golangci-lint`, `gofmt`.
- HTTP: Standardbibliothek (`net/http`) oder Server-Code aus der OpenAPI generieren
  (z.B. oapi-codegen) – architect entscheidet.
- GitHub Actions: zusätzlicher Job für `backend-go` (lint + test).
- `GET /health` implementiert, Vertragstest dafür grün.

Umsetzung (2026-10-17):
- Nur Standardbibliothek (`net/http`, `log/slog`), kein oapi-codegen: zwei Endpunkte, volle Kontrolle
  über 404/HEAD/Host-Prüfung. HTTP-Paket `internal/presentation/httpapi`.
- Eigener Router statt `ServeMux`-Methodenabgleich (der beantwortet HEAD automatisch und liefert 405).
- Ungültige Request-Ziele (`/%zz`) lehnt `net/http` vor jedem Handler als Klartext ab. Lösung:
  Listener-Wrapper liest die erste Zeile jeder Verbindung, ersetzt ein ungültiges Ziel durch `/` und
  setzt ein Flag (`atomic.Bool` über `ConnContext`); der Handler prüft zuerst den Host und liefert
  dann 400 `bad-request`. Dafür ist Keep-Alive aus (eine Anfrage pro Verbindung, auf Loopback vertretbar).
- `/positions/open` ist bis Phase 2 ein Stub mit 503 `not-configured`; der Vertragstest läuft ohne
  Keys daher vollständig grün. Muss in Phase 2 ersetzt werden.
- `go.mod`: `go 1.27.0`, `toolchain go1.27.2`. golangci-lint muss mit mindestens derselben Go-Version
  gebaut sein (CI: v2.15.0); ein älteres lokales golangci-lint bricht mit einer Versionsmeldung ab.
- CI: Job `backend-go`, Vertragstest als Matrix gegen TS- und Go-Backend.
- Bekannte Abweichungen vom TS-Backend (Go antwortet selbst, vor jedem Handler, als Klartext):
  HTTP/1.1 ohne Host, doppelter oder ungültiger Host-Header, kaputte Header (400), zu große Header (431).
  Strenger als TS: absolute-form-Ziele (`GET http://host/x`) ergeben 403. `PORT` nur als ganze
  Dezimalzahl (TS akzeptiert über `Number()` auch `1e3`, `0x10`, `3000.0`).
- Weitere Unterschiede, Go jeweils gleich gut oder besser: `GET /health#x` → 404 (TS 200),
  nacktes LF als Zeilenende wird akzeptiert (TS 400 ohne Problem), Leerzeile vor der Request-Zeile
  → Klartext-400 (TS problem+json), kleingeschriebene Methode → 404 problem+json (TS 400 ohne Problem).
- Phase 2a (offene Positionen) – bewusste Abweichungen vom TS-Backend:
  - `BX_BASE` wird geprüft (https immer, http nur für Loopback, keine Zugangsdaten/Query/Fragment in
    der URL); TS prüft nicht. Nach dem Go-Stand sollte TS angeglichen werden.
  - Der Go-Client folgt keinen Redirects (sonst ginge `X-BX-APIKEY` ans Ziel) und begrenzt den
    Antwort-Body auf 10 MiB.
  - Eine gemeinsame Abfrage im Cache hat zusätzlich eine Gesamtfrist (`FetchTimeout` 8 s, unter
    dem `WriteTimeout` von 10 s), sonst würde die Antwort ohne Problem-Body abgeschnitten.
  - `liquidationPrice` als JSON-Zahl: gleiches Ergebnis wie TS (`String(n)`), auch bei Exponent
    (`1e-7` → `0.0000001`). Als String mit Exponent bleibt es ein Fehler (wie TS).
  - Zeitfelder (`createTime`/`updateTime`) nur als Ganzzahl.

## Phase 2 – Domain, BingX-Client, Import
TDD, Fixtures aus Phase 0.
- Domain: Position, Income, Value Objects, typisierte Errors (Go: eigene Error-Typen,
  geprüft mit `errors.As`).
- BingX-Client: HMAC-SHA256, Header `X-BX-APIKEY`, neuer Timestamp + Signatur pro Retry,
  Backoff bei 429, `code != 0` → Fehler, Zahlen aus Strings defensiv parsen.
- Import: Income in 7-Tage-Fenstern, Positionen pro Symbol, Deduplizierung über IDs,
  State-Tabelle mit 1 h Überlappung, Scheduler alle 5 Min.
- Leere Positions-Antwort bei vorhandenem Income → loggen, nicht als erledigt markieren.

Aufteilung in Schritte (je eigener Branch, architect → test-author → programmer → reviewer):
- **2b – Geschlossene Positionen ohne DB:**
  - Domain: `ClosedPosition` (primitive Felder, `positionId` als String), Factory auf den vorhandenen
    Value Objects, Fehler `SuspiciousEmptyPositionHistoryError`; Ports `ClosedPositionSource`,
    `PositionRepository`, `IncomeLookup`.
  - BingX-Client: `positionHistory` pro Symbol, `pageIndex` ab 0, ein Aufruf deckt höchstens 89 Tage ab
    (größere Bereiche lehnt der Client ab), Seiten kommen als Liste von Roh-Objekten zurück.
  - Use Case `ImportClosedPositionsUseCase` fenstert in 89-Tage-Fenster. Pro Fenster meldet er
    `Imported` oder `Suspicious` (leer trotz Income) und liefert die verdächtigen Fenster zurück; der
    State wird erst in 2c geschrieben. Ein Fehler bricht den Import ab (Anzahl bisher importierter
    Positionen bleibt im Ergebnis), Wiederholen entscheidet später der Scheduler.
  - Infrastruktur: In-Memory-Repository und ein `IncomeLookup` mit festem `false`. Die Verdachtsregel ist
    damit im Betrieb noch wirkungslos, bis 2c/2d den echten Lookup liefern.
  - Als Erstes prüft der test-author Feldnamen, Antwortschlüssel und Zeitregeln (`updateTime`/`closeTime`,
    Leverage, `closeTime >= openTime`) gegen `fixtures/bingx/` und `docs/database/schema.md`.
- **2c – SQLite und State:** Schema mit `schema_version`, Repositories für `positions`, `income`, `state`,
  `balance`; `notes` wird nie vom Import beschrieben. Offen: SQLite-Treiber (Empfehlung `modernc.org/sqlite`,
  ohne cgo).
- **2d – Income-Import:** 7-Tage-Fenster, max. 1000 pro Seite, Deduplizierung über `tranId`, Fortschritt
  mit 1 h Überlappung.
- **2e – Scheduler und Vollimport:** Lauf alle 5 Min mit Überlappungsschutz, fortsetzbare Symbolsuche über
  `/quote/contracts`, Verdrahtung in `main.go` nur mit gültigen Keys.

## Phase 3 – HTTP-API und Umschalten
- Alle Endpunkte aus Phase 0 in Go. Das TS-Backend hat nur `/health` und `/positions/open`, beide sind in Go
  vorhanden; es gibt keine weiteren Endpunkte nachzuziehen.
- Vertragstest grün gegen **beide** Backends (Go am 2026-10-18: 101/101, Start ohne Keys, Port 3100).
- UI testweise auf das Go-Backend umstellen: `BACKEND_URL=http://127.0.0.1:3100 npm run dev` in `clients/web`,
  eine Woche parallel nutzen.

## Phase 4 – Daten übernehmen
- Gleiches SQLite-Schema oder Migrationsskript von der bestehenden DB-Datei.
- Schema-Migrationen versioniert (z.B. `schema_version`-Tabelle), Notizen-Tabelle
  wird nie vom Import beschrieben.
- Speicherort: `~/Library/Application Support/tip/journal.db`.

## Phase 5 – Aufräumen
- TS-Backend löschen, `backend-go/` → `backend/` (**erledigt**, siehe „Aktueller Stand“).
- `CLAUDE.md` und `.claude/agents/*` anpassen:
  Backend-Befehle `go test ./...`, `golangci-lint run`; Vitest/npm gelten nur noch fürs UI.
- README aktualisieren.

## Phase 6 – Wails-App
- `clients/desktop/` mit Wails; bindet das bestehende Web-UI ein.
- App spricht per HTTP mit `tip-server` auf localhost.
- Wenn der Server nicht läuft: klare Meldung in der App ("`brew services start tip-server`").
- Wails-Version (v2 stabil vs. neuere Version) beim Setup prüfen.

## Phase 7 – Release und Homebrew
- Eigener Tap: Repo `monkey-trader/homebrew-tap`.
- **Formula `tip-server`** mit `service`-Block → `brew services start tip-server`
  richtet den launchd-Dienst ein.
- **Cask `tip-app`** für die Wails-.app.
- GitHub Actions auf Tag `v*`: Go-Binary bauen (arm64 + amd64), Wails-App auf macOS-Runner bauen,
  GitHub Release erstellen, Formula/Cask im Tap aktualisieren (z.B. mit GoReleaser).
- Ohne Apple-Developer-Account ist die App **unsigniert**: für den Eigengebrauch einmal
  freigeben (Systemeinstellungen → Datenschutz & Sicherheit → "Trotzdem öffnen").
  Signieren + Notarisieren erst, wenn andere die App nutzen sollen (99 USD/Jahr).

---

## Risiken
| Risiko | Gegenmaßnahme |
|---|---|
| Go-Backend verhält sich anders als TS | Vertragstest gegen beide, gemeinsame Fixtures |
| Daten gehen bei Umstellung verloren | Backup der DB-Datei vor Phase 4, Notizen nie vom Import beschreiben |
| Positions-Import liefert wieder still nichts | Test mit echter Fixture, Logging leerer Antworten |
| Lernkurve Go | Kleine Phasen, reviewer achtet auf idiomatisches Go |
| Unsignierte App wird blockiert | Eigengebrauch: manuell freigeben; später signieren |

## Reihenfolge mit dem Chart-Feature
Migration bis Phase 3, dann das Chart-Feature (`docs/features/chart-journal.md`) direkt in Go.
Phasen 4–7 können parallel dazu laufen.
