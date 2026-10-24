---
id: TASK-13
title: Skills und Agents zentral aus ai-os, Backlog.md einführen
status: Done
assignee: []
created_date: '2026-10-24T20:01:00Z'
updated_date: '2026-10-24T20:01:00Z'
labels:
  - tooling
dependencies: []
priority: medium
type: chore
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Skills und Agents lagen als Kopien im Repo, der Projektstand als wachsender Fließtext in docs/migration/go-wails.md. Ziel: eine zentrale Quelle (ai-os) und ein Backlog, in dem Status und offene Arbeit immer sichtbar sind.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Agents und Skills kommen über .claude/agents.txt und .claude/skills.txt aus ai-os
- [x] #2 Offene Punkte aus go-wails.md und der Vault-Notiz sind als Tasks bzw. Drafts erfasst
- [x] #3 CLAUDE.md und agent-orchestration verweisen auf den Backlog statt auf den Status-Abschnitt
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
PR #38. Offene Punkte übernommen: TASK-1 bis TASK-12, Ideen als DRAFT-1 bis DRAFT-5. Der Abschnitt Aktueller Stand in docs/migration/go-wails.md bleibt als Verlauf bis 2026-10-24 stehen, wird aber nicht mehr fortgeschrieben.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Agents (6) und Skills (8) werden per ai-link aus ai-os verlinkt; in Claude Code geprüft: alle Agents gelistet, architect liest .claude/agent-rules.md, /skills zeigt die ai-os-Skills als project. Offene Punkte als TASK-1 bis TASK-12 und DRAFT-1 bis DRAFT-5 erfasst. CLAUDE.md und agent-orchestration verweisen auf den Backlog; go-wails.md ist nur noch Verlauf. Risiko: ohne ai-os-Klon fehlen Agents und Skills (Hinweis oben in CLAUDE.md).
<!-- SECTION:FINAL_SUMMARY:END -->
