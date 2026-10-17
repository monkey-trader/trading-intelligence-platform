# backend-go – Go-Backend (`tip-server`)

Das Go-Backend der Trading Intelligence Platform entsteht parallel zum TypeScript-Backend
(`../backend/`). Den Ablauf beschreibt [`docs/migration/go-wails.md`](../docs/migration/go-wails.md).

**Stand Phase 1:** `GET /health` und die allgemeinen Regeln aus `docs/openapi.yaml`
(Host-Prüfung, 404, 400, Fehler als `problem+json`) funktionieren. `GET /positions/open` liefert
bis Phase 2 immer 503 `not-configured`. Das Web-UI spricht weiterhin mit dem TS-Backend.

Alle Befehle unten laufen im Terminal in diesem Ordner:
```bash
cd backend-go
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
go run ./cmd/tip-server
```
Die API läuft danach auf http://127.0.0.1:3000.

Parallel zum TS-Backend, das schon Port 3000 belegt, startest du so:
```bash
PORT=3001 go run ./cmd/tip-server
```

Beenden mit **Ctrl+C**. Der Server schließt laufende Anfragen noch ab (höchstens 10 s).
Ein zweites Ctrl+C beendet ihn sofort.

## Bauen und Binary starten

```bash
go build -o tip-server ./cmd/tip-server    # erzeugt ./tip-server (steht in .gitignore)
./tip-server                               # Start auf 127.0.0.1:3000
PORT=3001 ./tip-server                     # anderer Port
```

Ein optimiertes, kleineres Binary für Apple Silicon und für Intel:
```bash
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o tip-server-arm64 ./cmd/tip-server
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o tip-server-amd64 ./cmd/tip-server
```

Ein selbst gebautes, unsigniertes Binary läuft im Terminal ohne Rückfrage. Nur ein heruntergeladenes Binary
blockiert macOS Gatekeeper. Das lässt sich einmalig freigeben:
```bash
xattr -d com.apple.quarantine ./tip-server
```

## Konfiguration (Umgebungsvariablen)

| Variable | Standard | Bedeutung |
|---|---|---|
| `PORT` | `3000` | Port, ganze Zahl 1–65535 |
| `HOST` | `127.0.0.1` | Bind-Adresse. Erlaubt ist nur Loopback, sonst braucht es `ALLOW_REMOTE=true`. |
| `ALLOW_REMOTE` | – | `true` erlaubt einen Nicht-Loopback-`HOST`. Die API hat noch keine Authentifizierung! |
| `ALLOWED_HOSTS` | – | Bei `ALLOW_REMOTE=true` Pflicht: kommagetrennte Host-Header mit Port, z.B. `meinrechner.local:3000` |
| `BX_KEY`, `BX_SECRET` | – | BingX-Zugang, wird erst ab Phase 2 genutzt. Fehlen sie, kommt beim Start nur ein Hinweis. |

Anders als das TS-Backend liest `tip-server` **keine `.env`-Datei**. Variablen setzt du direkt
vor den Befehl (`PORT=3001 ./tip-server`). Wenn nötig, lädst du die bestehende Datei in die Shell:
```bash
set -a && source ../backend/.env && set +a && ./tip-server
```

Das Log geht als JSON auf stderr. Zum Mitlesen mit `jq`:
```bash
go run ./cmd/tip-server 2>&1 | jq
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

Der Vertragstest in `../contract-tests/` prüft die laufende API gegen `docs/openapi.yaml`.
Es ist derselbe Test wie für das TS-Backend. Starte zuerst den Server in Terminal 1, dann in Terminal 2:
```bash
# Terminal 1
PORT=3001 go run ./cmd/tip-server

# Terminal 2 (einmalig vorher: cd ../contract-tests && npm ci)
cd ../contract-tests && CONTRACT_BASE_URL=http://127.0.0.1:3001 npm test
```

## Port belegt?

```bash
lsof -nP -iTCP:3001 -sTCP:LISTEN   # wer hört auf Port 3001?
kill <PID>
```

## Aufbau

```
cmd/tip-server/main.go          Composition Root: Konfiguration, Server, Graceful Shutdown
internal/domain/                Domain-Kern (ab Phase 2)
internal/application/           Use Cases (ab Phase 2)
internal/infrastructure/config/ Umgebungsvariablen lesen und prüfen
internal/presentation/httpapi/  HTTP: Router, Host-Prüfung, Problem-Antworten, Listener-Wrapper
.golangci.yml                   Lint-Regeln inkl. Schichtgrenzen (depguard)
```
Abhängigkeiten zeigen immer nach innen. Die Verdrahtung passiert nur in `cmd/`.
