---
id: TASK-5
title: Desktop-App signieren und notarisieren
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - desktop
  - release
dependencies: []
priority: medium
type: chore
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Die App ist nur ad-hoc signiert; die Cask entfernt die Quarantäne per postflight. Für eine saubere Installation braucht es Developer-ID-Signierung und Notarisierung.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Release-Job signiert mit Developer-ID und notarisiert die App
- [ ] #2 Cask braucht kein Entfernen der Quarantäne mehr
<!-- AC:END -->
