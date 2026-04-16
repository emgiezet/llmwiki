# LLM Wiki — Design Spec

**Date:** 2026-04-16
**Status:** Draft

## Overview

A persistent, compounding knowledge base for all active projects, following Karpathy's LLM Wiki pattern. The wiki is plain markdown files — browsable in any editor, injectable into Claude Code sessions, and maintained by LLMs (Claude or local Ollama depending on project config).

The system is built as a Go CLI first, with a thin MCP server layer added later to enable native Claude Code tool calls.

---

## Architecture

Four layers:

1. **Data sources** — local project directories. Scanner reads: `README*`, `*.md`, `go.mod` / `package.json` / `Cargo.toml`, `docker-compose*`, `*.proto`, `.env.example`, top-level directory names.
2. **CLI** — `llmwiki` binary with commands: `ingest`, `query`, `context`, `list`, `serve`.
3. **LLM router** — routes to Claude API (default) or local Ollama based on per-project config. Routing decisions are driven by: data privacy/NDA requirements and cost control.
4. **Wiki storage** — plain markdown files in `~/llmwiki/wiki/` (configurable), organized by project type and customer.

---

## Wiki Storage Structure

```
wiki/
├── clients/
│   ├── insly/
│   │   ├── mmx3/
│   │   │   ├── _index.md          ← project overview + service list
│   │   │   ├── api-gateway.md
│   │   │   ├── policy-service.md
│   │   │   └── document-service.md
│   │   └── other-project.md       ← single-file for simple projects
│   └── acme/
│       └── billing-api.md
├── personal/
│   └── llmwiki.md
├── opensource/
│   └── some-lib.md
└── _index.md                      ← master listing of all projects
```

**Rule:** Single-service or simple projects use one `.md` file. Multi-service projects use a directory with `_index.md` + one file per service. Service files are auto-detected from subdirectories, `docker-compose.yml` services, or Go workspace modules.

---

## Wiki Entry Format

### Project-level `_index.md` front matter

```yaml
---
name: mmx3
customer: insly
type: client          # client | personal | oss
status: active        # active | paused | archived
path: /home/mgz/workspace/mmx3
llm: claude           # claude | ollama
ollama_model: null    # e.g. llama3.2 when llm: ollama
tags: [go, grpc, insurance]
last_ingested: 2026-04-16T10:00:00Z
---
```

### Service-level file front matter

```yaml
---
service: api-gateway
project: mmx3
customer: insly
language: go
path: ./services/api-gateway
exposes: [REST :8080, gRPC :9090]
depends_on: [policy-service, document-service]
last_ingested: 2026-04-16T10:00:00Z
---
```

### LLM-written sections

**Project `_index.md`:**
- `## Domain` — what the project is about, business context
- `## Services` — list of services with one-line descriptions
- `## Flows` — key end-to-end flows between services
- `## Integrations` — external systems (APIs, queues, databases)
- `## Tech Stack` — languages, frameworks, infra
- `## Notes` — gotchas, architectural decisions, known issues

**Per-service file:**
- `## Purpose` — what this service does and why it exists
- `## API Surface` — endpoints, proto definitions, contracts
- `## Integrations` — what it calls, what calls it
- `## Notes` — gotchas, decisions, known issues

---

## Ingestion Pipeline

When `llmwiki ingest <path> [--service <name>]` runs:

1. **Scan** — walk the project directory, collect relevant files (README, config, proto, docker-compose, top-level dirs)
2. **Detect services** — if no `--service` flag, auto-detect services from subdirectories / `docker-compose.yml` / Go workspace
3. **Load existing wiki entry** — if one exists, pass it to the LLM as context so it updates rather than overwrites
4. **Route LLM** — check `llmwiki.yaml` at project root, then `~/.llmwiki/config.yaml` global config, then default to Claude
5. **Generate/update** — LLM receives scraped content + existing wiki entry + schema prompt instructing it to fill each section
6. **Write** — write/update markdown files, update `last_ingested` in front matter, update `_index.md` master listing

**Per-project config** (`llmwiki.yaml` at project root):

