---
id: TASK-14
title: 'TP-Definition: TP1 bei 3R, SL auf BE ab 2R, TP2/TP3 entfallen'
status: To Do
assignee: []
created_date: '2026-10-24T20:39:00Z'
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
- [ ] #1 docs/openapi.yaml beschreibt die Stufen neu (BE ab 2R, TP1 bei 3R, kein TP2/TP3) und der Vertragstest läuft grün
- [ ] #2 Backend liefert die neuen Stufen; Berechnung in Domain mit Tests inkl. Short-Positionen und fehlendem initialStopLoss
- [ ] #3 docs/features/chart-journal.md ist an die neue Definition angepasst
- [ ] #4 Chart-Journal zeigt Entry, SL, BE und TP1 ohne TP2/TP3
- [ ] #5 Bestehende Tabellenspalte «TPs (je 33 %)» und ihr (i)-Hinweis sind auf die neue Definition umgestellt oder entfernt
<!-- AC:END -->
