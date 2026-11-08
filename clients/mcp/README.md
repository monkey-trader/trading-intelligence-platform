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

## HTTP-Modus

Derselbe Server ist zusätzlich per MCP Streamable HTTP (Protokoll 2025-06-18) erreichbar, z.B. für
Agenten in Docker-Containern. stdio bleibt der Standard; ohne `TIP_MCP_HTTP_ADDR` öffnet der Server
keinen Port. Mit gesetzter Adresse läuft **nur** der HTTP-Modus (kein stdio).

| Variable | Bedeutung |
|---|---|
| `TIP_MCP_HTTP_ADDR` | Adresse und Port zum Lauschen, frei wählbar, z.B. `172.30.0.1:8787` (Docker-Gateway) oder `127.0.0.1:8787`. Nicht `0.0.0.0` verwenden. |
| `TIP_MCP_TOKEN_FILE` | Pfad zu einer Datei mit dem Bearer-Token (mindestens 16 Zeichen, Zeilenumbruch am Ende wird entfernt). Pflicht im HTTP-Modus; ohne gültige Datei startet der Server nicht. |
| `TIP_API_URL` | wie bisher, Basis-URL des Backends |

Token erzeugen und Start (der Token steht nur in der Datei, nie auf der Kommandozeile oder im Log):

```bash
umask 077 && openssl rand -hex 32 > /etc/trading-intelligence/mcp-token
TIP_MCP_HTTP_ADDR=172.30.0.1:8787 TIP_MCP_TOKEN_FILE=/etc/trading-intelligence/mcp-token \
  TIP_API_URL=http://127.0.0.1:3000 ./trading-intelligence-mcp
```

Endpunkt: `POST /mcp`, jede Antwort ist ein einzelnes `application/json`. Der Server ist zustandslos
(keine Sessions, kein SSE): `GET` und `DELETE` antworten mit 405, Notifications mit 202.
Ohne gültigen `Authorization: Bearer …` antwortet jede Anfrage mit 401 und leerem Body (der Token wird
in konstanter Zeit verglichen). Anfragen mit `Origin`-Header (Browser) lehnt der Server mit 403 ab,
Die Protokollversion wird bei `initialize` im Body ausgehandelt (unterstützt: `2025-06-18`,
`2025-03-26`; bei anderen Anfragen antwortet der Server mit `2025-06-18`). Der Header `MCP-Protocol-Version`
wird erst bei den folgenden Anfragen geprüft und muss dort eine unterstützte Version sein (sonst 400);
bei `initialize` selbst wird er ignoriert, weil Clients wie Hermes dort schon ihre eigene, neuere Version
senden.

Beispielaufruf (Platzhalter statt Token):

```bash
curl -s http://172.30.0.1:8787/mcp \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Accept: application/json, text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_open_positions","arguments":{}}}'
```

`$MCP_TOKEN` aus der Token-Datei in die Shell laden, z.B. `MCP_TOKEN=$(cat /etc/trading-intelligence/mcp-token)`.

Grenzen: Der Server spricht Klartext-HTTP ohne TLS. Er gehört deshalb nur an eine interne Adresse
(Docker-Gateway, Loopback) oder hinter einen TLS-Proxy. Der Token ist ein gemeinsames Geheimnis ohne
Ablauf; Rotation heißt Datei tauschen und Dienst neu starten.

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
