---
id: TASK-9
title: 2R-Suche auch für ältere Positionen
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - backend
  - chart
dependencies: []
priority: low
type: bug
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Die 2R-Suche schaut nur in die letzten 1000 5-Minuten-Kerzen (ca. 3,5 Tage). Ältere Positionen können ein früheres Erreichen von 2R verpassen.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 reached2RAt ist auch für Positionen korrekt, die älter als 3,5 Tage sind (Test mit konstruierter Kerzenreihe)
<!-- AC:END -->
