---
id: TASK-19
title: 'MCP-Server für das Journal (lesend, lokal)'
status: To Do
assignee: []
created_date: '2026-10-27T00:00:00Z'
labels:
  - mcp
  - client
dependencies: 
  - TASK-17
  - TASK-18
priority: medium
type: feature
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Aus DRAFT-3. Ein MCP-Server als **Client der HTTP-API** unter `clients/mcp/` (eigenes Go-Modul wie `clients/desktop`), damit Claude (Obsidian-Chat, Claude Code, Desktop) Fragen zu Trades im Journal beantworten kann, zum Beispiel „welche Trades der letzten Woche hatten SL nicht vor dem nächsten Fib-Level“ oder „zeige offene Positionen mit Abstand zu BE und TP1“. Er greift nie auf die Datenbank zu (Grundprinzip in CLAUDE.md). Zuerst nur lesend und lokal über stdio; Schreiben ins Journal und Zugriff aus der Ferne folgen später. Die Bewertung gegen Checkliste und Regeln bleibt im Skill `trade-check` im ai-os; der Server liefert nur die Zahlen.

Offene Entscheidung: Protokoll selbst umsetzen (nur Standardbibliothek, bei einem kleinen lesenden Server machbar) oder das offizielle Go-SDK verwenden (Abweichung von 'nur Standardbibliothek', als Entscheidung unter backlog/decisions festhalten). Empfehlung: Standardbibliothek, Entscheidung dokumentieren.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `clients/mcp/` ist ein eigenes Go-Modul, spricht nur über die HTTP-API (Basis-URL per Umgebungsvariable, Standard `http://127.0.0.1:3000`) und kennt keine Datenbank
- [ ] #2 Tools (nur lesend): `get_open_positions`, `get_closed_positions` (Filter Zeitraum und Symbol), `get_account_summary`, `get_position_note`; jedes mit Beschreibung auf Deutsch und JSON-Schema der Parameter
- [ ] #3 Antworten enthalten die Zahlen unverändert aus der API (Dezimal-String, nie umgerechnet); R, BE und TP1 stammen aus dem Backend, nicht aus dem MCP-Server
- [ ] #4 Transport stdio; ein nicht erreichbares Backend ergibt eine klare Fehlermeldung, kein Absturz; keine Schlüssel oder Secrets im Server oder in Antworten
- [ ] #5 Tests zuerst gegen einen Test-Server (Fixtures aus `fixtures/`); `go test ./...` im Modul, golangci-lint und gofmt sauber; Befehle in der CI-Liste ergänzt
- [ ] #6 Anleitung in README oder docs: wie der Server in Claude Code und in Obsidian (Claudian) per `.mcp.json` eingebunden wird, mit einer Beispielfrage
- [ ] #7 Entscheidung 'Protokoll selbst oder SDK' steht als Eintrag unter backlog/decisions
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. architect: Schnitt von `clients/mcp/` (Tools, API-Client, stdio-Schleife), Entscheidung Protokoll selbst oder SDK, Konzept für spätere Schreib-Tools (nur nach Freigabe).
2. test-author, programmer: API-Client mit Fixtures, Tool-Definitionen, stdio-Server; zuerst `get_open_positions`, dann die übrigen.
3. Manuell testen mit Claude Code (`claude mcp add`) und in Obsidian; Beispielfragen in der Doku festhalten.
4. reviewer, CI-Liste, unify.
<!-- SECTION:PLAN:END -->
