# AI-Setup: Entwickeln mit Claude Code und Subagents

Wie dieses Projekt mit AI-Agenten entwickelt wird. Die Diagramme sind Mermaid
und werden auf GitHub und in den meisten Editoren direkt gerendert.

## 1. Überblick: Wer liest was?

```mermaid
flowchart TB
    Mensch(["Mensch<br/>koordiniert, gibt frei, committet"])
    Claude["Claude Code<br/>Hauptsession im Projektordner"]

    subgraph Gedaechtnis["Projektgedächtnis (wird automatisch geladen)"]
        CLAUDE["CLAUDE.md<br/>Ziel, Architektur, Regeln, Sicherheit"]
    end

    subgraph Wissen["Referenzdokumente"]
        API["docs/openapi.yaml<br/>API-Vertrag"]
        DESIGN["docs/design.md<br/>Design-System"]
        PLAN["docs/migration/go-wails.md<br/>Migrationsplan"]
        FEAT["docs/features/*.md<br/>Feature-Spezifikationen"]
    end

    subgraph Agenten[".claude/agents/"]
        ARC["architect"]
        TST["test-author"]
        PRG["programmer"]
        DSG["designer"]
        REV["reviewer"]
    end

    Mensch -->|Auftrag| Claude
    Claude --> CLAUDE
    Claude -->|delegiert| Agenten
    ARC -.liest.-> API
    ARC -.liest.-> PLAN
    TST -.liest.-> FEAT
    PRG -.liest.-> FEAT
    DSG -.liest.-> DESIGN
    Agenten -->|Ergebnis| Claude
    Claude -->|Plan, Ergebnis| Mensch
```

## 2. Ablauf pro Feature

```mermaid
flowchart TB
    A["Feature-Idee"] --> P{"Plan Mode<br/>Shift+Tab"}
    P -->|Plan freigeben| ARC["architect<br/>Schnitt + API-Vertrag"]
    ARC --> TST["test-author<br/>Tests schreiben (rot)"]
    TST --> PRG["programmer<br/>Tests grün machen"]
    PRG --> UI{"Sichtbare<br/>UI-Änderung?"}
    UI -->|ja| DSG["designer<br/>Screenshots + Tokens"]
    UI -->|nein| REV
    DSG --> REV["reviewer<br/>Qualität + Sicherheit"]
    REV --> OK{"Befunde?"}
    OK -->|ja| PRG
    OK -->|nein| CHK["npm run lint && npm test"]
    CHK --> C["Commit auf feature/*<br/>nur auf Wunsch"]
```

## 3. TDD im Detail: Wer darf was ändern?

```mermaid
sequenceDiagram
    autonumber
    actor M as Mensch
    participant C as Claude Code
    participant A as architect
    participant T as test-author
    participant P as programmer
    participant D as designer
    participant R as reviewer

    M->>C: Feature beschreiben
    C->>A: Schnitt vorschlagen
    A-->>C: Entities, Ports, Use Cases, OpenAPI-Änderung
    C->>M: Plan zur Freigabe
    M->>C: freigegeben
    C->>T: Verhalten als Tests festhalten
    T-->>C: Tests rot ✗
    C->>P: Tests grün machen (Tests nicht ändern!)
    P-->>C: Tests grün ✓
    opt UI betroffen
        C->>D: UI prüfen (Screenshots hell/dunkel, 1440/390 px)
        D-->>C: Befunde / Token-Anpassungen
    end
    C->>R: Review
    R-->>C: Befunde nach Schweregrad
    C->>M: Ergebnis + Vorschlag Commit
```

## 4. Rechte der Rollen

```mermaid
flowchart LR
    subgraph Lesen["Nur lesen"]
        ARC["architect<br/>Read, Grep, Glob"]
        REV["reviewer<br/>Read, Grep, Glob, Bash"]
    end
    subgraph Schreiben["Schreiben, aber begrenzt"]
        TST["test-author<br/>nur *.test.ts"]
        PRG["programmer<br/>nur Produktivcode"]
        DSG["designer<br/>nur clients/ + docs/design.md"]
    end

    TST -. "darf nicht" .-x PRGCODE["Produktivcode"]
    PRG -. "darf nicht" .-x TESTS["Tests"]
    DSG -. "darf nicht" .-x BACK["Backend / Domain"]
```

| Rolle | Darf schreiben | Prüft vor allem |
|---|---|---|
| architect | nein | Abhängigkeiten zeigen nach innen, API passt zur `openapi.yaml` |
| test-author | nur Tests | Verhalten, Randfälle, echte BingX-Fixtures |
| programmer | nur Code | einfachster Code, der die Tests erfüllt |
| designer | nur `clients/` | Tokens, Kontrast, Konsistenz, per Screenshot |
| reviewer | nein | Secrets, Fehlerbehandlung, Lesbarkeit |

## 5. Wo die Agenten im System arbeiten

```mermaid
flowchart TB
    subgraph Clients["clients/ (austauschbar)"]
        WEB["web: React + Vite"]
        DESK["desktop: Wails (geplant)"]
    end
    VERTRAG[["docs/openapi.yaml<br/>HTTP/JSON-Vertrag"]]
    subgraph Backend["backend/ (Go)"]
        PRES["presentation<br/>HTTP-API"]
        APP["application<br/>Use Cases"]
        DOM["domain<br/>Entities, Value Objects"]
        INFRA["infrastructure<br/>BingX-Client, SQLite, Scheduler"]
    end
    BINGX[("BingX API<br/>nur Lese-Key")]
    DB[("SQLite<br/>lokal")]

    WEB --> VERTRAG
    DESK --> VERTRAG
    VERTRAG --> PRES
    PRES --> APP --> DOM
    INFRA --> DOM
    APP --> INFRA
    INFRA --> BINGX
    INFRA --> DB

    DSG(["designer"]) -.-> Clients
    ARC(["architect"]) -.-> VERTRAG
    ARC -.-> Backend
```

## 6. Dateien des Setups

```
CLAUDE.md                     Projektgedächtnis (immer geladen)
.claude/agents/
├── architect.md
├── test-author.md
├── programmer.md
├── designer.md
└── reviewer.md
docs/
├── ai-setup.md               dieses Dokument
├── local-ai-setup.md         Ollama + Qdrant + repo-rag
├── design.md                 Design-System (designer)
├── openapi.yaml              API-Vertrag (architect)
├── features/                 Feature-Spezifikationen
└── migration/                Migrationsplan Go + Wails
```

## 7. Übertragbar auf andere Projekte

Übertragbar sind die **Prinzipien**: ein zentrales Projektgedächtnis, getrennte Rollen mit
minimalen Rechten, Tests als Vertrag, den der Programmierer nicht ändern darf, ein unabhängiges
Review, und der Mensch koordiniert.

Claude-Code-spezifisch sind nur die Dateinamen und Formate (`CLAUDE.md`, `.claude/agents/`).

## 8. Lokales Setup zum Token-Sparen

Ollama, Qdrant, `repo-rag` und `ask-local` entlasten Claude bei Suche und Fleissarbeit.
Architektur-Grafik, Installation und TODOs: [local-ai-setup.md](local-ai-setup.md).
