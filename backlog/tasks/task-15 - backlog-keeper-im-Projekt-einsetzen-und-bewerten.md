---
id: TASK-15
title: backlog-keeper im Projekt einsetzen und bewerten
status: To Do
assignee: []
created_date: '2026-10-24T20:49:00Z'
labels:
  - tooling
dependencies: []
references:
  - >-
    https://github.com/ai-c3smonkey/ai-os/blob/main/ai/agents/development/backlog-keeper.md
priority: low
type: chore
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
In ai-os liegt der Haiku-Agent backlog-keeper für Massenarbeit im Backlog (viele Tasks anlegen, aufräumen, Statusbericht). Er ist bewusst noch nicht verlinkt: erst einsetzen, wenn echte Massenarbeit ansteht, und prüfen, ob er Tokens spart und der Lead ihn nicht für Einzeländerungen missbraucht.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 development/backlog-keeper ist in .claude/agents.txt eingetragen und per ai-link verlinkt
- [ ] #2 Einmal einen Statusbericht und einmal ein Aufräumen damit ausgeführt, Ergebnis als Notiz festgehalten
- [ ] #3 Kosten verglichen (Kontext/Kosten in der Statuszeile vorher und nachher) und Entscheidung notiert: behalten, anpassen oder entfernen
- [ ] #4 In einer Session geprüft, dass der Lead einzelne Status-Änderungen weiter selbst macht
<!-- AC:END -->
