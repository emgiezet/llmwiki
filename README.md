# llmwiki

A CLI that scans your project directories and generates a persistent, LLM-maintained markdown knowledge base. Inspired by [Karpathy's LLM Wiki pattern](https://x.com/karpathy/status/1908184210424959371) — plain markdown files that compound over time, browsable in any editor, injectable into AI coding sessions.

The wiki is fully compatible with [Obsidian](https://obsidian.md/) — point it at your `~/llmwiki/wiki/` directory and you get a navigable vault with rendered mermaid diagrams, cross-linked files, and YAML front matter properties out of the box.

## Features

- **Per-service wiki files** — multi-service projects get one detailed file per service, auto-detected from docker-compose, subdirectories, or code indicators (Go, PHP, Python, Rust, Java, Ruby)
- **Mermaid diagrams** — system architecture flowcharts and entity-relationship diagrams generated in every wiki entry, rendered natively in GitHub, GitLab, and Obsidian
- **Auto-generated tags** — technologies and architectural patterns (`go, grpc, event-driven, microservices, ...`) extracted into YAML front matter, searchable in Obsidian
- **API documentation** — swagger/openapi specs are scanned and endpoints listed as markdown tables in the API Surface section
- **Cross-file links** — wiki files that mention other tracked projects or services get automatic `[name](path.md)` links, creating a navigable knowledge graph
- **Client indexes** — per-client `_index.md` with executive summary, C4 system landscape diagram, architecture overview, and a projects table with links
- **Project indexes** — multi-service projects get a synthesized `_index.md` with domain overview, service table, and system diagram
- **CLAUDE.md injection** — pipe wiki context directly into `CLAUDE.md` files for AI coding sessions, with automatic marker-based replacement
- **Three LLM backends** — Claude Code CLI (default, uses subscription), Anthropic API, or local Ollama for private/NDA code
- **Incremental updates** — re-running ingest on a project feeds the existing wiki entry to the LLM, so content refines over time rather than regenerating from scratch

## Install

```bash
go install github.com/mgz/llmwiki@latest
```

Or build from source:

```bash
git clone https://github.com/mgz/llmwiki.git
cd llmwiki
go build -o llmwiki .
```

## Quick Start

```bash
# Ingest a project (uses Claude Code CLI by default)
llmwiki ingest ~/workspace/my-project

# See what's tracked
llmwiki list

# Get context for an AI coding session
llmwiki context my-project

# Inject into a CLAUDE.md
llmwiki context my-project --inject CLAUDE.md

# Ask a question across all projects
llmwiki query "which projects use gRPC?"

# Generate client-level executive summary
llmwiki index acme
```

## Commands

### `ingest <path>`

Scans a project directory, sends the collected files to an LLM, and writes structured wiki entries.

- Auto-detects multi-service projects from `docker-compose.yml` or subdirectories containing code indicators (`go.mod`, `package.json`, `composer.json`, `Dockerfile`, `src/`, etc.)
- Single-service projects get one markdown file; multi-service projects get one file per service plus a project `_index.md`
- Generates mermaid system diagrams, ERD diagrams, and auto-tags
- Re-running updates existing entries — the LLM sees the previous wiki content and refines it
- Automatically cross-links wiki files and regenerates the client index

```bash
llmwiki ingest ~/workspace/my-api
llmwiki ingest ~/workspace/my-api --service api-gateway  # refresh one service only
```

### `list`

Lists all tracked projects with their customer, type, status, and wiki path.

```
PROJECT            CUSTOMER  TYPE      STATUS  WIKI
-------            --------  ----      ------  ----
billing-api        acme      client    active  clients/acme/billing-api.md
ecommerce          acme      client    active  clients/acme/ecommerce/_index.md
my-tool                      personal  active  personal/my-tool.md
```

### `context <project>`

Prints key wiki sections to stdout — trimmed for token efficiency. Designed for piping into system prompts or `CLAUDE.md` files. Excludes diagrams and configuration to keep output concise.

```bash
# Print to stdout
llmwiki context my-api

# Inject between markers in a file
llmwiki context my-api --inject CLAUDE.md

# Get a specific service
llmwiki context my-api --service api-gateway
```

The `--inject` flag replaces content between `<!-- llmwiki:start -->` and `<!-- llmwiki:end -->` markers in the target file.

### `query "<question>"`

Asks a natural language question across all wiki entries. The LLM receives the full wiki content as context and synthesizes an answer.

```bash
llmwiki query "what databases does the billing service use?"
llmwiki query "which services use gRPC?"
```

### `index [customer]`

Generates or regenerates client-level and project-level index files without re-ingesting source code. Useful after ingesting multiple projects over time.

```bash
llmwiki index acme           # regenerate acme client index
llmwiki index                # regenerate all client indexes
```

Client indexes include an executive summary, C4 system landscape diagram, architecture overview, and a projects table with links.

### `link`

Manually triggers cross-file linking across all wiki files. Runs automatically after each ingest, but can be run standalone.

```bash
llmwiki link
```

## Wiki Structure

Wiki entries are plain markdown with YAML front matter, stored at `~/llmwiki/wiki/` by default:

```
wiki/
├── _index.md                              # global project listing
├── clients/
│   ├── acme/
│   │   ├── _index.md                      # client executive summary + C4 diagram
│   │   ├── billing-api.md                 # single-service project
│   │   ├── notification-service.md
│   │   └── ecommerce/
│   │       ├── _index.md                  # project overview + service table
│   │       ├── cart-service.md            # per-service wiki
│   │       ├── payment-service.md
│   │       └── ...
│   └── globex/
│       ├── _index.md                      # client executive summary
│       └── platform/
│           ├── _index.md                  # project overview
│           ├── auth-service.md
│           ├── user-service.md
│           └── ...
├── personal/
│   └── my-tool.md
└── opensource/
    └── some-lib.md
```

### Wiki entry sections

**Project files:** Domain, Architecture, Services, Features, Flows, System Diagram, Data Model Diagram, Integrations, Tech Stack, Configuration, Notes

**Service files:** Purpose, Architecture, API Surface, System Diagram, Data Model, Data Model Diagram, Integrations, Configuration, Notes

**Client index:** Executive Summary, C4 Diagram, Architecture Overview, Projects

**Project index (multi-service):** Domain, Architecture, Services, System Diagram, Integrations, Tech Stack

### YAML front matter

Every wiki file has structured YAML front matter with metadata:

```yaml
---
name: billing-api
customer: acme
type: client
status: active
tags: [go, gin, grpc, kubernetes, rabbitmq, event-driven]
last_ingested: 2025-01-15T10:30:00Z
---
```

Tags are auto-generated by the LLM and include languages, frameworks, infrastructure, and architectural patterns.

## Obsidian Compatibility

![llmwiki in Obsidian](docs/obsidian.png)

![llmwiki graph view in Obsidian](docs/obsidian2.png)

The wiki directory works as an Obsidian vault out of the box:

1. Open Obsidian, choose "Open folder as vault", select `~/llmwiki/wiki/`
2. Mermaid diagrams (system architecture, ERD, C4) render natively in Obsidian's preview
3. Cross-file links (`[service-name](path.md)`) are navigable — click through from client index to project to service
4. YAML front matter shows as properties in Obsidian's properties view
5. Tags in front matter are searchable via Obsidian's tag pane

For the best experience, enable these Obsidian core plugins:
- **Tags** — browse by technology (`#go`, `#grpc`, `#kubernetes`)
- **Graph view** — visualize connections between projects and services
- **Backlinks** — see which files reference a given service

## LLM Backends

Three backends, configured per-project or globally:

| Backend | Config value | How it works |
|---------|-------------|-------------|
| Claude Code CLI | `claude-code` (default) | Shells out to `claude -p`. Uses your Claude Code subscription — no API key needed. |
| Claude API | `claude-api` | Anthropic Go SDK. Requires `ANTHROPIC_API_KEY` env var or config. |
| Ollama | `ollama` | Local REST API. For NDA/private code or cost control. |

## Configuration

### Per-project: `llmwiki.yaml`

Drop this in the project root to set its type, customer, and LLM backend:

```yaml
type: client         # client | personal | oss
customer: acme
llm: ollama
ollama_model: llama3.2
```

### Global: `~/.llmwiki/config.yaml`

```yaml
wiki_root: ~/llmwiki/wiki
llm: claude-code
ollama_host: http://localhost:11434
anthropic_api_key: ""   # or set ANTHROPIC_API_KEY env var
```

Per-project config overrides global. If neither exists, defaults to `claude-code` with wiki at `~/llmwiki/wiki/`.

## Claude Code Integration

Add markers to your project's `CLAUDE.md`:

```markdown
# My Project

<!-- llmwiki:start -->
<!-- llmwiki:end -->
```

Then run after each ingest:

```bash
llmwiki context my-project --inject CLAUDE.md
```

The wiki context (Domain, Architecture, Services, Features, Flows) is injected between the markers, giving Claude Code immediate understanding of your project's architecture.

## License

MIT
