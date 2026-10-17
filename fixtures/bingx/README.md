# BingX-Fixtures

Echte BingX-Antworten, anonymisiert, als sprachneutrales JSON. Das TypeScript-Backend
(`backend/`, Import über den Alias `@fixtures/bingx/...`) und später das Go-Backend
(`backend-go/`, Pfad relativ zum Repo-Root) testen gegen dieselben Dateien.

## Regeln
- Inhalte bleiben **exakt wie von BingX geliefert**, auch Zahlen-Strings wie `"0.8650"`
  (keine Formatierung durch Prettier oder Editor).
- Keine echten Konto-Daten: IDs (`positionId`, `tranId`, `tradeId`) sind durch
  synthetische 19-stellige Werte ersetzt, Beträge und Zeitstempel sind verändert,
  Struktur und Feldnamen sind unverändert.
- Neue Fixtures nur anonymisiert ablegen und hier eintragen.

## Dateien

| Datei | BingX-Endpunkt | Inhalt |
|---|---|---|
| `positionHistory.page0.json` | `GET /openApi/swap/v1/trade/positionHistory` | Volle Antwort, erste Seite (`pageIndex=0`) mit Positionen |
| `positionHistory.short.json` | dito | Volle Antwort, eine SHORT-Position |
| `positionHistory.empty.json` | dito | Volle Antwort, leere Liste (`code: 0`) |
| `positionHistory.error.rangeTooLarge.json` | dito | Fehler `109400`, Zeitraum über 90 Tage |
| `positionHistory.error.symbolRequired.json` | dito | Fehler `109400`, `symbol` fehlt |
| `openPositions.json` | `GET /openApi/swap/v2/user/positions` | Volle Antwort mit offenen Positionen (Hedge-Modus; `liquidationPrice` als number) |
| `openPositions.empty.json` | dito | Volle Antwort ohne offene Positionen |
| `income.page.json` | `GET /openApi/swap/v2/user/income` | **Nur das `data`-Array** einer Seite (ohne `code`/`msg`) |
| `contracts.sample.json` | `GET /openApi/swap/v2/quote/contracts` | **Nur das `data`-Array**, Auszug (öffentliche Daten) |

Gemessen am 2026-10-16 gegen die echte API. Die Messergebnisse (Paging ab 0, 90-Tage-Grenze,
Filter nach `updateTime`) stehen in `CLAUDE.md`.
