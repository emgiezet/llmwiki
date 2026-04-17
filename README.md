# llmwiki

A CLI that scans your project directories and generates a persistent, LLM-maintained markdown knowledge base. Inspired by [Karpathy's LLM Wiki pattern](https://x.com/karpathy/status/1908184210424959371) — plain markdown files that compound over time, browsable in any editor, injectable into AI coding sessions.

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
```

## Commands

### `ingest <path>`

Scans a project directory, sends the collected files to an LLM, and writes structured wiki entries.

- Auto-detects multi-service projects from `docker-compose.yml` or subdirectories containing code
- Single-service projects get one markdown file; multi-service projects get one file per service
- Re-running updates existing entries — the LLM sees the previous wiki content and refines it

```bash
llmwiki ingest ~/workspace/my-api
llmwiki ingest ~/workspace/my-api --service api-gateway  # refresh one service only
```

### `list`

Lists all tracked projects with their customer, type, status, and wiki path.

```
PROJECT   CUSTOMER  TYPE      STATUS  WIKI
-------   --------  ----      ------  ----
my-api    acme      client    active  clients/acme/my-api.md
llmwiki             personal  active  personal/llmwiki.md
```

### `context <project>`

Prints the Domain, Services, and Flows sections to stdout — trimmed for token efficiency. Designed for piping into system prompts or `CLAUDE.md` files.

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
```

## Wiki Structure

Wiki entries are plain markdown with YAML front matter, stored at `~/llmwiki/wiki/` by default:

```
wiki/
├── clients/
│   └── acme/
│       └── billing-api.md
├── personal/
│   └── llmwiki.md
├── opensource/
│   └── some-lib.md
└── _index.md
```

Multi-service projects get a directory with one file per service:

```
wiki/clients/insly/mmx3/
├── _index.md
├── api-gateway.md
├── policy-service.md
└── worker.md
```

Each wiki entry contains LLM-generated sections:

**Project files:** Domain, Services, Flows, Integrations, Tech Stack, Notes

**Service files:** Purpose, API Surface, Integrations, Notes

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

The wiki context (Domain, Services, Flows) is injected between the markers, giving Claude Code immediate understanding of your project's architecture.

## License

MIT
