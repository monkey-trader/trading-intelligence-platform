---
id: TASK-22
title: 'BE-Auslöser: veraltete 5m-Kerzen nicht als „nicht erreicht“ werten'
status: Done
assignee: []
created_date: '2026-10-27T19:40:00Z'
updated_date: '2026-10-27T20:14:00Z'
labels:
  - positions
dependencies: []
priority: medium
type: bug
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Am 2026-10-27 zeigt GET /positions/open für ENA-USDT breakEvenTriggerReached=false und kein reached2RAt, obwohl der Kurs bei 2.39R liegt und der SL auf BE steht. Ursache: twoRStatus liest 5m-Kerzen nur aus dem SQLite-Cache. Für ENA enden sie am 2026-10-18 21:00 (300 Stück, höchstes Hoch 0.19328, unter dem 2R-Level 0.20561); im Backend schreibt kein Code 5m-Kerzen (nur 15m werden gepflegt). ReachedAt findet nichts und liefert (nil, nil), twoRStatus macht daraus false statt unbekannt. Teil 1: 5m-Kerzen aktuell halten oder auf ein gepflegtes Intervall umstellen. Teil 2: twoRStatus liefert nil (unbekannt), wenn die Kerzen den Zeitraum seit Eröffnung bis jetzt nicht abdecken. Fachliche Quellen: docs/domain.md (BE-Auslöser erreicht)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 twoRStatus liefert (nil, nil), wenn die letzte Kerze deutlich vor dem Abfragezeitpunkt liegt (veraltet)
- [x] #2 twoRStatus liefert (nil, nil), wenn die erste Kerze nach der Eröffnung beginnt (Lücke am Anfang)
- [x] #3 Lückenlose, aktuelle Kerzen verhalten sich unverändert (true/false wie bisher)
- [x] #4 Teil 1: 5m-Kerzen werden aktuell gehalten oder die Quelle wird auf ein gepflegtes Intervall umgestellt (offen, Folgearbeit)
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Intervall 15m statt 5m (Nutzer bestätigt 2026-10-27: Auflösung von reached2RAt 15 min reicht). 2. twoRStatus holt vor dem Cache-Lesen frische Kerzen über CandleSource und schreibt per UpsertCandles; BingX-Fehler -> Cache lesen, coversSinceOpen entscheidet. 3. Limit aus Haltedauer ableiten; über BingX-Obergrenze (1440) -> unbekannt. 4. Verdrahtung in cmd/trading-intelligence. 5. docs/domain.md (Auflösung 15m). Optional später: TTL-Cache ~60 s je (Symbol, Intervall).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Teil 2 umgesetzt (lead, inline TDD): coversSinceOpen in getopenpositions.go; 5 neue/angepasste Tests; gofmt, vet, golangci-lint, go test -race grün. Fixtures: now=riskNow (hinter letzter Kerze), Kerzenreihen beginnen ab Eröffnung. Teil 1 (5m-Kerzen pflegen) offen, deshalb nicht Done.

architect: DONE_WITH_CONCERNS, Anforderungs-Check ohne Widerspruch. Ursache: UpsertCandles läuft nur im Chart-Use-Case (Default 15m), kein Scheduler. Empfehlung Option c angenommen.

test-author: DONE. 4 Tests rot (Fresh*, WithCandleSource-Stub ruft nichts ab), übrige grün; Fixtures auf 15m, riskNow=1_791_600_900_000. limit=ceil((now-openTime)/15min)+Puffer 1..10; >1440 Intervalle: kein Abruf, 2R unbekannt.

programmer: DONE. refreshCandles (Fetch->Upsert->List, 15m, limit=ceil+2, >1440 unbekannt), Verdrahtung in main.go, docs/domain.md + schema.md. Lead-Check: gofmt/vet/lint/test -race grün.

reviewer: freigegeben mit Anmerkungen. Offen: TTL-Cache (30-60 s) gegen einen BingX-Request pro Position bei jedem GET /positions/open (Folgearbeit, nicht umgesetzt); Test Fetch-Fehler+Cache vorhanden (TestFresh_FetchFehlerRechnetMitCacheWeiter); Hinweis zum 1000er-Limit in domain.md ergänzt. AC 4 noch nicht abgehakt: fehlt der Nachweis am laufenden Backend (ENA nach Neustart).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
twoRStatus wertete veraltete 5m-Kerzen als nicht erreicht (ENA false trotz 2.39R). Jetzt: coversSinceOpen liefert unbekannt (null) bei Lücke oder veralteten Kerzen; refreshCandles holt 15m-Kerzen frisch über CandleSource (Fetch->Upsert->List, limit aus Haltedauer, >1440 Intervalle unbekannt). Geprüft: gofmt, vet, golangci-lint, go test -race grün; Branch-Build gegen Kopie der Produktiv-DB auf Port 3001: ENA breakEvenTriggerReached=true, reached2RAt=1792567800000. Risiken/Folgearbeit: ein BingX-Request pro offener Position bei jedem GET /positions/open (TTL-Cache bewusst nicht umgesetzt); reached2RAt ist auf 15 min genau.
<!-- SECTION:FINAL_SUMMARY:END -->
