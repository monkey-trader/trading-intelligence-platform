---
id: TASK-20
title: 'API: Journal-Notizen als Liste lesen (GET /notes)'
status: To Do
assignee: []
created_date: '2026-10-27T12:39:00Z'
labels:
  - backend
  - api
dependencies:
  - TASK-18
priority: low
type: feature
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Folge aus TASK-18: mehrere Notizen auf einmal mit Filter from, to, limit. Offen: Bezug des Filters (updated_at ist oft NULL, Alternative Join auf positions.close_time); optional hasNote-Flag in /positions/closed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Filterbezug (updated_at oder closeTime) entschieden und in docs/openapi.yaml beschrieben
- [ ] #2 GET /notes mit from, to, limit, Contract-Test gesichert
<!-- AC:END -->
