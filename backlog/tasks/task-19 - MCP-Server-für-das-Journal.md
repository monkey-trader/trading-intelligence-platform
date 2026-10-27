---
id: TASK-19
title: MCP-Server für das Journal (lesend, lokal)
status: Done
assignee: []
created_date: '2026-10-27T00:00:00Z'
updated_date: '2026-10-27T13:16:00Z'
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

Sichtbarkeit in der App: Die Anwendung bekommt einen Menüpunkt **Einstellungen** (Web-UI, in der Desktop-App sichtbar, weil sie dasselbe UI zeigt). Dort steht im Abschnitt **MCP-Server**, wie der Server erreicht wird (Adresse bzw. Startbefehl) und ob er läuft, mit einem kopierbaren Ausschnitt für `.mcp.json`. Ohne dass der Nutzer in Dateien suchen muss.

Offene Entscheidung zum Transport: Über stdio gibt es keine Adresse, nur einen Startbefehl. Soll die Einstellungsseite eine Adresse zeigen, braucht der Server zusätzlich einen lokalen HTTP-Transport (z. B. `http://127.0.0.1:<Port>/mcp`, nur Loopback). Empfehlung: erst stdio mit Startbefehl und Konfig-Ausschnitt, den HTTP-Transport als eigenen Schritt, sobald ein Client (z. B. Mission Control) eine Adresse braucht.

Offene Entscheidung: Protokoll selbst umsetzen (nur Standardbibliothek, bei einem kleinen lesenden Server machbar) oder das offizielle Go-SDK verwenden (Abweichung von 'nur Standardbibliothek', als Entscheidung unter backlog/decisions festhalten). Empfehlung: Standardbibliothek, Entscheidung dokumentieren.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `clients/mcp/` ist ein eigenes Go-Modul, spricht nur über die HTTP-API (Basis-URL per Umgebungsvariable, Standard `http://127.0.0.1:3000`) und kennt keine Datenbank
- [x] #2 Tools (nur lesend): `get_open_positions`, `get_closed_positions` (Filter Zeitraum und Symbol), `get_account_summary`, `get_position_note`; jedes mit Beschreibung auf Deutsch und JSON-Schema der Parameter
- [x] #3 Antworten enthalten die Zahlen unverändert aus der API (Dezimal-String, nie umgerechnet); R, BE und TP1 stammen aus dem Backend, nicht aus dem MCP-Server
- [x] #4 Transport stdio; ein nicht erreichbares Backend ergibt eine klare Fehlermeldung, kein Absturz; keine Schlüssel oder Secrets im Server oder in Antworten
- [x] #5 Tests zuerst gegen einen Test-Server (Fixtures aus `fixtures/`); `go test ./...` im Modul, golangci-lint und gofmt sauber; Befehle in der CI-Liste ergänzt
- [x] #6 Anleitung in README oder docs: wie der Server in Claude Code und in Obsidian (Claudian) per `.mcp.json` eingebunden wird, mit einer Beispielfrage
- [x] #7 Entscheidung 'Protokoll selbst oder SDK' steht als Eintrag unter backlog/decisions
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Freigabe 2026-10-27: Einstellungsseite (alt AC #8/#9) abgespalten, Protokoll mit Standardbibliothek (Entscheidung unter backlog/decisions). 1. architect: Schnitt clients/mcp/ (Tools, API-Client, stdio-JSON-RPC), Entscheidung dokumentieren. 2. test-author, programmer: zuerst get_open_positions, dann get_closed_positions, get_account_summary, get_position_note. 3. Doku (.mcp.json, Claude Code, Obsidian), CI-Liste. 4. reviewer, unify.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
architect: Anforderungs-Check ohne Widerspruch. Schnitt clients/mcp (Modul, nur stdlib): main.go, internal/{apiclient,tools,mcp}; Protokoll 2025-06-18 (initialize, ping, tools/list, tools/call), Tool-Fehler als isError mit deutschem Text; Antworten als json.RawMessage unverändert. Neu fixtures/api/ (eigene API-Antworten), CI-Job mcp, agent-rules/CLAUDE.md ergänzen.

test-author: Tests + fixtures/api rot (26 FAIL, Kompilier-Stubs mit TODO(programmer) in apiclient/tools/mcp). Schnittstellen: tools.Getter, Registry.List/Lookup, ErrInvalidArguments, NewServer(*tools.Registry,name,version).

reviewer: freigegeben, keine kritischen Befunde. Behoben: SIGTERM beendet den Server, bekannte Grenzen im README. Offen (Folgepunkte): Zeile >8 MiB graceful melden statt beenden, Tests für Batch/id-Typen/jsonrpc!=2.0, Abbruch per notifications/cancelled und parallele Anfragen, Größenlimit für Backend-Antworten.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
MCP-Server unter clients/mcp (eigenes Go-Modul, nur Standardbibliothek, stdio, Protokoll 2025-06-18) mit den lesenden Tools get_open_positions, get_closed_positions, get_account_summary, get_position_note. Spricht nur über die HTTP-API (TIP_API_URL, Standard 127.0.0.1:3000), reicht Antworten unverändert (RawMessage) durch, Fehler als isError mit deutschem Klartext ohne URL/Header. Neu: fixtures/api, CI-Job mcp, decision-1, README mit .mcp.json/claude mcp add. Einstellungsseite abgespalten nach TASK-21. Geprüft: gofmt, vet, golangci-lint (0 issues), go test -race, build jeweils Exit 0 nach der letzten Änderung; Rauchtest mit echtem Backend (initialize, tools/list, tools/call get_account_summary). Nicht geprüft: Einbindung in Claude Code/Obsidian von Hand (Anleitung steht im README). Risiken: seriell, 8-MiB-Zeilenlimit, siehe Notizen.
<!-- SECTION:FINAL_SUMMARY:END -->
