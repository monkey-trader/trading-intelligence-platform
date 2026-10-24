---
id: TASK-3
title: Historie nachladen als Nutzeraktion
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - backend
  - web
dependencies: []
priority: medium
type: feature
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Der Import startet erst ab dem ersten Serverstart. RunFullSearch/RunFullSearchUntilDone existieren, sind aber nicht verdrahtet; ältere Trades fehlen deshalb im Journal.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Nutzer kann im UI das Nachladen der Historie starten und sieht Fortschritt bzw. Ende
- [ ] #2 Nachladen respektiert die BingX-Rate-Limits (kein Code 100410)
- [ ] #3 Bereits importierte Positionen und Notizen bleiben unverändert
<!-- AC:END -->
