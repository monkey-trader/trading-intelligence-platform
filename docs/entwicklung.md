# Entwickeln und lokal starten

Technische Anleitung für Entwickler. Die Installation für Anwender steht im [README](../README.md).

## Lokal starten (aus dem Quellcode)

### 1. Einmalig: installieren und Keys hinterlegen
Voraussetzung: Go 1.27 (`brew install go`), Node (siehe `.nvmrc`), für die Prüfungen zusätzlich golangci-lint v2.15 oder neuer,
mit Go 1.27 gebaut: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`.
```bash
(cd clients/web && npm ci) && (cd contracts && npm ci)
export BX_KEY=… BX_SECRET=…   # Key nur mit Lese-Rechten; das Backend liest keine .env-Datei
```

### 2. App starten (zwei Terminals, jeweils im Repo-Root)

**Terminal 1: Backend** (API auf http://127.0.0.1:3000, Import alle 5 Minuten, Details in [`backend/README.md`](../backend/README.md))
```bash
cd backend && go run ./cmd/trading-intelligence
```

**Terminal 2: Web-UI** (Proxy `/api` → Backend)
```bash
cd clients/web && npm run dev
```

Danach http://127.0.0.1:5173 öffnen (Vite bindet nur auf IPv4-Loopback; `localhost` kann auf `::1` zeigen).
Stoppen: `Ctrl+C` im jeweiligen Terminal. Die SQLite-Datei liegt standardmäßig unter
`<UserConfigDir>/tip/journal.db` (macOS: `~/Library/Application Support/tip/journal.db`, `JOURNAL_DB` überschreibt).

### 3. Vertragstest gegen das laufende Backend (Terminal 3)
Prüft die Antworten des Backends gegen `openapi.yaml`. Das Backend aus Terminal 1 muss laufen.
```bash
cd contracts && CONTRACT_BASE_URL=http://127.0.0.1:3000 CONTRACT_EXPECT_CONFIGURED=true npm test
```
- `CONTRACT_EXPECT_CONFIGURED=true` nur, wenn `BX_KEY`/`BX_SECRET` gesetzt sind (erwartet dann 200 für
  `/positions/open`). Ohne Keys weglassen, dann wird 503 `not-configured` erwartet (wie in CI).
- Ohne laufenden Server (nur Validator und Abdeckungs-Check): `cd contracts && npm run test:unit`

### 4. Prüfungen des Backends
```bash
cd backend && test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...
```

### API von Hand ausprobieren
```bash
curl -i http://127.0.0.1:3000/health                           # 200 {"status":"ok"}
curl -s http://127.0.0.1:3000/positions/open                   # offene Positionen (JSON)
curl -i http://127.0.0.1:3000/gibtsnicht                       # 404 not-found
curl -i -H 'Host: evil.example' http://127.0.0.1:3000/health   # 403 forbidden-host
```

### Prüfungen vor einem Commit
```bash
(cd clients/web && npm run lint && npm run typecheck && npm test)
(cd contracts && npm run lint && npm run typecheck && npm run test:unit)
(cd backend && test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...)
```

### Server hängt im Hintergrund?
```bash
lsof -nP -iTCP:3000 -sTCP:LISTEN   # Port 3000 (Backend) bzw. 5173 (UI)
kill <PID>
```

## Weiterführend

- Regeln, Architektur und Ablauf: [`CLAUDE.md`](../CLAUDE.md)
- API-Vertrag: [`openapi.yaml`](openapi.yaml)
- Backend: [`backend/README.md`](../backend/README.md)
- Entwickeln mit AI-Agenten: [`ai-blueprint.md`](ai-blueprint.md)
- Screenshots neu erzeugen: `node clients/web/scripts/mock-api.mjs`, dann `BACKEND_URL=http://127.0.0.1:3299 npx vite --port 5174` (beides in `clients/web`) und `node scripts/screenshot.mjs` (erzeugt beide Bilder in `docs/images/`).
