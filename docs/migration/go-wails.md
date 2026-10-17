# Migrationsplan: Backend auf Go, Desktop-App mit Wails, Auslieferung über Homebrew

Status: Phase 0 in Arbeit (Branch `feature/go-phase-0-vertrag`)

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

## Phase 2 – Domain, BingX-Client, Import
TDD, Fixtures aus Phase 0.
- Domain: Position, Income, Value Objects, typisierte Errors (Go: eigene Error-Typen,
  geprüft mit `errors.As`).
- BingX-Client: HMAC-SHA256, Header `X-BX-APIKEY`, neuer Timestamp + Signatur pro Retry,
  Backoff bei 429, `code != 0` → Fehler, Zahlen aus Strings defensiv parsen.
- Import: Income in 7-Tage-Fenstern, Positionen pro Symbol, Deduplizierung über IDs,
  State-Tabelle mit 1 h Überlappung, Scheduler alle 5 Min.
- Leere Positions-Antwort bei vorhandenem Income → loggen, nicht als erledigt markieren.

## Phase 3 – HTTP-API und Umschalten
- Alle Endpunkte aus Phase 0 in Go.
- Vertragstest grün gegen **beide** Backends.
- UI testweise auf das Go-Backend umstellen (`API_BASE_URL`), eine Woche parallel nutzen.

## Phase 4 – Daten übernehmen
- Gleiches SQLite-Schema oder Migrationsskript von der bestehenden DB-Datei.
- Schema-Migrationen versioniert (z.B. `schema_version`-Tabelle), Notizen-Tabelle
  wird nie vom Import beschrieben.
- Speicherort: `~/Library/Application Support/tip/journal.db`.

## Phase 5 – Aufräumen
- TS-Backend löschen, `backend-go/` → `backend/`.
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
