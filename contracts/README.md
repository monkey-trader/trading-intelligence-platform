# Vertragstest (contracts)

Sprachneutraler Vertragstest für die HTTP-API. Er spricht nur über HTTP mit einem
**laufenden** Backend und prüft die Antworten gegen `../docs/openapi.yaml`. So muss jedes
Backend (TypeScript heute, Go später) denselben Vertrag erfüllen. Es gibt keine Imports
aus `../backend` oder `../clients`.

Geprüft wird:

- jede Operation der Spec: Status, Content-Type und Body gegen das Schema (Ajv, JSON Schema 2020-12)
- die allgemeinen Regeln aus `info.description`: Host-Prüfung vor dem Routing (403 `forbidden-host`,
  ohne Groß-/Kleinschreibung, auch bei ungültigen Pfaden), 404 `not-found` für unbekannte Pfade und
  Methoden (auch `HEAD`, `OPTIONS`, `DELETE`), unabhängig vom Body (z.B. kaputtes JSON),
  400 `bad-request` für formal fehlerhafte Anfragen (z.B. ungültige Prozent-Kodierung `/%zz`),
  Fehler als `application/problem+json` ohne interne Details, `Retry-After` nur bei 503 `rate-limited`
- ohne Server: jede `operationId` der Spec hat einen Vertragstest (`src/contract/coverage.ts`)

## Aufruf

```bash
npm ci
# Backend ohne BingX-Keys starten:
(cd ../backend && env -u BX_KEY -u BX_SECRET PORT=3999 go run ./cmd/tip-server &)
CONTRACT_BASE_URL=http://127.0.0.1:3999 npm test
pkill -f "tip-server"

npm run test:unit   # nur die Tests ohne Server (Validator, HTTP-Helfer, Abdeckung)
npm run lint && npm run typecheck && npm run format:check
```

## Umgebungsvariablen

| Variable                     | Pflicht | Bedeutung                                                                                                                                      |
| ---------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `CONTRACT_BASE_URL`          | ja      | Basis-URL der laufenden API, nur Schema, Host und Port (z.B. `http://127.0.0.1:3000`). Fehlt sie, bricht der Test mit einer klaren Meldung ab. |
| `CONTRACT_EXPECT_CONFIGURED` | nein    | `true`: Das Backend hat BingX-Keys, `GET /positions/open` muss 200 liefern. Sonst wird 503 `not-configured` erwartet.                          |

Für den Vertragstest reicht ein Backend ohne Keys. Echte Keys nur mit Lese-Rechten und nie in
dieses Paket oder in Logs.
