---
id: TASK-1
title: 'Chart Stufe 3: abgeschlossene Trades im Journal'
status: To Do
assignee:
  - monkey
created_date: '2026-10-24T20:01:00Z'
updated_date: '2026-10-24T20:54:00Z'
labels:
  - chart
  - backend
  - web
dependencies: []
references:
  - docs/features/chart-journal.md
priority: high
type: feature
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wertvollster Teil für die Auswertung: Wo lag mein SL, wo wäre er besser gewesen? Heute zeigt der Chart nur offene Positionen. BingX allOrders erlaubt nur 7-Tage-Fenster (gemessen 2026-10-18). Spezifikation: docs/features/chart-journal.md, Abschnitt Stufe 3.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Für einen per SL geschlossenen Trade gilt outcome = STOP_LOSS_HIT und der Exit-Marker liegt am SL
- [ ] #2 rMultiple stimmt mit der Handrechnung an einem Fixture-Trade überein
- [ ] #3 GET /positions/{id}/chart liefert auch für geschlossene Positionen Kerzen (Open − Puffer bis Close + Puffer)
- [ ] #4 Notizen bleiben vom Import unberührt
<!-- AC:END -->
