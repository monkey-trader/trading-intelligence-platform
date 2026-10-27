# MCP-Server für das Journal

Stellt das Trading-Journal KI-Clients (Claude Code, Obsidian mit Claudian) über das Model Context Protocol
bereit. Der Server spricht stdio (Protokoll 2025-06-18) und ruft nur die HTTP-API des Backends auf.
Er ist **nur lesend** und kennt keine Secrets (kein BingX-Key).

Tools: `get_open_positions`, `get_closed_positions` (Filter `from`, `to`, `symbol`, `limit`),
`get_account_summary`, `get_position_note` (`positionId` als String).

## Start

Das Backend muss laufen (`cd backend && go run ./cmd/trading-intelligence`).

```bash
cd clients/mcp && go build -o trading-intelligence-mcp .
TIP_API_URL=http://127.0.0.1:3000 ./trading-intelligence-mcp
```

`TIP_API_URL` ist die Basis-URL der API, Standard `http://127.0.0.1:3000`.
stdout gehört dem Protokoll, Logs gehen nach stderr. Ende mit EOF oder Ctrl+C.

## Einbindung

Claude Code:

```bash
claude mcp add trading-intelligence -e TIP_API_URL=http://127.0.0.1:3000 -- /absoluter/pfad/trading-intelligence-mcp
```

Oder per `.mcp.json`:

```json
{
  "mcpServers": {
    "trading-intelligence": {
      "command": "/absoluter/pfad/trading-intelligence-mcp",
      "env": { "TIP_API_URL": "http://127.0.0.1:3000" }
    }
  }
}
```

Obsidian (Claudian): denselben Eintrag (`command`, `env`) in der MCP-Konfiguration des Plugins hinterlegen.

## Beispielfrage

„Wie lief mein Oktober? Zeig mir die geschlossenen Positionen mit dem schlechtesten R-Ergebnis und ihre Notizen.“

## Bekannte Grenzen

- Anfragen werden nacheinander bearbeitet; ein hängendes Backend (Timeout 10 s) hält weitere Anfragen auf.
  `notifications/cancelled` wird ignoriert.
- Eine einzelne Nachricht über 8 MiB beendet den Server.

## Entwicklung

`cd clients/mcp && test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...`
