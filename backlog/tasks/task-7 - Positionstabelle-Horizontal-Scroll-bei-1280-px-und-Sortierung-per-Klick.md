---
id: TASK-7
title: 'Positionstabelle: Horizontal-Scroll bei 1280 px und Sortierung per Klick'
status: To Do
assignee: []
created_date: '2026-10-24T20:01:00Z'
updated_date: '2026-10-24T20:41:00Z'
labels:
  - web
dependencies:
  - TASK-14
references:
  - docs/design/TASK-7-positionstabelle-sortierung/prototype.html
  - docs/design/TASK-7-positionstabelle-sortierung/flow.png
priority: low
type: enhancement
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Seit v1.0.0 scrollt die Positionstabelle bei 1280 px horizontal, und Zeilen lassen sich nicht sortieren. Der Trader soll alle Spalten ohne Scrollen sehen und per Klick auf den Spaltenkopf sortieren. Design: Variante B, freigegeben 2026-10-24 (Prototyp und Flow siehe Referenzen). Sortiert wird nur im Client auf den geladenen Zeilen, die API ändert sich nicht.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Bei 1280 px Fensterbreite kein horizontales Scrollen der Tabelle oder Seite
- [ ] #2 Spalten (Desktop): Symbol, Wert / Margin (gestapelt), Unreal. PnL, Real. PnL, Einstieg / SL (gestapelt), BE (2R), TP1 (3R); BE und TP1 sind eigene Spalten mit reiner Zahl ohne Präfix, damit sich Preise kopieren lassen
- [ ] #3 Klick auf einen Spaltenkopf sortiert aufsteigend, erneuter Klick absteigend, dritter Klick zurück zur Reihenfolge der API; aktive Spalte zeigt Pfeil und aria-sort
- [ ] #4 Gestapelte Spalten (Wert / Margin, Einstieg / SL) haben je einen Sortier-Knopf pro Wert
- [ ] #5 Klick sortiert, Drag am Kopf verschiebt die Spalte weiter; Symbol bleibt links und ist nicht ziehbar; Ziel wird markiert
- [ ] #6 Zustand leer: Köpfe nicht klickbar; Fehler und veraltet: gewählte Sortierung bleibt, Hinweis wie heute
- [ ] #7 Lädt: Köpfe nicht klickbar, Platzhalterzeilen
- [ ] #8 Mobil unter 768 px: Karten mit Spaltenname aus data-label; Sortierung per Auswahl plus Richtungsknopf mit Touch-Ziel ≥ 44 px
- [ ] #9 Long/Short-Balken (3 px, positive/negative) bleibt: Desktop an der Symbol-Zelle, mobil am Kartenrand
- [ ] #10 Nur Design-Tokens; Kontrast Text ≥ 4.5:1 und Nicht-Text ≥ 3:1 in Hell und Dunkel
- [ ] #11 Zeilen ohne SL (initialStopLoss unbekannt) stehen bei jeder Sortierung am Ende, auch bei absteigend; ihre Reihenfolge untereinander bleibt die der API
- [ ] #12 Tastatur: Enter und Leertaste auf dem Spaltenkopf sortieren (Zyklus wie beim Klick); Alt+Pfeil links/rechts verschiebt die Spalte, Symbol bleibt links; Fokus bleibt auf dem verschobenen Kopf und die Änderung wird angesagt (aria-live)
<!-- AC:END -->
