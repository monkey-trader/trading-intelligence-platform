# BingX-Fixtures

Echte BingX-Antworten, anonymisiert, als sprachneutrales JSON. Die Tests des Go-Backends
(`backend/`, Pfad relativ zum Repo-Root) laufen gegen diese Dateien.

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
| `klines.sample.json` | `GET /openApi/swap/v3/quote/klines` | Volle Antwort, 8 Kerzen BTC-USDT 15m (öffentliche Daten). **v3 liefert absteigend** (neueste zuerst), Preise als Strings, `time` als number |
| `klines.error.interval.json` | dito | Fehler `109400`, ungültiges Intervall |
| `klines.error.symbolNotFound.json` | dito | Fehler `109425`, unbekanntes Symbol |
| `openOrders.json` | `GET /openApi/swap/v2/trade/openOrders` | Volle Antwort: ein SL (`STOP_MARKET`), zwei Teil-TPs (`TAKE_PROFIT_MARKET`), eine Entry-Order (`LIMIT`, `positionID` 0). Auslösepreis in `stopPrice`; `positionID` ist hier eine **number** über 2^53 |
| `allOrders.page.json` | `GET /openApi/swap/v2/trade/allOrders` | Volle Antwort: Entry (`MARKET`), ausgelöster SL (`FILLED`), stornierte TP/SL (`CANCELLED`); maximal 7-Tage-Fenster pro Abfrage |
| `income.page.json` | `GET /openApi/swap/v2/user/income` | **Nur das `data`-Array** einer Seite (ohne `code`/`msg`) |
| `contracts.sample.json` | `GET /openApi/swap/v2/quote/contracts` | **Nur das `data`-Array**, Auszug (öffentliche Daten) |

Gemessen am 2026-10-16 gegen die echte API. Die Messergebnisse (Paging ab 0, 90-Tage-Grenze,
Filter nach `updateTime`) stehen in `CLAUDE.md`.

Klines, openOrders und allOrders gemessen am 2026-10-18. Orders sind anonymisiert (synthetische IDs,
Mengen und Preise verändert); `positionID`/`orderId` nicht als Number parsen (Präzisionsverlust).
