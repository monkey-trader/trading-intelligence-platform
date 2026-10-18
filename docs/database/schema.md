# SQLite-Schema

Stand: TypeScript-Backend, Quelle `backend/src/infrastructure/persistence/sqlite/schema.ts`.
Diese Beschreibung ist der Maßstab für das Go-Backend (Migration Phase 4).

## Allgemein
- Datei: `JOURNAL_DB`; Standard im Go-Backend `<UserConfigDir>/tip/journal.db` (macOS: `~/Library/Application Support/tip/journal.db`). Das TS-Backend nutzt weiter `data/journal.sqlite`.
- Beim Öffnen wird zuerst `PRAGMA foreign_keys = ON` gesetzt, danach das Schema idempotent
  angelegt (`CREATE TABLE IF NOT EXISTS`). Die Versionierung steht in `schema_version` (siehe unten).
- **Beträge sind `TEXT`** mit Dezimal-Strings, damit sie exakt erhalten bleiben (nie als Float speichern).
- **`position_id` ist `TEXT`** (19-stellig, würde als Zahl Genauigkeit verlieren).
- Zeitpunkte sind `INTEGER` in Epoch-Millisekunden (UTC).
- Booleans sind `INTEGER` 0/1.

## Tabelle `positions`
Geschlossene Positionen aus `positionHistory`. Eine Zeile pro BingX-Position.

| Spalte | Typ | Bedeutung |
|---|---|---|
| `position_id` | TEXT, PK | BingX-Positions-ID |
| `symbol` | TEXT | z.B. `SUI-USDT` |
| `side` | TEXT | `LONG` oder `SHORT` |
| `isolated` | INTEGER | 1 = Isolated Margin, 0 = Cross |
| `open_time` | INTEGER | Eröffnung |
| `close_time` | INTEGER | Schluss (BingX `updateTime`, danach filtert die API) |
| `avg_open_price` | TEXT | Durchschnittlicher Einstiegspreis |
| `avg_close_price` | TEXT | Durchschnittlicher Ausstiegspreis |
| `realised_profit` | TEXT | Realisierter PnL |
| `net_profit` | TEXT | Netto-PnL nach Gebühren und Funding |
| `position_amount` | TEXT | Positionsgröße |
| `closed_amount` | TEXT | Geschlossene Menge |
| `commission` | TEXT | Gebühren |
| `total_funding` | TEXT | Summe Funding |
| `leverage` | REAL | Hebel (kann Dezimalwert sein) |
| `fully_closed` | INTEGER | 1 = ganz geschlossen |
| `raw` | TEXT | Unveränderte Rohantwort von BingX für diese Position (JSON) |
| `imported_at` | INTEGER | Zeitpunkt des letzten Imports dieser Zeile |

**Offene Positionen** (Go-Backend) stehen als Platzhalter-Zeile in derselben Tabelle: `close_time = 0`, `fully_closed = 0`, `avg_close_price`/`net_profit` etc. `'0'`, `raw` = JSON der offenen Position. Angelegt wird nur, was noch fehlt (`ON CONFLICT DO NOTHING`); der Closed-Import überschreibt die Zeile später per Upsert. Abfragen nach `close_time` müssen `close_time > 0` bzw. `fully_closed = 1` berücksichtigen.

Index `positions_symbol_close_time` auf `(symbol, close_time)`.

**Schreiben nur per Upsert** `INSERT … ON CONFLICT(position_id) DO UPDATE SET …`, alle Spalten außer
`position_id`, in einer Transaktion pro Batch. Nie `INSERT OR REPLACE`: Das löscht die Zeile und
legt sie neu an, was mit dem Fremdschlüssel aus `notes` kollidiert.

## Tabelle `income`
Kontoweite Einträge von BingX (Gebühren, Funding, realisierter PnL), Quelle der Symbol-Suche.

| Spalte | Typ | Bedeutung |
|---|---|---|
| `transaction_id` | TEXT, PK | `tranId` von BingX (19-stellig, als String) |
| `symbol` | TEXT | z. B. `ONDO-USDT` |
| `income_type` | TEXT | z. B. `REALIZED_PNL`, `TRADING_FEE`, `FUNDING_FEE` |
| `amount` | TEXT | Betrag als Dezimal-String (exakt, kein REAL) |
| `time` | INTEGER | Zeitpunkt (Epoch-ms) |
| `asset` | TEXT | z. B. `USDT` |
| `trade_id` | TEXT | Leer bei Funding |
| `raw` | TEXT | Rohantwort |
| `imported_at` | INTEGER | Importzeitpunkt (Epoch-ms) |

