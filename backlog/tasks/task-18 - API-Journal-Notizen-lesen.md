---
id: TASK-18
title: 'API: Journal-Notizen lesen'
status: Done
assignee: []
created_date: '2026-10-27T00:00:00Z'
updated_date: '2026-10-27T12:46:00Z'
labels:
  - backend
  - api
dependencies: []
priority: medium
type: feature
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Teil von DRAFT-3. Die Tabelle `notes` (setup, rating, mistakes, notes) existiert, hat laut docs/database/schema.md aber noch keine API. Diese Task macht die Einträge lesbar, damit Clients Trade und Notiz zusammen sehen. Schreiben (Anlegen, Ändern) ist bewusst nicht Teil dieser Task und kommt mit dem Notizen-Feature (siehe DRAFT-1 und DRAFT-2).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `GET /positions/{positionId}/note` ist in docs/openapi.yaml beschrieben; 200 mit Notiz, 404 wenn keine existiert, Format der Fehler wie bisher (`problem.go`)
- [x] #2 Felder: positionId, setup, rating, mistakes, notes, updatedAt (Zeitpunkt)
- [x] #3 Der Endpunkt schreibt nichts; die Notiz lässt sich nicht ohne zugehörige Position lesen (Fremdschlüssel bleibt gewahrt)
- [x] #4 Tests zuerst, `go test -race ./...`, golangci-lint und gofmt sauber
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Freigabe 2026-10-27: Schnitt laut architect (Feature journal, GET /positions/{positionId}/note, 404 ohne Notiz, 503 not-configured ohne Datenbank mit neutralem Text, updatedAt Epoch-ms). AC #3 (GET /notes) entfällt, kommt als Folge-Task. Keine Einbettung in /positions/closed. 1. openapi.yaml + schema.md zuerst. 2. test-author. 3. programmer. 4. reviewer, unify.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
architect (2026-10-27): Anforderungs-Check ohne Widerspruch. Schnitt: Feature journal, Entity JournalNote, Port NoteRepository.FindByPositionID, NoteNotFoundError (404), Use Case GetPositionNote, sqlite-Adapter (nur SELECT), Route GET /positions/{positionId}/note. updatedAt = Epoch-ms wie übrige Spec. Empfehlung: AC #3 (GET /notes) nicht bauen (from/to-Filter fachlich offen: updated_at oft NULL), nicht in /positions/closed einbetten. Rating-Wertebereich undefiniert, Spec nur 'integer, nullable'. Wartet auf Freigabe des Nutzers.

Lead: Freigabe erteilt (AC #3 -> TASK-20, kein Einbetten, 503 not-configured ohne DB). openapi.yaml (getPositionNote, JournalNote, Tag Journal) + schema.md angepasst, redocly lint grün, Web-API-Typen neu generiert.

test-author: Tests rot (Compile-Fehler, Paket journal fehlt) in domain/journal, application/journal, sqlite, httpapi + Contract-Test; problem_test: Titel not-configured neutral 'Der Dienst ist nicht konfiguriert.'

reviewer: freigegeben. Behoben: updatedAt in Spec als integer (0 = kein Zeitpunkt gespeichert) statt nullable, wie die Implementierung; Imports gruppiert. Offen als Folgepunkte: Fehlerart für 'ohne DB' ohne Kopplung an Position-Domain; positionId-Prüfung nur in Domain (Handler mappt InvalidPositionIDError); Name JournalNote/nolint.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
GET /positions/{positionId}/note liefert die Journal-Notiz (positionId, setup, rating, mistakes, notes, updatedAt als Epoch-ms, 0 wenn unbekannt); 404 ohne Notiz, 400 bei ungültiger ID, 503 not-configured ohne Datenbank (Titel neutral 'Der Dienst ist nicht konfiguriert.'). Nur SELECT, Feature journal nach Onion. GET /notes (frühere AC #3) ausgegliedert nach TASK-20, keine Einbettung in /positions/closed. Geprüft: gofmt, go vet, golangci-lint (0 issues), go test -race grün; Contract-Tests 145 grün gegen keyless Backend; redocly lint, contracts lint/typecheck/format, web typecheck grün. Risiken: Contract-Test deckt nur die DB-lose Variante ab (200 nur in Go-Tests); Zeitpunkt 0 als Sentinel.
<!-- SECTION:FINAL_SUMMARY:END -->
