# backend – Go-Backend (`trading-intelligence`)

Das Backend der Trading Intelligence Platform (Go, nur Standardbibliothek): BingX-Import, SQLite,
Fachlogik und HTTP-API gemäß [`docs/openapi.yaml`](../docs/openapi.yaml). Das frühere TypeScript-Backend
ist entfernt; den Ablauf der Migration beschreibt [`docs/migration/go-wails.md`](../docs/migration/go-wails.md).

Alle Befehle unten laufen im Terminal in diesem Ordner:
```bash
cd backend
```

## Voraussetzungen (macOS)

```bash
brew install go                  # Go 1.27 oder neuer
go version                       # Kontrolle, z.B. "go version go1.27.2 darwin/arm64"
```

Für die Lint-Prüfung brauchst du golangci-lint **v2.15 oder neuer, mit Go 1.27 gebaut**.
Ein älteres golangci-lint bricht mit „Go language version … lower than the targeted Go version“ ab.
```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
golangci-lint version
```
`go install` legt das Programm nach `~/go/bin`. Fehlt dieser Ordner im `PATH`, trägst du ihn einmalig nach:
```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Externe Go-Abhängigkeiten gibt es nicht, nur die Standardbibliothek. Deshalb ist kein `go mod download` nötig.

## Starten (Entwicklung)

Ohne Build-Schritt, Go kompiliert das Programm im Hintergrund:
```bash
go run ./cmd/trading-intelligence
```
Die API läuft danach auf http://127.0.0.1:3000.

Belegt schon ein anderer Prozess Port 3000, wählst du einen anderen Port:
```bash
PORT=3001 go run ./cmd/trading-intelligence
```

Beenden mit **Ctrl+C**. Der Server schließt laufende Anfragen noch ab (höchstens 10 s).
Ein zweites Ctrl+C beendet ihn sofort.

## Bauen und Binary starten

```bash
go build -o trading-intelligence ./cmd/trading-intelligence    # erzeugt ./trading-intelligence (steht in .gitignore)
./trading-intelligence                               # Start auf 127.0.0.1:3000
PORT=3001 ./trading-intelligence                     # anderer Port
```

Ein optimiertes, kleineres Binary für Apple Silicon und für Intel:
```bash
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o trading-intelligence-arm64 ./cmd/trading-intelligence
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o trading-intelligence-amd64 ./cmd/trading-intelligence
```

Ein selbst gebautes, unsigniertes Binary läuft im Terminal ohne Rückfrage. Nur ein heruntergeladenes Binary
blockiert macOS Gatekeeper. Das lässt sich einmalig freigeben:
```bash
xattr -d com.apple.quarantine ./trading-intelligence
```

## Konfiguration (Umgebungsvariablen)

| Variable | Standard | Bedeutung |
|---|---|---|
| `PORT` | `3000` | Port, ganze Zahl 1–65535 |
| `HOST` | `127.0.0.1` | Bind-Adresse. Erlaubt ist nur Loopback, sonst braucht es `ALLOW_REMOTE=true`. |
| `ALLOW_REMOTE` | – | `true` erlaubt einen Nicht-Loopback-`HOST`. Die API hat noch keine Authentifizierung! |
| `ALLOWED_HOSTS` | – | Bei `ALLOW_REMOTE=true` Pflicht: kommagetrennte Host-Header mit Port, z.B. `meinrechner.local:3000` |
| `BX_KEY`, `BX_SECRET` | – | BingX-Zugang, Lese-Key von BingX. Fehlen sie, kommt beim Start nur ein Hinweis und die Positions- und Chart-Routen liefern 503 `not-configured`. |

`trading-intelligence` liest **keine `.env`-Datei**. Variablen setzt du direkt vor den Befehl
(`PORT=3001 ./trading-intelligence`) oder exportierst sie in der Shell (`BX_KEY`, `BX_SECRET`).

Das Log geht als JSON auf stderr. Zum Mitlesen mit `jq`:
```bash
go run ./cmd/trading-intelligence 2>&1 | jq
```

## Ausprobieren

```bash
curl -i http://127.0.0.1:3000/health                           # 200 {"status":"ok"}
curl -i http://127.0.0.1:3000/positions/open                   # 503 not-configured (bis Phase 2)
curl -i http://127.0.0.1:3000/gibtsnicht                       # 404 not-found
curl -i -H 'Host: evil.example' http://127.0.0.1:3000/health   # 403 forbidden-host
```

## Tests und Prüfungen

```bash
go test ./...                    # alle Tests
go test -race ./...              # mit Data-Race-Erkennung (so läuft die CI)
go test -run TestReadServerConfig ./internal/infrastructure/config   # einzelner Test
go test -cover ./...             # mit Abdeckung
```

Alles, was die CI prüft, in einem Befehl:
```bash
test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./... && go build ./...
```
Formatierung korrigieren: `gofmt -w .`

## Vertragstest

Der Vertragstest in `../contracts/` prüft die laufende API gegen `docs/openapi.yaml`.
Starte zuerst den Server in Terminal 1, dann in Terminal 2:
```bash
# Terminal 1
PORT=3001 go run ./cmd/trading-intelligence

# Terminal 2 (einmalig vorher: cd ../contracts && npm ci)
cd ../contracts && CONTRACT_BASE_URL=http://127.0.0.1:3001 npm test
```

## Port belegt?

```bash
lsof -nP -iTCP:3001 -sTCP:LISTEN   # wer hört auf Port 3001?
kill <PID>
```

## Aufbau

```
cmd/trading-intelligence/main.go          Composition Root: Konfiguration, Server, Graceful Shutdown
internal/domain/                Domain-Kern (ab Phase 2)
internal/application/           Use Cases (ab Phase 2)
internal/infrastructure/config/ Umgebungsvariablen lesen und prüfen
internal/presentation/webui/    eingebettetes Web-UI (dist, im Release vom CI-Build befüllt)
internal/presentation/httpapi/  HTTP: Router, Host-Prüfung, Problem-Antworten, Listener-Wrapper
.golangci.yml                   Lint-Regeln inkl. Schichtgrenzen (depguard)
```
Abhängigkeiten zeigen immer nach innen. Die Verdrahtung passiert nur in `cmd/`.

## Web-UI

`internal/presentation/webui/dist` wird per `go:embed` eingebettet und unter `/` ausgeliefert
(SPA-Fallback auf `index.html`). Im Quellbaum liegt dort nur `.gitkeep`; dann antwortet `/` mit 404
"Web-UI nicht eingebettet". Der Release-Build baut `clients/web` mit `VITE_API_BASE=''` und kopiert `dist` hierher.
