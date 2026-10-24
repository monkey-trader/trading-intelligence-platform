---
id: TASK-8
title: Negativ-Cache für Orders-Fehler
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - backend
  - bingx
dependencies: []
priority: low
type: enhancement
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Bei Orders-Fehlern fragt jeder Poll erneut bei BingX an; das belastet das Rate-Limit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Nach einem Orders-Fehler wird für eine feste Zeit nicht erneut angefragt (Test belegt das)
<!-- AC:END -->
