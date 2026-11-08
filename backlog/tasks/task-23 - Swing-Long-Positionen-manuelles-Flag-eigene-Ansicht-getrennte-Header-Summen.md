---
id: TASK-23
title: 'Swing-Long-Positionen: manuelles Flag, eigene Ansicht, getrennte Header-Summen'
status: Done
assignee: []
created_date: '2026-11-08T13:51:00Z'
updated_date: '2026-11-08T14:33:00Z'
labels:
  - feature
dependencies: []
priority: medium
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Swing-Long-Positionen (z. B. BTC) dürfen ins Minus gehen und sollen die Summen der übrigen Trades nicht herunterziehen. Entscheidung Nutzer: manuelles Flag pro Position (PUT/DELETE-Endpoint wie Ursprungs-SL, Flag in DB, bleibt nach Schliessen erhalten); Ansicht als Tab/Umschalter im UI (Trades | Swing Long); Header zeigt Summen getrennt.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 OpenAPI: Endpoint zum Setzen/Entfernen des Swing-Long-Flags, Feld swingLong in offenen Positionen
- [x] #2 Offene Positionen: Summen ohne Swing-Long; Swing-Long-Summe separat
- [x] #3 Kontoweite Header-Werte trennen Swing-Long klar vom Rest
- [x] #4 Web: Tab Trades / Swing Long, Umschalter pro Zeile zum Markieren
- [x] #5 Tests (Go + Vitest + Vertrag) grün
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Umgesetzt: manuelles Flag (Tabelle position_swing_long, Schema v2), PUT/DELETE /positions/{id}/swing-long, Summen offen/Konto/Monat ohne Swing Long, Reiter Trades|Swing Long, Header passt sich dem Reiter an. Entscheidungen: realisierter PnL per Näherung Symbol+Zeitraum (siehe docs/domain.md); Heute/7T/30T ohne Swing Long, ausgewiesen wird nur der Monatswert. Offen: docs/design.md um Baustein Schalter pro Zeile ergänzen; Reiter ohne Pfeiltasten; Account-Summary schlägt fehl, wenn BingX-Positionen bei gesetztem Flag nicht abrufbar sind.
<!-- SECTION:NOTES:END -->
