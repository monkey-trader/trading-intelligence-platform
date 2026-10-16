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
  Deduplizierung über `tranId`.
- Positionen (`/openApi/swap/v1/trade/positionHistory`): **pro Symbol** abfragen,
  7-Tage-Fenster (größere Fenster liefern still eine leere Liste), Paging über `pageIndex`.
- Fortschritt pro Datenquelle in einer State-Tabelle merken, mit 1 h Überlappung.
- Nutzer-Notizen werden vom Import **niemals** überschrieben.

### Bekanntes Problem im Prototyp
Im Prototyp ist die Tabelle `positions` **leer**, obwohl der Sync alle Symbole als erledigt
markiert und `income` 2'137 Einträge hat. Der Import scheitert still (vermutlich falscher
Antwort-Schlüssel beim Auslesen oder ungeeignete Parameter). Ohne Positionen können Notizen
keinem Trade zugeordnet werden.

Konsequenz für dieses Projekt:
- Erstes Feature ist der **Positions-Import**, und er muss nachweislich Daten liefern.
- Eine leere API-Antwort bei vorhandenem Income für dasselbe Symbol ist verdächtig und wird
  geloggt, nicht stillschweigend als "erledigt" markiert.
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

## Subagents (`.claude/agents/`)
- `architect` – prüft Schichtengrenzen und den API-Vertrag
- `test-author` – schreibt Tests zuerst
- `programmer` – macht Tests grün, ohne sie zu ändern
- `reviewer` – Qualität und Sicherheit am Schluss

Ablauf pro Feature: architect (Schnitt) → test-author → programmer → reviewer.
