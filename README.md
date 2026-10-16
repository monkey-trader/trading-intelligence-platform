# Trading Intelligence Platform

Persönliches Trading-Journal mit automatischem Import von BingX (USDT-M Futures).

- **API-first:** Ein eigenes Backend besitzt Import, Datenbank und Fachlogik.
  Clients (Web-UI, Skripte, …) sind austauschbar und sprechen nur über die HTTP/JSON-API.
- **Architektur:** Onion / DDD, feature-sliced, Test-Driven.
- **Stack:** Node.js + TypeScript, SQLite, Vitest, npm, GitHub Actions.
- **Referenz:** Der Python-Prototyp `../bingx-journal` (siehe CLAUDE.md, inkl. bekanntem Positions-Bug).

Details und Regeln für die Entwicklung stehen in [`CLAUDE.md`](CLAUDE.md).

## Lokal starten
Einmalig installieren und die Keys hinterlegen:
```bash
(cd backend && npm ci) && (cd clients/web && npm ci)
cp backend/.env.example backend/.env   # BX_KEY und BX_SECRET eintragen (Key nur mit Lese-Rechten)
```
Dann zwei Terminals:
```bash
cd backend && npm run dev        # Terminal 1: API auf http://127.0.0.1:3000
cd clients/web && npm run dev    # Terminal 2: Web-UI, Proxy /api → Backend
```
Danach http://127.0.0.1:5173 öffnen (Vite bindet nur auf IPv4-Loopback; `localhost` kann auf `::1` zeigen).

---

## Entwickeln mit AI-Agenten (Blueprint)

Dieses Repo wird mit [Claude Code](https://code.claude.com) entwickelt. Statt einem einzigen
"Alles-Assistenten" gibt es **feste Rollen**, die sich gegenseitig kontrollieren.
Das Setup ist bewusst einfach gehalten und lässt sich auf andere Projekte übertragen.

### Bausteine

```
CLAUDE.md                     Projektgedächtnis: Ziel, Architektur, Regeln, Sicherheit
.claude/agents/
├── architect.md              prüft Schichtengrenzen und API-Vertrag (nur lesen)
├── test-author.md            schreibt Tests zuerst, nie Produktivcode
├── programmer.md             macht Tests grün, ändert keine Tests
└── reviewer.md               Qualität + Sicherheit vor dem Commit (nur lesen)
```

**`CLAUDE.md`** wird bei jedem Start automatisch geladen. Hier steht alles, was *immer* gilt:
Ziel, Stack, Architektur, Konventionen, Sicherheitsregeln, bekannte Probleme.
Unterordner können eigene `CLAUDE.md` haben, die nur dort gelten.

**Subagents** sind Markdown-Dateien mit einem Kopfblock (Frontmatter):

```markdown
---
name: reviewer
description: Wann dieser Agent eingesetzt wird (Claude wählt danach aus).
tools: Read, Grep, Glob, Bash     # Rechte bewusst einschränken
---

Anweisungen für die Rolle …
```

Jeder Subagent arbeitet in einem eigenen Kontext und liefert nur sein Ergebnis zurück.
Über `tools` bekommt jede Rolle nur die Rechte, die sie braucht.

### Warum diese vier Rollen

| Rolle | Darf schreiben | Zweck |
|---|---|---|
| architect | nein | Struktur vor Details: Abhängigkeiten zeigen nach innen |
| test-author | nur Tests | TDD: das gewünschte Verhalten wird zuerst festgehalten |
| programmer | nur Code | Tests grün machen, **ohne** Tests zu verbiegen |
| reviewer | nein | unabhängige Prüfung, v.a. Secrets und Fehlerbehandlung |

Die Trennung von test-author und programmer ist der Kern: Wer den Code schreibt,
darf die Messlatte nicht selbst verschieben. Einen "Manager"-Agenten gibt es bewusst nicht,
die Koordination macht der Mensch.

### Ablauf pro Feature

1. Im Projektordner `claude` starten.
2. Mit **Shift+Tab** in den *Plan Mode* wechseln, damit zuerst ein Plan entsteht.
3. Feature beschreiben, z.B.:
   ```
   Feature: Positionen von BingX importieren.
   Starte mit dem architect für den Schnitt, dann test-author,
   dann programmer, am Schluss reviewer.
   ```
4. Plan prüfen und freigeben.
5. Nach dem Review: `npm run lint && npm test`, dann committen lassen.

### Nützliche Befehle

| Was | Wie |
|---|---|
| Definierte Agenten anzeigen | Claude fragen: "Welche Subagents sind definiert?" |
| Laufende Agenten / Protokolle | `/tasks` |
| Agent-Dateien prüfen | `claude plugin validate .claude/agents` |
| Session als bestimmte Rolle starten | `claude --agent reviewer` |
| Agent anlegen oder ändern | Datei in `.claude/agents/` bearbeiten (wirkt ohne Neustart) |

> Hinweis: Der frühere `/agents`-Wizard wurde in neueren Claude-Code-Versionen entfernt.
> Agenten werden direkt über die Dateien verwaltet.

### Übertragbar auf andere Projekte und Tools

Übertragbar sind die **Prinzipien**:
- ein zentrales Projektgedächtnis mit Architektur, Regeln und bekannten Problemen,
- getrennte Rollen mit minimalen Rechten,
- Tests als Vertrag, den der Programmierer nicht ändern darf,
- unabhängiges Review vor jedem Commit,
- der Mensch koordiniert.

Claude-Code-spezifisch sind die **Dateinamen und Formate** (`CLAUDE.md`, `.claude/agents/`).
Andere AI-Coding-Tools haben eigene Dateien für Projektregeln und Rollen;
der Inhalt lässt sich weitgehend übernehmen.

---

## Sicherheit

- BingX-API-Key **nur mit Lese-Rechten**.
- `BX_KEY` / `BX_SECRET` nur in `.env` (nicht im Repo).
