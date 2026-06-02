# How It Works

The scan → detect → generate → write pipeline, and how graymatter memory layers on top.

[← Back to README](../README.md)

## Core Pipeline

```mermaid
flowchart LR
    subgraph Scan
        A[Project Directory] --> B[Scanner]
        B --> |README, go.mod,\nDockerfile, .proto,\nswagger specs...|C[Scan Summary]
    end

    subgraph Detect
        A --> D{docker-compose\nor service dirs?}
        D --> |single service|E[One wiki file]
        D --> |multi-service|F[One file per service\n+ project index]
    end

    subgraph Generate
        C --> G[Build Prompt]
        G --> H[LLM]
        H --> I[Structured Markdown\n+ YAML Front Matter]
    end

    subgraph Write
        I --> J[Wiki Files]
        J --> K[Cross-Link]
        K --> L[Update Index]
    end

    subgraph Use
        L --> M[llmwiki context\n→ inject into CLAUDE.md]
        L --> N[llmwiki query\n→ ask across projects]
        L --> O[Browse in\nObsidian / editor]
    end
```

## With Graymatter Memory

When `memory_enabled: true`, [graymatter](https://github.com/angelnicolasc/graymatter) adds a persistent memory layer. Knowledge compounds across ingestion runs — the LLM sees what it learned before, cross-project patterns surface automatically, and the `query` command gets semantic search instead of brute-force file walking.

```mermaid
flowchart LR
    subgraph Ingest ["llmwiki ingest"]
        A[Scanner] --> B[Scan Summary]
        B --> C[Build Prompt]
        GM[(graymatter\nmemory)] --> |"Recall:\nprevious facts,\ncross-project\npatterns"|C
        C --> D[LLM]
        D --> E[Wiki Files]
        E --> |"Remember:\nextract & store\natomic facts"|GM
    end

    subgraph Context ["llmwiki context"]
        F[Wiki Sections] --> G[Output]
        GM --> |"Recall:\ncross-project\nknowledge"|G
        G --> H[Inject into\nCLAUDE.md]
    end

    subgraph Query ["llmwiki query"]
        I[Wiki Walk] --> J[Combined\nContext]
        GM --> |"Recall:\nsemantic search\nacross all facts"|J
        J --> K[LLM]
        K --> L[Answer]
    end

    subgraph Docs ["llmwiki docs"]
        M[Scan + Wiki\n+ Existing Doc] --> N[Build Prompt]
        GM --> |"Recall:\nfull project\nhistory"|N
        N --> O[LLM]
        O --> P[Updated\nREADME.md]
    end
```

**Memory stores facts at two levels:**
- **Per-project** (`llmwiki/project/{name}`) — architecture, integrations, tech stack, service topology
- **Per-customer** (`llmwiki/customer/{name}`) — shared infrastructure, cross-project patterns, technology standards

Facts decay over time (30-day half-life) and consolidate in the background. Embedding search uses whatever's available: Ollama → OpenAI → Anthropic → keyword-only fallback.
