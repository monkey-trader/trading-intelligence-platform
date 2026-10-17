# backend – TypeScript-Backend

Produktives Backend der Trading Intelligence Platform: BingX-Import, SQLite und HTTP-API laut
[`docs/openapi.yaml`](../docs/openapi.yaml). Es basiert auf Node.js 24, TypeScript, Fastify und `node:sqlite`.
Bis zum Ende der Go-Migration ([`docs/migration/go-wails.md`](../docs/migration/go-wails.md))
bleibt dieses Backend maßgeblich; danach ersetzt es `../backend-go/`.

Alle Befehle unten laufen im Terminal in diesem Ordner:

```bash
cd backend
```

## Voraussetzungen (macOS)

Node.js 24 oder neuer (siehe `../.nvmrc`), entweder mit Homebrew:

```bash
brew install node@24
node --version                   # Kontrolle, z.B. "v24.x"
```

oder mit nvm. Dieser Befehl liest `../.nvmrc` aus dem Repo-Root:

```bash
cd .. && nvm install && nvm use && cd backend
```

Abhängigkeiten installieren (einmalig und nach jeder Änderung an `package-lock.json`). Erlaubt ist nur npm, kein pnpm oder yarn:

```bash
npm ci
```

## BingX-Zugang einrichten

```bash
cp .env.example .env             # danach BX_KEY und BX_SECRET eintragen
```

- Den API-Key bei BingX **nur mit Lese-Rechten** anlegen.
- `.env` steht in `.gitignore` und wird nie committet.
- Ohne Keys startet der Server trotzdem. `GET /positions/open` liefert dann 503 `not-configured`.

## Starten (Entwicklung)

```bash
npm run dev
```

- Die API läuft auf http://127.0.0.1:3000.
- `tsx watch` startet den Server bei jeder Änderung an einer Datei neu und lädt `.env`, falls vorhanden.
- Beenden mit **Ctrl+C**.

## Bauen und gebaut starten

```bash
npm run build                    # bündelt nach dist/ (tsup): dist/main.js und dist/cli/importPositions.js
npm start                        # startet dist/main.js mit .env
```

Ohne `.env`, mit Variablen direkt im Befehl:

```bash
PORT=3001 node dist/main.js
```

## Positionen von BingX importieren

```bash
npm run import:positions
```

- Der Import schreibt geschlossene Positionen nach `data/journal.sqlite` (überschreibbar mit `JOURNAL_DB`). Nutzer-Notizen fasst er nie an.
- Beim ersten Lauf sucht er einmalig alle USDT-Kontrakte ab. Das kann lange dauern (ca. 1 h).
- Der Lauf ist fortsetzbar: Mit Ctrl+C abbrechen und später erneut starten.
- Fortschritt und Hinweise gehen auf stderr.
- Exit-Codes:
  - `0`: ok
  - `1`: Fehler
  - `2`: verdächtige Symbole, d.h. Income vorhanden, aber keine Positionen gefunden

Datenbank ansehen:

```bash
sqlite3 data/journal.sqlite 'SELECT symbol, COUNT(*) FROM positions GROUP BY symbol;'
```

Das Schema beschreibt [`docs/database/schema.md`](../docs/database/schema.md).

## Konfiguration (Umgebungsvariablen, in `.env`)

| Variable              | Standard                     | Bedeutung                                                                         |
| --------------------- | ---------------------------- | --------------------------------------------------------------------------------- |
| `BX_KEY`, `BX_SECRET` | –                            | BingX-Zugang (nur Lese-Rechte)                                                    |
| `PORT`                | `3000`                       | Port der API                                                                      |
| `HOST`                | `127.0.0.1`                  | Bind-Adresse; nur Loopback, sonst braucht es `ALLOW_REMOTE=true`                  |
| `ALLOW_REMOTE`        | –                            | `true` erlaubt einen Nicht-Loopback-`HOST` (die API hat keine Authentifizierung!) |
| `ALLOWED_HOSTS`       | –                            | Mit `ALLOW_REMOTE=true` Pflicht: Host-Header mit Port, kommagetrennt              |
| `BX_BASE`             | `https://open-api.bingx.com` | BingX-API-Basis-URL                                                               |
| `JOURNAL_DB`          | `data/journal.sqlite`        | Pfad zur SQLite-Datenbank                                                         |
| `IMPORT_FULL_DAYS`    | `365`                        | Tage Historie beim ersten Vollimport                                              |

## Ausprobieren

```bash
curl -i http://127.0.0.1:3000/health                           # 200 {"status":"ok"}
curl -s http://127.0.0.1:3000/positions/open | jq              # offene Positionen live von BingX
curl -i http://127.0.0.1:3000/gibtsnicht                       # 404 not-found
curl -i -H 'Host: evil.example' http://127.0.0.1:3000/health   # 403 forbidden-host
```

## Tests und Prüfungen

```bash
npm test                         # alle Tests (Vitest)
npm run test:watch               # Watch-Modus
npx vitest run src/presentation/http/createServer.test.ts   # eine Datei
npx vitest run -t "not-found"                                # Tests nach Name
```

Alles, was die CI prüft:

```bash
npm run lint && npm run typecheck && npm run format:check && npm test
```

Formatierung korrigieren: `npm run format`

## Vertragstest

Der Vertragstest in `../contract-tests/` prüft die laufende API gegen `docs/openapi.yaml`.
Starte zuerst den Server in Terminal 1, dann in Terminal 2:

```bash
# Terminal 1
npm run dev

# Terminal 2 (einmalig vorher: cd ../contract-tests && npm ci)
cd ../contract-tests && CONTRACT_BASE_URL=http://127.0.0.1:3000 CONTRACT_EXPECT_CONFIGURED=true npm test
```

`CONTRACT_EXPECT_CONFIGURED=true` gehört nur dazu, wenn Keys in `.env` stehen. Ohne Keys wird 503 `not-configured` erwartet.

## Port belegt?

```bash
lsof -nP -iTCP:3000 -sTCP:LISTEN   # wer hört auf Port 3000?
kill <PID>
```

## Aufbau

```
src/main.ts                    Composition Root der HTTP-API
src/cli/importPositions.ts     Composition Root des Import-CLI
src/domain/                    Entities, Value Objects, Errors, Ports, Domain-Services
src/application/               Use Cases (Import, offene Positionen)
src/infrastructure/            BingX-Client, SQLite, Konfiguration, Cache
src/presentation/              HTTP (Fastify-Routen, Host-Prüfung, Problem-Antworten), CLI-Ausgabe
```

- Abhängigkeiten zeigen immer nach innen; ESLint prüft das.
- Importiert wird über den Alias `@/` (→ `src/`), Fixtures über `@fixtures/` (→ `../fixtures/`).
