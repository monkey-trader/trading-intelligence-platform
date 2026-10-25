---
id: TASK-7
title: 'Positionstabelle: Horizontal-Scroll bei 1280 px und Sortierung per Klick'
status: Done
assignee: []
created_date: '2026-10-24T20:01:00Z'
updated_date: '2026-10-25T14:49:00Z'
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
- [x] #1 Bei 1280 px Fensterbreite kein horizontales Scrollen der Tabelle oder Seite
- [x] #2 Spalten (Desktop): Symbol, Wert / Margin (gestapelt), Unreal. PnL, Real. PnL, Einstieg / SL (gestapelt), BE (2R), TP1 (3R); BE und TP1 sind eigene Spalten mit reiner Zahl ohne Präfix, damit sich Preise kopieren lassen
- [x] #3 Klick auf einen Spaltenkopf sortiert aufsteigend, erneuter Klick absteigend, dritter Klick zurück zur Reihenfolge der API; aktive Spalte zeigt Pfeil und aria-sort
- [x] #4 Gestapelte Spalten (Wert / Margin, Einstieg / SL) haben je einen Sortier-Knopf pro Wert
- [x] #5 Klick sortiert, Drag am Kopf verschiebt die Spalte weiter; Symbol bleibt links und ist nicht ziehbar; Ziel wird markiert
- [x] #6 Zustand leer: Köpfe nicht klickbar; Fehler und veraltet: gewählte Sortierung bleibt, Hinweis wie heute
- [x] #7 Lädt: Köpfe nicht klickbar, Platzhalterzeilen
- [x] #8 Mobil unter 768 px: Karten mit Spaltenname aus data-label; Sortierung per Auswahl plus Richtungsknopf mit Touch-Ziel ≥ 44 px
- [x] #9 Long/Short-Balken (3 px, positive/negative) bleibt: Desktop an der Symbol-Zelle, mobil am Kartenrand
- [x] #10 Nur Design-Tokens; Kontrast Text ≥ 4.5:1 und Nicht-Text ≥ 3:1 in Hell und Dunkel
- [x] #11 Zeilen ohne SL (initialStopLoss unbekannt) stehen bei jeder Sortierung am Ende, auch bei absteigend; ihre Reihenfolge untereinander bleibt die der API
- [x] #12 Tastatur: Enter und Leertaste auf dem Spaltenkopf sortieren (Zyklus wie beim Klick); Alt+Pfeil links/rechts verschiebt die Spalte, Symbol bleibt links; Fokus bleibt auf dem verschobenen Kopf und die Änderung wird angesagt (aria-live)
- [x] #13 Chart-Panel bleibt unverändert und bezieht sich auf die gewählte Position (positionId), nicht auf die Zeilenposition: Sortieren oder Spalten verschieben ändert weder die Auswahl noch die Chart-Anzeige
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Reine Sortierlogik (presentation/position/sorting): Sortschlüssel je Spalte/Teilwert, Zyklus asc→desc→API-Reihenfolge, Zeilen ohne SL immer am Ende (AC3, 4, 11).
2. Hook useTableSort (Zustand: Schlüssel+Richtung; bleibt bei Fehler/veraltet, AC6).
3. OpenPositionsTable: neue Spalten (Wert/Margin gestapelt, Einstieg/SL gestapelt, BE, TP1), Sortier-Knöpfe, aria-sort, Drag/Alt+Pfeil mit aria-live, Symbol fix links (AC2–5, 12).
4. CSS: 1280 px ohne Scroll, Karten <768 px mit data-label, Sortierauswahl + Richtungsknopf ≥44 px, Long/Short-Balken, nur Tokens (AC1, 7–10).
5. Chart-Auswahl per positionId prüfen (AC13).
Reihenfolge: test-author → programmer → designer (gegen Prototyp) → reviewer. Kein architect (reine Presentation, API unverändert).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
test-author: 4 neue Testdateien rot (sortPositions, useTableSort, Table.sorting, Page.sorting); bestehende Spaltentests (AC2) noch anzupassen.

programmer: 4 neue Testdateien grün; 56 alte Tests rot wegen AC2 (columnOrder, useColumnOrder, OpenPositionsTable.test/columns/beTarget/beTargetInfo/initialStopLoss). AC1/AC9 visuell offen.

test-author: 7 alte Testdateien an AC2 angepasst, position-Tests 390/390 grün. TP-Assertions des alten TP/SL-Tests entfallen (Ersatz: be/tp1-Spalten). designer läuft.

reviewer: bedingt freigegeben (Alt+Pfeil am Sortier-Knopf, Ansage mit Stelle, Locale de) -> test-author + programmer laufen. designer: AC1/2/3/4/5/6/9/10 geprüft, Chart-Knopf mobil 44px, Balken Desktop/mobil, Dunkel-Kontrast --color-control-border angehoben (design-tokens.css, design.md). OFFEN AC7: Erstladen zeigt nur Text, Platzhalter unerreichbar (OpenPositionsPage rendert Tabelle nur bei positions.length>0).

programmer: Review-Korrekturen umgesetzt, position-Tests 397/397 grün; Ansage 'von Stelle X auf Stelle Y von N'.

AC7: Erstladen rendert Tabelle mit Platzhaltern (OpenPositionsPage); alte Page-Tests auf Warten umgestellt. CI-Liste frisch grün: generate:api ohne Drift, lint, typecheck, format:check, 526 Tests, build.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Positionstabelle: Spalten Symbol, Wert/Margin, Unreal. PnL, Real. PnL, Einstieg/SL, BE (2R), TP1 (3R); Sortierung per Klick (asc, desc, API-Reihenfolge) mit aria-sort, je Wert ein Knopf bei gestapelten Spalten, Zeilen ohne SL immer am Ende; Alt+Pfeil verschiebt Spalten (auch vom Sortier-Knopf, mit Ansage von Stelle zu Stelle), Symbol fix links; Erstladen zeigt Tabelle mit Platzhaltern; mobil Karten mit Sortierauswahl und Richtungsknopf. Rein Client-seitig, API unverändert. Geprüft: CI-Liste frisch grün (generate:api ohne Drift, lint, typecheck, format:check, 526 Tests, build); designer misst im Browser 0 px Überlauf bei 1280 px (hell/dunkel), Touch-Ziele 44 px, Balken und Kontraste; reviewer zweimal (bedingt freigegeben, Befunde umgesetzt, dann freigegeben). Abweichungen: alter TP-Anzeige-Test entfiel (TP-Preise jetzt in be/tp1-Spalten getestet); Seitentests warten auf Platzhalter-Ende; --color-control-border dunkel #928374 -> #a89984 (app-weit, Kontrast 2.40 -> 3.17:1); Ansage-Text ist 'von Stelle A auf Stelle B von N'. Risiken/offen: 768-1024 px scrollt der Container weiterhin (nicht Teil von AC1); Extremwerte (36-stelliges Symbol, 7-stellige Beträge) brauchen 1200 px; Layout-Sprung 24/59 px beim ersten Laden (wie Prototyp); Drag nur simuliert, echte Touch-Geräte und mobile Dunkelkarten nur gemessen, nicht gesehen; tabIndex=0 an Scroll-Region unter 768 px.
<!-- SECTION:FINAL_SUMMARY:END -->
