# Fixtures

## coingecko/

Öffentliche CoinGecko-Antworten (`GET /api/v3/coins/markets`, kein Key), gekürzt auf
`id`, `symbol`, `name`, `image`, `market_cap_rank`.

| Datei | Inhalt |
|---|---|
| `markets.sample.json` | Echte Antwort für `btc,sui,ena,ondo,algo,xrp` |
| `markets.collision.json` | Von Hand gebaut: Duplikat-Symbol `sui` (Rang 28, 900, null), nur Treffer ohne Rang (`ond`), fremder Image-Host (`evil`), `http` statt `https` (`plain`) |

## api/

Antworten der eigenen API (`docs/openapi.yaml`), anonymisiert, Dezimalwerte als Strings, `positionId` als String.

| Datei | Inhalt |
|---|---|
| `open.json`, `closed.json`, `summary.json`, `note.json` | 200-Antworten von `/positions/open`, `/positions/closed`, `/account/summary`, `/positions/{positionId}/note` |
| `problem-404.json`, `problem-503.json` | `application/problem+json` (`not-found`, `not-configured`) |
