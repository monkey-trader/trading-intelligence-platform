---
id: TASK-4
title: 'Desktop-App Variante 2: Server läuft unabhängig vom Fenster'
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
labels:
  - desktop
dependencies: []
priority: medium
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Seit der Ein-App-Lösung läuft der 5-Minuten-Import nur bei offener App, und ein abgestürzter App-Prozess lässt den Server als Waise zurück.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Menüleisten-Symbol mit Öffnen und Beenden
- [ ] #2 Fenster schließen beendet den Server nicht; Beenden über die Menüleiste schon
- [ ] #3 Option Beim Login starten
- [ ] #4 Nach einem Absturz der App läuft kein verwaister Server weiter bzw. wird beim nächsten Start übernommen
<!-- AC:END -->
