---
id: TASK-17
title: 'API: geschlossene Positionen lesen'
status: To Do
assignee: []
created_date: '2026-10-27T00:00:00Z'
labels:
  - backend
  - api
dependencies: []
priority: high
type: feature
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Teil von DRAFT-3 (MCP-Server fürs Journal). Die geschlossenen Positionen liegen in der Tabelle `positions` (siehe docs/database/schema.md), sind aber über die API nicht erreichbar. Ohne sie kann kein Client (Web-UI, MCP-Server) Trades auswerten oder mit den Regeln vergleichen. API-first: erst docs/openapi.yaml, dann implementieren; die Fachlogik (R-Ergebnis) gehört ins Backend.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `GET /positions/closed` ist in docs/openapi.yaml beschrieben (Filter `from`, `to` als Zeitpunkt, `symbol`, `limit`; Sortierung nach `closeTime`, neueste zuerst) und per Contract-Test gesichert
- [ ] #2 Antwort je Position: positionId, symbol, side, openTime, closeTime, avgOpenPrice, avgClosePrice, positionAmount, leverage, realisedProfit, netProfit, commission, totalFunding; Beträge als Dezimal-String (nie Float)
- [ ] #3 Nur vollständig geschlossene Positionen (`fully_closed = 1`, `close_time > 0`); die Platzhalter-Zeilen offener Positionen erscheinen nicht
- [ ] #4 Ist das ursprüngliche Risiko bekannt (Tabelle `position_risk`), liefert die API es mit und das R-Ergebnis (netProfit geteilt durch Risiko), berechnet im Backend; sonst `null`, nie geschätzt
- [ ] #5 Die Antwort enthält BE (2R) und TP1 (3R) als Preislevel nach der bestehenden Definition (domain/position/rlevels.go), falls das Risiko bekannt ist
- [ ] #6 Fehler laut bestehendem Problem-Format (`problem.go`); leere Liste, wenn nichts passt; der Endpunkt schreibt nichts
- [ ] #7 Tests zuerst (test-author), `go test -race ./...`, golangci-lint und gofmt sauber; Contract-Test gegen das laufende Backend grün
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. architect: Schnitt prüfen (Repository-Port für Abfrage geschlossener Positionen, Use Case, Route; Abhängigkeiten nach innen). Klären: Wo liegt das ursprüngliche Risiko für bereits geschlossene Positionen (`position_risk` wird beim ersten Sehen angelegt).
2. docs/openapi.yaml zuerst; Contract-Test ergänzen.
3. test-author, programmer: Repository-Methode mit Zeitfilter über Index `positions_symbol_close_time`, Use Case, DTO und Route.
4. reviewer, CI-Liste, unify.
<!-- SECTION:PLAN:END -->