```yaml
llm: ollama
ollama_model: llama3.2
customer: insly
type: client
```

If absent, CLI defaults to Claude API using `ANTHROPIC_API_KEY`.

**Global config** (`~/.llmwiki/config.yaml`):

```yaml
wiki_root: ~/llmwiki/wiki    # where wiki files are stored
llm: claude                   # default LLM for all projects
ollama_host: http://localhost:11434
anthropic_api_key: ""         # falls back to ANTHROPIC_API_KEY env var
```

Per-project `llmwiki.yaml` overrides any field from the global config.

---

## CLI Commands

```
llmwiki ingest <path> [--service <name>]
```
Scan project directory and generate/update wiki entries. Without `--service`, ingests all detected services. With `--service`, refreshes only that service file.

```
llmwiki query "<question>"
```
Ask a natural language question across all wiki entries. LLM synthesizes answer from matching wiki content. Uses Claude by default (no per-project routing — this is a meta-operation).

```
llmwiki context <project> [--service <name>]
```
Print wiki markdown for a project or service to stdout. Intended for piping into `CLAUDE.md` or system prompts. Output is trimmed to Domain + Services + Flows for token efficiency.

```
llmwiki list
```
List all tracked projects with status, type, customer, and last ingested timestamp.

```
llmwiki serve
```
*(Future)* Start an MCP server exposing `get_project_context`, `ingest_project`, `query_wiki` as Claude tools.

---

## LLM Routing

Three backends, two routing axes:

| Mode | Config value | How it works |
|------|-------------|-------------|
| Claude Code CLI | `llm: claude-code` | Shells out to `claude -p`, uses Claude Code subscription |
| Claude API | `llm: claude-api` | Anthropic Go SDK + `ANTHROPIC_API_KEY`, faster for bulk |
| Ollama | `llm: ollama` | Local REST API, for NDA/private code or cost control |

**Note:** Using Claude subscription OAuth tokens directly in third-party tools violates Anthropic ToS (changed April 4, 2026). The `claude-code` mode is safe — it delegates to the official Claude Code CLI binary.

Two routing axes per project:

| Axis | claude-code / claude-api | ollama |
|------|--------------------------|--------|
| Privacy | Public / internal projects | NDA / sensitive client code |
| Cost | Default | Low-priority or bulk operations |

**Global default: `claude-code`** (uses existing Claude Code install, no extra billing). Per-project override via `llm:` field in `llmwiki.yaml`. When `ollama` is selected, `ollama_model` specifies which model (e.g., `llama3.2`).

---

## Claude Code Integration

**Today (without MCP):**

Add to a project's `CLAUDE.md`:

```bash
<!-- auto-generated: llmwiki context mmx3 -->
<wiki content here>
```

Run `llmwiki context mmx3 --inject CLAUDE.md` after each ingest to refresh. The `--inject` flag replaces the content between `<!-- llmwiki:start -->` and `<!-- llmwiki:end -->` markers in the target file. The `context` command outputs only Domain + Services + Flows sections.

**Future (MCP via `llmwiki serve`):**

MCP server exposes three tools:
- `get_project_context(project, service?)` — returns wiki content for injection
- `ingest_project(path, service?)` — trigger ingestion mid-session
- `query_wiki(question)` — cross-project natural language query

The MCP server is a thin HTTP wrapper around the same CLI logic — no new business logic.

---

## Out of Scope (v1)

- Git/GitHub integration (branches, PRs, history)
- External tool connectors (Jira, Linear, Confluence)
- Vector embeddings / semantic search
- Web UI
- Automated scheduling / file-watching daemon

---

## Tech Stack

- **Language:** Go
- **LLM SDKs:** Anthropic Go SDK (Claude), Ollama REST API
- **LLM backends:** Claude Code CLI (`claude -p`), Anthropic Go SDK (API key), Ollama REST API
- **Config format:** YAML (`llmwiki.yaml` per project, `~/.llmwiki/config.yaml` global)
- **Wiki format:** Markdown with YAML front matter
- **Binary distribution:** Single compiled binary, installable via `go install`
