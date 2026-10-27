---
id: TASK-17
title: 'API: geschlossene Positionen lesen'
status: Done
assignee: []
created_date: '2026-10-27T00:00:00Z'
updated_date: '2026-10-27T06:50:00Z'
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
- [x] #1 `GET /positions/closed` ist in docs/openapi.yaml beschrieben (Filter `from`, `to` als Zeitpunkt, `symbol`, `limit`; Sortierung nach `closeTime`, neueste zuerst) und per Contract-Test gesichert
- [x] #2 Antwort je Position: positionId, symbol, side, openTime, closeTime, avgOpenPrice, avgClosePrice, positionAmount, leverage, realisedProfit, netProfit, commission, totalFunding; Beträge als Dezimal-String (nie Float)
- [x] #3 Nur vollständig geschlossene Positionen (`fully_closed = 1`, `close_time > 0`); die Platzhalter-Zeilen offener Positionen erscheinen nicht
- [x] #4 Ist das ursprüngliche Risiko bekannt (Tabelle `position_risk`), liefert die API das R-Ergebnis `rResult` = netProfit ÷ (|Entry − SL| × positionAmount) (Einheit R, zum Beispiel +3 oder −1), berechnet im Backend; sonst `null`, nie geschätzt. Netto (netProfit) ist die bisherige Annahme, die Entscheidung netto oder brutto ist im Vault noch offen (Trading Strategie Regeln, Begriffe)
- [x] #5 Die Antwort enthält `breakEvenPrice` = **BE-Level (2R)**, den Preis bei 2× R, an dem der SL auf Break-even gezogen wird (wie `TWO_R` bei den offenen Positionen), und `tp1Price` = **TP1-Level (3R)** (wie `THREE_R`), beide nach rlevels.go aus Entry und ursprünglichem SL; sonst `null`. `breakEvenPrice` ist **nicht** der Entry-Preis
- [x] #6 Fehler laut bestehendem Problem-Format (`problem.go`); leere Liste, wenn nichts passt; der Endpunkt schreibt nichts
- [x] #7 Tests zuerst (test-author), `go test -race ./...`, golangci-lint und gofmt sauber; Contract-Test gegen das laufende Backend grün
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. architect: Schnitt prüfen (Repository-Port für Abfrage geschlossener Positionen, Use Case, Route; Abhängigkeiten nach innen). Klären: Wo liegt das ursprüngliche Risiko für bereits geschlossene Positionen (`position_risk` wird beim ersten Sehen angelegt).
2. docs/openapi.yaml zuerst; Contract-Test ergänzen.
3. test-author, programmer: Repository-Methode mit Zeitfilter über Index `positions_symbol_close_time`, Use Case, DTO und Route.
4. reviewer, CI-Liste, unify.

Vorgaben aus dem ersten Durchlauf (2026-10-27, bleiben gültig, Nutzer bestätigt beim Start): Port `ListClosedPositions`; Risiko nur lesend über `PositionRiskRepository.Get` (kein Resolve, wegen AC #6); `from` und `to` als Epoch-ms und inklusiv; `limit` Default 100, Maximum 500, sonst 400, ebenso `from > to`; `leverage` bleibt eine Zahl wie bei `/positions/open`. Ein früher Stand mit `breakEvenPrice = Entry` ist verworfen (liegt im Stash, nur als Vorlage).
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
GET /positions/closed umgesetzt (openapi.yaml zuerst, Wrapper {positions:[]}; Filter from/to inklusiv, symbol, limit 1..500, Default 100; 400 bei ungültiger Eingabe, 503 not-configured ohne Datenbank). rResult = netProfit ÷ (|Entry−SL|×positionAmount) mit big.Rat, 4 Nachkommastellen; breakEvenPrice = TWO_R, tp1Price = THREE_R; ohne bekanntes oder verwertbares Risiko null, Risiko nur lesend. Neuer Index positions_close_time (EXPLAIN: ohne ihn Scan plus Sortierschritt). Geprüft: gofmt, go vet, golangci-lint (0 issues), go test -race grün; Contract-Test gegen frisch gebautes Backend mit Kopie der echten DB (Port 3001) grün, die drei 503-Tests gegen Backend ohne Keys grün (17/17). Abweichungen: 503-Fall und Index kamen über Architect/Reviewer dazu. Risiken: netto statt brutto beim R-Ergebnis ist im Vault weiter offen; die installierte Homebrew-Version kennt die Route erst nach neuem Release.
<!-- SECTION:FINAL_SUMMARY:END -->

Fachliche Quellen: Vault ai-os `03 Areas/Trading/Trading Strategie Regeln.md` (Begriffe: R, BE-Level, BE-Stop, TP1-Level, R-Ergebnis); `docs/openapi.yaml` (Stufen `TWO_R` und `THREE_R`, `initialStopLoss`).