Index `income_symbol_time` auf `(symbol, time)`. Schreiben per Upsert über `transaction_id`
(`ON CONFLICT … DO UPDATE`), ein Batch pro 7-Tage-Fenster in einer Transaktion. Fortschritt: `income_scan_until`.

## Tabelle `notes`
Journal-Einträge des Nutzers. **Der Import schreibt hier nie**, und es gibt kein `ON DELETE CASCADE`.

| Spalte | Typ | Bedeutung |
|---|---|---|
| `position_id` | TEXT, PK, FK → `positions.position_id` | Zugehörige Position |
| `setup` | TEXT | Setup / Strategie |
| `rating` | INTEGER | Bewertung |
| `mistakes` | TEXT | Fehler |
| `notes` | TEXT | Freitext |
| `updated_at` | INTEGER | Letzte Änderung |

Noch ohne API (kommt mit dem Notizen-Feature).

## Tabelle `import_state`
Import-Fortschritt, Schlüssel/Wert. `value` ist immer `INTEGER` (Epoch-ms).
Schreiben per `INSERT … ON CONFLICT(key) DO UPDATE`. Präfix-Suche über `substr`, nicht `LIKE`
(`_` und `%` sind keine Platzhalter).

| Schlüssel | Wert | Bedeutung |
|---|---|---|
| `import_started_at` | Zeitpunkt | Erster Serverstart des Go-Backends; Beginn des Imports, wird nie überschrieben. |
| `income_scan_until` | Zeitpunkt | Income ist bis hier nach gehandelten Symbolen durchsucht. Wird erst am Ende eines Laufs geschrieben. |
| `position_history_until:<SYMBOL>` | Zeitpunkt | Positionen dieses Symbols sind bis hier importiert. Nächster Lauf beginnt mit 1 h Überlappung. Verdächtige oder fehlgeschlagene Symbole rücken nicht vor; ohne bisherigen Eintrag bekommen sie den Beginn des Suchbereichs. |
| `position_suspicious:<SYMBOL>` | Zeitpunkt (erstmals gesehen) | Verdacht: Income vorhanden, aber `positionHistory` leer. Bleibt über Läufe bestehen, bis Positionen gefunden werden. |
| `contract_discovery_checked:<SYMBOL>` | `fullImportStart` des Laufs | Einmalige Kontrakt-Suche hat dieses Symbol geprüft. |
| `contract_discovery_completed` | `fullImportStart` des Laufs | Kontrakt-Suche ist vollständig. |

Die beiden `contract_discovery_*`-Werte speichern den Beginn des Vollimports (`IMPORT_FULL_DAYS`,
Standard 365). Wird der Zeitraum größer, gilt die Suche als unvollständig und läuft erneut.
Einen Schlüssel umzubenennen erzwingt einen vollständigen Neu-Import der betroffenen Daten.

## Tabelle `schema_version`
Nur Go-Backend. Spalte `version INTEGER NOT NULL`, genau eine Zeile mit der aktuellen Version
(`CurrentSchemaVersion`, derzeit 1). Eine TS-DB ohne diese Tabelle gilt als Version 0.

- Beim Öffnen einer bestehenden DB mit älterer Version wird vor der Migration eine Sicherung angelegt:
  `<datei>.bak-v<alteVersion>-<UTC 20060102T150405Z>` (Rechte 0600, `O_EXCL`, identischer Inhalt).
  Schlägt sie fehl, wird nicht migriert. Frische oder aktuelle DBs bekommen keine Sicherung.
- Ist die DB-Version höher als die des Programms, wird das Öffnen abgelehnt (`SchemaTooNewError`).
- Die Sicherung liegt neben der DB im selben Verzeichnis; alte Sicherungen löscht der Nutzer selbst.
- Weitere Tabellen (z.B. `balance` aus dem TS-Backend) bleiben unberührt.
