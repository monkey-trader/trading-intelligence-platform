# clients/web – Web-UI

Web-Client der Trading Intelligence Platform. Er basiert auf Vite, React 19, TypeScript und CSS Modules.
Er zeigt die aktuell offenen Positionen und aktualisiert sie alle 15 s.
Der Client spricht **nur über die HTTP-API** mit dem Backend. Fachlogik und Berechnungen kommen aus dem Backend.

Alle Befehle unten laufen im Terminal in diesem Ordner:

```bash
cd clients/web
```

## Voraussetzungen (macOS)

Node.js 24 oder neuer (siehe `../../.nvmrc`):

```bash
brew install node@24
node --version
```

Abhängigkeiten installieren (einmalig und nach jeder Änderung an `package-lock.json`). Erlaubt ist nur npm:

```bash
npm ci
```

## Starten (Entwicklung)

Das UI braucht ein laufendes Backend auf http://127.0.0.1:3000. Es läuft also in zwei Terminals:

```bash
# Terminal 1: Backend (siehe ../../backend/README.md)
cd ../../backend && go run ./cmd/tip-server

# Terminal 2: Web-UI
npm run dev
```

Danach http://127.0.0.1:5173 öffnen.

- Vite bindet nur auf IPv4-Loopback. `localhost` kann auf `::1` zeigen und dann nicht erreichbar sein.
- Anfragen an `/api/*` leitet Vite an das Backend weiter (Proxy in `vite.config.ts`, ohne das Präfix `/api`).
- Beenden mit **Ctrl+C**.

## Bauen und Build ansehen

```bash
npm run build                    # Typecheck und Build nach dist/
npm run preview                  # Build lokal ausliefern, http://127.0.0.1:4173
```

Hinweis: Der Proxy `/api` gilt nur für `npm run dev`. Ein ausgelieferter Build braucht einen Server,
der `/api` an das Backend weiterleitet. Bis zur Desktop-App (Migration Phase 6) ist `npm run dev` der vorgesehene Weg.

## API-Typen aus der Spezifikation erzeugen

Die TypeScript-Typen der API (`src/infrastructure/api/generated.ts`) werden aus
[`docs/openapi.yaml`](../../docs/openapi.yaml) erzeugt. Nach jeder Änderung an der Spec:

```bash
npm run generate:api
```

Die CI prüft, dass die eingecheckte Datei zur Spec passt.

## Tests und Prüfungen

```bash
npm test                         # alle Tests (Vitest, happy-dom, Testing Library)
npm run test:watch               # Watch-Modus
npx vitest run src/presentation/position/components/OpenPositionsPage.test.tsx   # eine Datei
```

Alles, was die CI prüft:

```bash
npm run generate:api && git status --porcelain -- src/infrastructure/api/   # muss leer sein
npm run lint && npm run typecheck && npm run format:check && npm test && npm run build
```

Formatierung korrigieren: `npm run format`

## Port belegt?

```bash
lsof -nP -iTCP:5173 -sTCP:LISTEN   # wer hört auf Port 5173?
kill <PID>
```

## Aufbau

```
src/main.tsx                   Composition Root (verdrahtet Gateway und Seite)
src/domain/position/           Typen und Port (OpenPositionsGateway), ohne React und fetch
src/infrastructure/api/        HTTP-Gateway und generierte API-Typen
src/presentation/position/     Komponenten, View-Model-Hook, Formatierung, errorMapper
src/styles/                    Design-Tokens (hell/dunkel) und globale Styles
```

- `presentation` importiert nichts aus `infrastructure`; ESLint prüft das.
- Styles nur über CSS Modules und Design-Tokens, keine Inline-Styles.
