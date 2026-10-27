---
id: decision-1
title: MCP-Protokoll mit Standardbibliothek
date: '2026-10-27T13:10:00Z'
status: accepted
---
## Context

TASK-19 baut einen lesenden MCP-Server unter `clients/mcp/` (stdio), der die HTTP-API des Backends
abfragt. Das Protokoll kann selbst umgesetzt oder über das offizielle Go-SDK genutzt werden.
Der Server braucht nur `initialize`, `ping`, `tools/list` und `tools/call`.

## Decision

Das Protokoll wird mit der Standardbibliothek umgesetzt (JSON-RPC über stdio, eine Nachricht pro
Zeile), ohne SDK. Der Nutzer hat das am 2026-10-27 so freigegeben. Der Server unterstützt die
Protokollversion `2025-06-18`.

## Consequences

- Keine neue Abhängigkeit, kein `go.sum`; passt zu "nur Standardbibliothek" im Rest des Repos.
- Rund 300 Zeilen Protokollcode gehören dem Projekt und müssen bei neuen Protokollversionen
  oder neuen Features (z. B. Schreib-Tools, HTTP-Transport) selbst nachgezogen werden.
- Ein Wechsel auf das offizielle SDK bleibt möglich, weil Tools hinter einer eigenen Schnittstelle
  (`internal/tools`) liegen und nicht vom Protokollcode abhängen.
