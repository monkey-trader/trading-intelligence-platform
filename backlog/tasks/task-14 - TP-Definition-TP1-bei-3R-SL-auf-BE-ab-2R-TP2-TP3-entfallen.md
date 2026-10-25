---
id: TASK-14
title: 'TP-Definition: TP1 bei 3R, SL auf BE ab 2R, TP2/TP3 entfallen'
status: Done
assignee: []
created_date: '2026-10-24T20:39:00Z'
updated_date: '2026-10-25T14:06:00Z'
labels:
  - api
  - web
dependencies: []
references:
  - docs/design/TASK-7-positionstabelle-sortierung/prototype.html
priority: medium
type: enhancement
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Neue Fachdefinition der Take-Profit-Stufen (Entscheid des Nutzers 2026-10-24, aus dem UI-Entwurf TASK-7): Bei 2× SL-Abstand (2R) wird der SL auf Break-even gezogen (BE). Bei 3× SL-Abstand (3R) liegt TP1. TP2 und TP3 entfallen. Heute definieren docs/openapi.yaml, docs/features/chart-journal.md und die Tabellenspalte «TPs (je 33 %)» TP1 = 1R, TP2 = 2R (BE), TP3 = 3R mit je 33 % der Position. Offen vor Start: Was passiert mit der 33-%-Aufteilung, und was bedeutet die 1R-Stufe künftig (Chart-Linie oder weg)? Gemäß API-first erst die Spec anpassen, dann Backend, dann Web. Die Spalten BE (2R) und TP1 (3R) im Prototyp von TASK-7 setzen diese Definition voraus.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/openapi.yaml beschreibt die Stufen neu (BE ab 2R, TP1 bei 3R, kein TP2/TP3) und der Vertragstest läuft grün
- [x] #2 Backend liefert die neuen Stufen; Berechnung in Domain mit Tests inkl. Short-Positionen und fehlendem initialStopLoss
- [x] #3 docs/features/chart-journal.md ist an die neue Definition angepasst
- [x] #4 Chart-Journal zeigt Entry, SL, BE und TP1 ohne TP2/TP3
- [x] #5 Bestehende Tabellenspalte «TPs (je 33 %)» und ihr (i)-Hinweis sind auf die neue Definition umgestellt oder entfernt
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Entscheid Nutzer 2026-10-25: 1R-Stufe entfällt ganz (keine Chart-Linie). Strategie: SL ca. max. 1 % vom Konto; bei 2R SL auf BE; bei 3R TP1, üblich 33 % der Position, selten die ganze Position. Die 33 % sind ein Richtwert, keine feste Aufteilung (Restmenge/Stufen danach nicht modelliert).

Entscheide 2026-10-25: Kind-Namen TWO_R/THREE_R bleiben; Major-Release (feat!) ok; Merge-Logik 'TP = 3R' (buildPriceLines.ts) wandert ins Backend. Reihenfolge: Spec+Docs, test-author, programmer, Web, reviewer.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Fachdefinition umgesetzt: nur noch 2R (BE, SL auf Break-even) und 3R (TP1); 1R, TP2, TP3 entfallen. Spec v0.6.0 (breaking: ONE_R raus, rMultiple 2|3, levels maxItems 2, neues RLevel.coincidesWithTakeProfit). Backend: RLevels liefert 2R/3R, LevelAt statt Index (twoRStatus), TakeProfitCoincides mit big.Rat (<= 0,1 % des Entry) in der Domain, DTO ohne Logik. Web: Typen/Gateways/generated.ts, buildPriceLines (BE, TP1, 'TP = 3R' vom Backend-Flag, nur die nächste TP-Order entfällt), BreakEvenTargetInfo, Tabellenspalte 'BE / TP1' mit Dialog 'BE und TP1'. Geprüft: go test -race, golangci-lint, go vet grün; Web 457/457, tsc und eslint sauber; Vertragstest gegen laufende API 137/137 (Backend ohne Keys). Review: keine kritischen Befunde, 2 mittlere behoben (TP-Linie bei mehreren Orders, Mock-API). Abweichungen: Tabellenspalte minimal angepasst (statt entfernt), Spaltensplit/Sortierung bleiben TASK-7. Risiken: kein visueller UI-Check im Browser (nur Komponententests); coincidesWithTakeProfit und levels gegen echte BingX-Daten nicht live gesehen (Vertragstest lief ohne Keys). Folgearbeit: docs/design.md Tabellenabschnitte und ungenutzte CSS-Klasse levelFirst in TASK-7; takeProfitTriggers in getpositionchart.go vor die Schleife ziehen (Stil). Breaking, Commit mit feat! für Major-Release.
<!-- SECTION:FINAL_SUMMARY:END -->
