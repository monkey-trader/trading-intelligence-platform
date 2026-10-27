# Fachwissen der Anwendung (Domain-Glossar)

Einzige Quelle dafür, **was die Anwendung unter ihren Begriffen rechnet**. Änderungen an Begriffen oder
Formeln gehören zuerst hierher und in `docs/openapi.yaml`, dann in den Code. Die **Strategie des Nutzers**
(Warum, Handelsregeln, Checkliste) liegt im Vault `ai-os`, Notiz `03 Areas/Trading/Trading Strategie Regeln.md`.
Dieses Glossar verweist darauf, kopiert sie aber nicht.

Stand: 2026-10-27. „bestätigt“ heißt: vom Nutzer ausdrücklich so festgelegt.

## R und Level
| Begriff | Bedeutung | API / Code |
|---|---|---|
| **R** | Abstand zwischen Entry und **ursprünglichem Stop-Loss** (nicht dem aktuellen SL). Bei LONG liegt der SL unter dem Entry, bei SHORT darüber. | `domain/position/rlevels.go` |
| **Ursprünglicher SL** | Der zuerst gesetzte SL der Position auf der Verlustseite. Quelle: Tabelle `position_risk` (beim ersten Sehen der Position gespeichert), sonst frühester STOP-Order-Trigger aus der `allOrders`-Historie, sonst vom Nutzer manuell gesetzt. Ohne SL oder SL = Entry ist das Risiko unbekannt. | `initialStopLoss`, `initialStopLossSource` |
| **BE-Level (2R)** | Der **Preis bei 2× R**, an dem der SL auf Break-even gezogen wird: LONG Entry + 2R, SHORT Entry − 2R. **Bestätigt (2026-10-27).** Das ist nicht der Entry-Preis. | Stufe `TWO_R`, Spalte „BE (2R)“ |
| **BE-Stop** | Der SL liegt auf dem Entry-Preis. Ziel, sobald das BE-Level erreicht ist. | Feld `breakEven` (SL aktuell nicht im Verlust) |
| **BE-Auslöser erreicht** | Der Kurs hat seit Eröffnung das BE-Level erreicht (Hoch/Tief der Kerzen), bleibt danach `true`. Die Kerzen sind 15-Minuten-Kerzen (Auflösung von `reached2RAt` 15 min); ohne lückenlose, aktuelle Kerzen ist der Wert unbekannt (`null`). Ohne Kerzenquelle liest die API nur den Cache (höchstens 1000 Kerzen, ca. 10 Tage); ältere Positionen bleiben dann unbekannt. | `breakEvenTriggerReached`, `reached2RAt` |
| **TP1-Level (3R)** | Der Preis bei 3× R: LONG Entry + 3R, SHORT Entry − 3R. Liegt ein offener TP dort, wird er mit dem Level zusammengelegt („TP = 3R“, Berechnung im Backend). | Stufe `THREE_R`, `coincidesWithTakeProfit` |

Es gibt **keine** 1R-Stufe und kein TP2/TP3 (seit TASK-14). Die Aufteilung der Restmenge nach TP1 ist nicht modelliert
(laut Strategie üblich 33 % bei TP1, nur ein Richtwert).

## Ergebnis einer geschlossenen Position
| Begriff | Bedeutung | Stand |
|---|---|---|
| **R-Ergebnis** (`rResult`) | Gewinn oder Verlust geteilt durch den Risikobetrag: `netProfit ÷ (|Entry − SL| × positionAmount)`, Einheit R (zum Beispiel +3 oder −1). `null`, wenn das ursprüngliche Risiko unbekannt ist, nie geschätzt. Berechnung im Backend. | Formel aus TASK-17, **offen:** netto (`netProfit`, nach Gebühren und Funding) oder brutto (`realisedProfit`). Bisherige Annahme: netto. |

## Querschnittsregeln
- Beträge und Preise sind **Dezimal-Strings**, nie Float (Datenbank `TEXT`).
- Zeitpunkte sind Epoch-Millisekunden (UTC).
- Der Client rechnet nichts: Summen, Abstände, R-Werte und Level kommen aus dem Backend.
- Die Anwendung ist nur lesend gegenüber BingX; sie schreibt nur in die lokale Datenbank.
