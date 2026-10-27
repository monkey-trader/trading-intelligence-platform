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
- [ ] #4 Ist das ursprüngliche Risiko bekannt (Tabelle `position_risk`), liefert die API das R-Ergebnis `rResult` = netProfit ÷ (|Entry − SL| × positionAmount) (Einheit R, zum Beispiel +3 oder −1), berechnet im Backend; sonst `null`, nie geschätzt. Netto (netProfit) ist die bisherige Annahme, die Entscheidung netto oder brutto ist im Vault noch offen (Trading Strategie Regeln, Begriffe)
- [ ] #5 Die Antwort enthält `breakEvenPrice` = **BE-Level (2R)**, den Preis bei 2× R, an dem der SL auf Break-even gezogen wird (wie `TWO_R` bei den offenen Positionen), und `tp1Price` = **TP1-Level (3R)** (wie `THREE_R`), beide nach rlevels.go aus Entry und ursprünglichem SL; sonst `null`. `breakEvenPrice` ist **nicht** der Entry-Preis
- [ ] #6 Fehler laut bestehendem Problem-Format (`problem.go`); leere Liste, wenn nichts passt; der Endpunkt schreibt nichts
- [ ] #7 Tests zuerst (test-author), `go test -race ./...`, golangci-lint und gofmt sauber; Contract-Test gegen das laufende Backend grün
<!-- AC:END -->

Fachliche Quellen: Vault ai-os `03 Areas/Trading/Trading Strategie Regeln.md` (Begriffe: R, BE-Level, BE-Stop, TP1-Level, R-Ergebnis); `docs/openapi.yaml` (Stufen `TWO_R` und `THREE_R`, `initialStopLoss`).

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. architect: Schnitt prüfen (Repository-Port für Abfrage geschlossener Positionen, Use Case, Route; Abhängigkeiten nach innen). Klären: Wo liegt das ursprüngliche Risiko für bereits geschlossene Positionen (`position_risk` wird beim ersten Sehen angelegt).
2. docs/openapi.yaml zuerst; Contract-Test ergänzen.
3. test-author, programmer: Repository-Methode mit Zeitfilter über Index `positions_symbol_close_time`, Use Case, DTO und Route.
4. reviewer, CI-Liste, unify.

Vorgaben aus dem ersten Durchlauf (2026-10-27, bleiben gültig, Nutzer bestätigt beim Start): Port `ListClosedPositions`; Risiko nur lesend über `PositionRiskRepository.Get` (kein Resolve, wegen AC #6); `from` und `to` als Epoch-ms und inklusiv; `limit` Default 100, Maximum 500, sonst 400, ebenso `from > to`; `leverage` bleibt eine Zahl wie bei `/positions/open`. Ein früher Stand mit `breakEvenPrice = Entry` ist verworfen (liegt im Stash, nur als Vorlage).
<!-- SECTION:PLAN:END -->
