---
id: TASK-21
title: Einstellungsseite mit Abschnitt MCP-Server
status: To Do
assignee: []
created_date: '2026-10-27T13:10:00Z'
labels:
  - web
  - api
  - mcp
dependencies:
  - TASK-19
priority: low
type: feature
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Abgespalten aus TASK-19. Die App bekommt einen Menüpunkt Einstellungen (Web-UI, in der Desktop-App sichtbar). Abschnitt MCP-Server zeigt, wie der Server erreicht wird, ob er läuft, mit kopierbarem .mcp.json-Ausschnitt. Transport zuerst stdio (Startbefehl); HTTP-Transport mit Adresse nur, wenn ein Client sie braucht. Neue Ansicht: zuerst Skill ui-prototyping, Freigabe durch den Nutzer.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Menüpunkt Einstellungen mit Abschnitt MCP-Server: Transport, Startbefehl, verwendete Backend-URL, Status (läuft / nicht erreichbar) und kopierbarer .mcp.json-Ausschnitt; Daten vom Backend per API (docs/openapi.yaml zuerst, z. B. GET /settings/mcp), der Client zeigt sie nur an
- [ ] #2 Per Tastatur bedienbar, nur Design-Tokens (Kontrast in Hell und Dunkel), bei nicht erreichbarem Server klarer Hinweis statt leerem Bereich, keine Secrets
- [ ] #3 Prototyp unter docs/design/ vom Nutzer freigegeben, bevor Code entsteht
<!-- AC:END -->
