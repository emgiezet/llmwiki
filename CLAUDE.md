# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...                        # build everything
go build -o llmwiki .                 # build the binary
go test ./...                         # run all tests
go test ./internal/ingestion/... -v   # run a specific package
go test ./... -run TestFoo -v         # run a single test by name
go vet ./...                          # static analysis
```

## Architecture

The binary is a single cobra CLI. `main.go` wires the subcommands from `internal/cmd/`. The core data flow is:

**ingest:** `scanner` → `ingestion` → `llm` → `wiki`

1. `internal/scanner` — walks a project directory and collects relevant files (README, go.mod, docker-compose, .proto, etc.) into a single text summary. `DetectServices` auto-detects multi-service layouts from docker-compose.yml first, then subdirectory heuristics.
2. `internal/ingestion` — orchestrates the pipeline: scan → prompt → LLM call → write. `IngestProject` branches on whether services were detected: zero services = single project file, one+ services = one file per service.
3. `internal/llm` — three backends behind the `LLM` interface (`Generate(ctx, prompt) (string, error)`): `claude-code` (shells to `claude -p`), `claude-api` (Anthropic SDK), `ollama` (REST). `NewFakeLLM` is the test double used throughout.
4. `internal/wiki` — reads/writes markdown files with YAML front matter. `WriteProjectEntry`/`WriteServiceEntry` own the file format. `UpsertIndex` maintains the master `_index.md`.
5. `internal/config` — two-level config: global (`~/.llmwiki/config.yaml`) and per-project (`llmwiki.yaml` at project root). `Merge` applies project overrides on top of global defaults.
6. `internal/tracker` — change tracking via git history. `cochange.go` clusters files that change together (union-find, 30% co-occurrence threshold) into `Area`s; `area.go` computes a content-addressed hash from `git ls-tree HEAD` output; `freshness.go` compares a stored hash against the current one. `GitRunner` is the injectable git-subprocess interface (real impl shells to `git`, `fakeGitRunner` in tests). At ingest time `ingestion.buildTracking` writes the resulting `wiki.TrackingMeta` into entry front matter; `cmd/check.go` re-checks it.

**check:** `wiki` (read entries) → `tracker.CheckFreshness` → fresh/stale report. The `check` command powers three triggers: manual `llmwiki check`, the Git pre-commit hook (`init --hooks`), and the graymatter Stop hook.

## Wiki Storage Layout

Files land at `~/llmwiki/wiki/` (configurable via `wiki_root`):

```
wiki/
├── clients/{customer}/{project}.md           # single-service client project
├── clients/{customer}/{project}/{svc}.md     # multi-service (one file per service)
├── personal/{project}.md
├── opensource/{project}.md
└── _index.md                                 # YAML front matter listing all projects
```

`ingestion.TypeToDir` handles the type→directory mapping (`client`→`clients`, `personal`→`personal`, `oss`→`opensource`).

## Per-project Config

Drop `llmwiki.yaml` in the project root to override LLM backend and set metadata:

```yaml
type: client       # client | personal | oss
customer: acme
llm: ollama
ollama_model: llama3.2
output_mode: both        # central (default) | local | both
local_docs_dir: docs/llmwiki
```

Absent fields fall back to the global config. The `llm` field accepts `claude-code` (default, uses Claude Code subscription), `claude-api` (requires `ANTHROPIC_API_KEY`), or `ollama`. `output_mode` controls where wiki files are written: `central` (`~/llmwiki/wiki/` only), `local` (`<project>/<local_docs_dir>/` only), or `both`.

## CLAUDE.md Injection

The `context` command is designed to inject wiki content into CLAUDE.md files:

```bash
llmwiki context myproject --inject CLAUDE.md
```

The target file must contain `<!-- llmwiki:start -->` and `<!-- llmwiki:end -->` markers. The command replaces content between them with the project's Domain + Services + Flows sections.
