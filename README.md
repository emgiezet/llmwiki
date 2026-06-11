# llmwiki

> Turn any codebase into a living, LLM-maintained wiki — with architecture diagrams, cross-linked services, and automatic staleness detection.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/emgiezet/llmwiki)](https://github.com/emgiezet/llmwiki/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/emgiezet/llmwiki)](go.mod)
[![CI](https://github.com/emgiezet/llmwiki/actions/workflows/ci.yml/badge.svg)](https://github.com/emgiezet/llmwiki/actions/workflows/ci.yml)
[![Container](https://img.shields.io/badge/ghcr.io-emgiezet%2Fllmwiki-blue?logo=docker)](https://github.com/emgiezet/llmwiki/pkgs/container/llmwiki)

**You can't keep 30 projects in your head. Neither can your AI coding assistant.**

Docs rot the moment you write them. `llmwiki` scans a project and generates a persistent, LLM-maintained markdown wiki — architecture diagrams, service maps, integration maps, cross-links — then flags entries when the code drifts away from the docs. Plain markdown, no database, no SaaS. Inspired by [Karpathy's LLM Wiki pattern](https://x.com/karpathy/status/1908184210424959371).

**Your AI agents read it directly over [MCP](#features)** — `llmwiki mcp` serves every project's extracted domain, services, and flows to Claude Code, Cursor, and any MCP client, with no LLM call and no re-explaining the repo each session.

![llmwiki in Obsidian](docs/obsidian.png)

## Quick start

```bash
# 1. Install (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh | sh

# 2. Configure once (interactive)
llmwiki setup

# 3. Generate a wiki for any project
llmwiki ingest ~/workspace/my-api
```

`my-api` now has a structured wiki entry with diagrams and cross-links.

Prefer a package manager or container? `npx llmwiki ingest .` · `brew install emgiezet/tap/llmwiki` · `docker run --rm ghcr.io/emgiezet/llmwiki:latest version` · `go install github.com/emgiezet/llmwiki@latest`. All install options (pinned versions, pre-built binaries, Docker usage) → [docs/installation.md](docs/installation.md).

## What you get

One `ingest` turns a repo into a structured markdown file — domain & architecture, a service map, API docs, an integration map, auto-generated tags, and **Mermaid diagrams that render right here on GitHub**:

```mermaid
flowchart LR
    GW[API Gateway] --> BILL[billing-api]
    BILL --> DB[(Postgres)]
    BILL --> Q[[Kafka]]
```

Multi-client setups also get executive summaries with C4 diagrams, and every file is cross-linked into a navigable knowledge graph. See how the pipeline works → [docs/architecture.md](docs/architecture.md). Want to see real output first? [`examples/llmwiki.md`](examples/llmwiki.md) is what `ingest` produced on this very repo.

> **Under NDA?** Keep your default backend on the Claude Code subscription and override just the secret project to a local **Ollama** model — that project's code never leaves your machine. No cloud calls, no NDA risk. → [NDA / local-LLM recipe](docs/configuration.md#nda-projects-local-llm-override)

## Features

- **MCP server for agents** — agents query the extracted wiki over stdio (`llmwiki mcp`), filtered by client/project, with no LLM call.
- **AI-coding integration** — inject Domain/Architecture/Services/Flows straight into `CLAUDE.md`.
- **Automatic service detection** — reads `docker-compose.yml` + code indicators, one wiki file per service.
- **Mermaid diagrams** — architecture flowcharts, ERDs, and C4 landscapes; render in GitHub/GitLab/Obsidian.
- **Cross-file linking** — service mentions become clickable links across the knowledge graph.
- **Knowledge layers** — company, department, and client knowledge that belongs to no single repo, consulted in priority order and attributed by layer. Share a layer with your team as a git submodule. → [Knowledge layers](docs/configuration.md#knowledge-layers)
- **Incremental refinement** — re-running `ingest` refines the previous entry instead of starting over.
- **Change tracking & freshness** — knows which source files each entry describes and flags drift (`llmwiki check`).
- **Docs alongside code** — write wikis into the repo (`output_mode: local|both`) so one PR shows code + doc.
- **Three LLM backends** — Claude Code subscription, Claude API, or local Ollama.
- **Sovereign / local-first** — run fully offline on a local Ollama model; code never leaves the box.
- **Not just code** — build wikis from notes, research, and articles via document extraction (PDF/DOCX/ODT/EPUB).
- **Client & project indexes** — executive summaries across all of a client's projects.

## How it compares

Most "AI docs" tools either host your code on a server or generate a static site you still have to keep current by hand. `llmwiki` is local-first markdown that an LLM keeps in sync and that your agents can query directly.

| | llmwiki | Hosted code-wiki (e.g. DeepWiki) | Static-site generator (mkdocs / Docusaurus) | Hand-written `CLAUDE.md` |
|---|:---:|:---:|:---:|:---:|
| Storage | Plain local markdown, no SaaS | Vendor-hosted | Local, but you write it | Local, you write it |
| Keeps itself current | LLM re-ingest + drift detection | Re-index on push | Manual | Manual |
| Drift / staleness flags | ✅ (`check`, git co-change) | — | — | — |
| Multi-project / multi-client | ✅ index + C4 diagrams | per-repo | per-repo | per-repo |
| Agent access without an LLM call | ✅ MCP server | varies | — | reads the file |
| Runs fully offline (local LLM) | ✅ Ollama, NDA-safe | ❌ | n/a | n/a |
| Cost | Your own LLM / subscription | Subscription | Free | Free |

Different problem from context-compression layers (e.g. [headroom](https://github.com/chopratejas/headroom)): those shrink tokens *in flight*; llmwiki gives you and your agents **persistent, self-maintaining knowledge** of every repo. They compose well.

## Documentation

| Guide | What's inside |
|-------|---------------|
| [Installation](docs/installation.md) | Download, one-liner installer, Go install, updating, releases |
| [Configuration](docs/configuration.md) | Global / client / project config, presets, non-code projects & document extraction, NDA local-LLM recipe |
| [Commands](docs/commands.md) | Full command reference, wiki layout, freshness tracking, CLAUDE.md injection |
| [Memory](docs/memory.md) | graymatter persistent memory, modes, seeding, absorb queue |
| [Integrations](docs/integrations.md) | Supported AI tools & session hooks, Obsidian, NanoClaw |
| [Architecture](docs/architecture.md) | How the scan → generate → write pipeline works |

## Who it's for

Consultants juggling many client codebases, tech leads who need docs that match the code, and developers tired of re-explaining project structure to their AI assistant every session.

## Security

Path-traversal rejection, scrubbed LLM prompt/response pipeline, loopback-only Ollama default, bounded subprocess/HTTP deadlines. See [SECURITY.md](SECURITY.md) and the [threat model](docs/threat-model.md); the CI gate lives in [.github/workflows/security.yml](.github/workflows/security.yml).

## License

MIT
