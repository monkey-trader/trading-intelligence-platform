---
id: DRAFT-6
title: Mission Control – Entwicklungsprozess sichtbar machen
status: Draft
assignee: []
created_date: '2026-10-27T06:00:00Z'
labels:
  - mission-control
dependencies: []
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Das Mission Control zeigt nicht nur Trades, sondern auch den **Entwicklungsprozess** mit den Agents: wer hat welchen Task, in welcher Station, mit welchem Ergebnis. Anregung: Paperclip (Org-Chart und Tickets für Agents). Zu Paperclip selbst ist hier nichts geprüft, es dient nur als Idee.

Heute zeigt das Backlog-Board (`backlog browser`) nur To Do, In Progress und Done. Der Prozess dazwischen (Anforderungs-Check, Tests, Code, Review, Abschluss) und die Übergaben zwischen den Agents sind nur im Terminal und in den Task-Notizen sichtbar.

**Stationen pro Task** (Laufbahn): `architect` (Anforderungs-Check, Schnitt) → `test-author` → `programmer` → `designer` (nur UI) → `reviewer` → `unify` → Done. Je Station Status (`DONE`, `DONE_WITH_CONCERNS`, `NEEDS_CONTEXT`, `BLOCKED`), Zeit, Kurzbericht.

**Inbox „braucht dich“:** Jeder `NEEDS_CONTEXT` und jedes `BLOCKED` erscheint als Karte mit den Fundstellen nebeneinander (z. B. Task, Vault-Notiz, `openapi.yaml`) und einem Antwortfeld. Der Widerspruch bei BE (Entry gegen 2R-Preis) wäre dort aufgetaucht.

**Datenwege:**
1. Aus dem Backlog lesen: die Notizen der Tasks enthalten die Agent-Schritte („test-author: 4 Tests rot …“). Reicht für eine erste, nur lesende Ansicht, ohne neuen Baustein. Nachteil: Freitext, wenig Struktur.
2. Ereignis-Log: jeder Agent-Schritt schreibt eine Zeile (Task, Agent, Status, Zeit, Kurzbericht), z. B. per Claude-Code-Hook beim Ende eines Subagents. Saubere Zeitleiste, aber ein eigener Baustein.
Empfehlung: mit 1 starten, 2 nachziehen, wenn die Ansicht sich lohnt.

**Abgrenzung:** Eigener Ausbau nach dem Journal-MCP (TASK-17 bis TASK-19). Das Mission Control ist ein weiterer Client derselben API. Offene Frage: Wohnt es im Trading-Repo oder in ai-os? Der Prozess betrifft mehrere Repos und die Agents kommen aus ai-os.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Entwurf einer Ansicht „Prozess“: Laufbahn je Task mit den Stationen und dem Status je Station, Freigabe durch den Nutzer (ui-prototyping)
- [ ] #2 Die Ansicht liest aus den Backlog-Dateien (Datenweg 1); Quelle und Parser sind im Plan beschrieben
- [ ] #3 Die Inbox „braucht dich“ zeigt alle Tasks mit `NEEDS_CONTEXT` oder `BLOCKED` samt Fundstellen
- [ ] #4 Entscheidung: Ereignis-Log (Datenweg 2) ja oder nein, mit Format der Zeilen
- [ ] #5 Entscheidung: Wohnort des Mission Control (Trading-Repo oder ai-os) und Zugriff auf mehrere Repos
<!-- AC:END -->
