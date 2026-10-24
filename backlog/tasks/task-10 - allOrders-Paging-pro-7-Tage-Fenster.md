---
id: TASK-10
title: 'allOrders: Paging pro 7-Tage-Fenster'
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - backend
  - bingx
dependencies: []
priority: low
type: enhancement
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
allOrders liefert höchstens 1000 Einträge pro Fenster; heute gibt es dann nur eine Warnung im Log.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Mehr als 1000 Orders in einem Fenster werden vollständig gelesen
<!-- AC:END -->
