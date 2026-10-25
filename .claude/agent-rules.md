# Agent-Regeln: Trading Intelligence Platform

Die Agents selbst sind generisch (Onion, DDD, TDD) und kommen aus ai-os (`.claude/agents.txt`, `ai-link`).
Hier steht nur, was für dieses Projekt gilt. Jeder Agent liest `## Alle` und seinen eigenen Abschnitt.
Diese Regeln haben Vorrang vor den allgemeinen.

## Alle
- Stack: Go-Backend in `backend/` (Onion: `internal/{domain,application,infrastructure,presentation}/`,
  Verdrahtung nur in `cmd/`, HTTP-Paket heißt `httpapi`), Web-Client in `clients/web` (Vite, React,
  TypeScript, gleiche Schichten unter `src/`), Desktop-App (Wails) in `clients/desktop`.
- API-Vertrag: `docs/openapi.yaml`, Vertragstests in `contracts/`. Nur abschnittsweise lesen
  (`rg -n "<pfad oder Schema>" docs/openapi.yaml`).
- Sicherheit: Der BingX-Key hat nur Lese-Rechte, der BingX-Client macht nur signierte GET-Requests.
  Die eigene API schreibt nur in die lokale SQLite-Datenbank.
- Fachsprache: Position, Trade, Stop-Loss (SL), Take-Profit (TP), R (Risiko-Einheit), PnL, Margin, Liquidation.
- Kommentare auf Deutsch.

## architect
- Schichtregeln prüfen über Imports, z. B.
  `rg -n '"[^"]*/internal/(application|infrastructure|presentation|httpapi)' backend/internal/domain`.
- Zusätzlich `cd backend && golangci-lint run ./...` (depguard in `backend/.golangci.yml` erzwingt die Schichten).
- Ports in `domain/<feature>/interfaces/`, Adapter in `infrastructure/`. Clients nur über die API.

## test-author
- Go: Tests als `foo_test.go` neben `foo.go`, tabellengetrieben mit `t.Run`, nur Standardbibliothek
  (`testing`, `net/http/httptest`). Rot bestätigen: `cd backend && go test ./internal/<paket>/...`.
  Domain-Fehler mit `errors.As`/`errors.Is`. Stubs mit `// TODO(programmer)`.
- Fachliche Schwerpunkte im Backend:
  - Mapping der BingX-Antworten (fehlende oder abweichende Felder)
  - Deduplizierung beim Import (gleiche ID zweimal, Überlapp-Fenster), Fortschritts-Marker („importiert bis“)
  - Kennzahlen: Trefferquote, Gewinn-Verlust-Verhältnis, Gesamt-PnL
  - Fehlerfälle: leere Historie, HTTP 429, `code != 0`
  - Grenzen: z. B. genau 89 Tage und 89 Tage + 1 ms, volle Seite = pageSize
- Web: Vitest, `Foo.ts` → `Foo.test.ts` daneben, rot bestätigen mit `cd clients/web && npx vitest run <datei>`,
  Domain-Fehler mit `instanceof`. Schwerpunkte: Gateways (Mapping laut OpenAPI, HTTP-Fehler),
  Formatter in `presentation/*/format/` (de-CH, Dezimal-Strings).
- BingX-Fixtures nicht roh lesen: `ask-local -f fixtures/bingx/<datei>.json "Liste alle Felder mit Pfad, Typ und Beispielwert. Nur die Liste."`

## programmer
- Go: idiomatisch, nur Standardbibliothek, solange nichts anderes vereinbart ist. Fehler als eigene Typen,
  mit `%w` umhüllen. Kein `any`/`interface{}` ohne Not.
- Web: kein `any`, Imports über `@/`.
- Einzelne Tests: `cd backend && go test ./internal/<paket>/...` bzw. `cd clients/web && npx vitest run <pfad>`.
- Abschluss Go: `cd backend && test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...`
- Abschluss Web (wenn geändert), **1:1 wie die CI** (`ci.yml`, Job Web-Client):
  `cd clients/web && npm run generate:api && test -z "$(git status --porcelain -- src/infrastructure/api/)" && npm run lint && npm run typecheck && npm run format:check && npm test && npm run build`
  Schlägt `format:check` an: `npm run format`, dann alles erneut. Nach jeder Änderung **nach** dem Review (auch ein
  einzelner Test) die ganze Liste erneut ausführen.
- Abschluss Vertragstests (wenn `contracts/` geändert): `cd contracts && npm run lint && npm run typecheck && npm run format:check`
- Ändert sich `.github/workflows/ci.yml`, diese Abschlussbefehle im selben PR angleichen.

## reviewer
- BingX: nur GET-Requests; Fehlerbehandlung (429-Backoff, `code != 0`, leere Antworten).
- Import überschreibt keine Nutzer-Notizen; Deduplizierung ist abgesichert.
- Go: `context` durchgereicht, keine Goroutine-Lecks, `go test -race` grün, Verhalten laut Vertragstest `contracts/`.
- Abgeschwächte Tests finden: `git diff main...HEAD -- '*_test.go' '*.test.ts'`.

## designer
- UI-Code nur in `clients/`, Design-Doku `docs/design.md` (gezielt mit `rg -n "<Thema>" docs/design.md`),
  Tokens in `clients/web/src/styles/design-tokens.css` (hell und dunkel).
- Zahlen: Formatter in `presentation/*/format/` (de-CH, exakt mit Dezimal-Strings, nie über `Number`).
- Gewinn/Verlust: nie nur Farbe, immer zusätzlich Vorzeichen, Text oder Symbol. Trading-UI darf dicht sein, mit klarer Hierarchie.
- Starten: Backend `cd backend && go run ./cmd/trading-intelligence`, Web `cd clients/web && npm run dev`.
- Screenshots: Breiten 1440 px und 390 px, Zustände mit Daten / leer / Fehler / veraltet,
  ablegen unter `clients/web/screenshots/` (in `.gitignore`).
- Playwright: `npm i -D @playwright/test` in `clients/web` und `npx playwright install chromium`, vorher fragen.

## lead-designer
- Es gilt `## designer`.
- Entwürfe: `docs/design/<TASK-ID>-<kurzname>/` mit `flow.excalidraw`, `flow.png`, `prototype.html`.
- Tokens im Prototyp: `<link rel="stylesheet" href="../../../clients/web/src/styles/design-tokens.css">`
  (Theme über `data-theme="light|dark"` am `<html>`), Grundstile aus `clients/web/src/styles/global.css`.
- Fake-Daten realistisch: echte Symbole (BTC-USDT, ONDO-USDT, NCCOGOLD2USD → „Gold“), Long/Short, Hebel,
  negative PnL, lange Namen, fehlender SL („kein SL“), BE-Badge.
- Zustände der App: mit Daten, leer, lädt, Fehler (BingX nicht erreichbar), veraltet (letzter Import > 10 min).
- Fachlicher Kontext für Entscheidungen: Zahlen, Gewinn/Verlust, Margin, Liquidation, R-Linien (1R/2R/3R), Break-even.
- Bestehende Ansichten als Referenz: `docs/images/app.png`, `docs/images/app-chart.png`, `docs/design.md`.
