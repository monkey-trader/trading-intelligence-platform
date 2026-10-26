---
id: TASK-18
title: 'API: Journal-Notizen lesen'
status: To Do
assignee: []
created_date: '2026-10-27T00:00:00Z'
labels:
  - backend
  - api
dependencies: []
priority: medium
type: feature
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Teil von DRAFT-3. Die Tabelle `notes` (setup, rating, mistakes, notes) existiert, hat laut docs/database/schema.md aber noch keine API. Diese Task macht die Einträge lesbar, damit Clients Trade und Notiz zusammen sehen. Schreiben (Anlegen, Ändern) ist bewusst nicht Teil dieser Task und kommt mit dem Notizen-Feature (siehe DRAFT-1 und DRAFT-2).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `GET /positions/{positionId}/note` ist in docs/openapi.yaml beschrieben; 200 mit Notiz, 404 wenn keine existiert, Format der Fehler wie bisher (`problem.go`)
- [ ] #2 Felder: positionId, setup, rating, mistakes, notes, updatedAt (Zeitpunkt)
- [ ] #3 Optional `GET /notes` mit Filter `from`, `to` und `limit` für mehrere Einträge auf einmal (im Plan entscheiden); Contract-Test gesichert
- [ ] #4 Der Endpunkt schreibt nichts; die Notiz lässt sich nicht ohne zugehörige Position lesen (Fremdschlüssel bleibt gewahrt)
- [ ] #5 Tests zuerst, `go test -race ./...`, golangci-lint und gofmt sauber
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. architect: Repository-Port für Notizen, Use Case, Route; prüfen, ob `GET /positions/closed` die Notiz direkt einbetten soll (Antwortgröße gegen Zahl der Aufrufe abwägen).
2. docs/openapi.yaml zuerst; Contract-Test.
3. test-author, programmer, reviewer, unify.
<!-- SECTION:PLAN:END -->
