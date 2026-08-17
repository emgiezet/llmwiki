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

```bash
llmwiki setup                         # interactive global config wizard (~/.llmwiki/config.yaml)
llmwiki init                          # interactive per-project wizard (no flags + TTY)
```

## Architecture

The binary is a single cobra CLI. `main.go` wires the subcommands from `internal/cmd/`. The core data flow is:

**ingest:** `scanner` (+ `extractor`) → `ingestion` → `llm` → `wiki`

1. `internal/scanner` — walks a project directory and collects relevant files (README, go.mod, docker-compose, .proto, etc.) into a single text summary. `DetectServices` auto-detects multi-service layouts from docker-compose.yml first, then subdirectory heuristics. `ScanProject(ctx, dir, opts...)` takes a `WithExtractor` option; when set, binary document files (`.pdf`/`.docx`/`.odt`/`.epub`) are converted to text and folded into the summary (capped at 50 files / ~50 KB each).
   - `internal/extractor` — shells out to configurable external converters (`pandoc`, `pdftotext`) to turn document files into text. `CommandExtractor` maps an extension to a command template (`{{input}}` placeholder, run without a shell), mirroring the subprocess pattern in `llm/cli_backend.go`. A missing tool yields `ErrToolNotFound` and the scanner skips the file rather than failing.
2. `internal/ingestion` — orchestrates the pipeline: scan → prompt → LLM call → write. `IngestProject` branches on whether services were detected: zero services = single project file, one+ services = one file per service.
3. `internal/llm` — three backends behind the `LLM` interface (`Generate(ctx, prompt) (string, error)`): `claude-code` (shells to `claude -p`), `claude-api` (Anthropic SDK), `ollama` (REST). `NewFakeLLM` is the test double used throughout.
4. `internal/wiki` — reads/writes markdown files with YAML front matter. `WriteProjectEntry`/`WriteServiceEntry` own the file format. `UpsertIndex` maintains the master `_index.md`. `query.go` adds a read-only, LLM-free query layer: `Store` (rooted at the wiki dir) with `Search(client, project)` (filters `_index.md`; client = exact case-insensitive, project = substring), `GetProject(client, project, service)`, and `ListServices`. It composes the existing `ReadIndex`/`Parse*Entry`/`ExtractSection` helpers and backs the MCP server.
   - `knowledge.go` — the knowledge-layer axis (see Wiki Storage Layout). `SearchKnowledge(layers, query)` (empty query = list), `GetKnowledge(layer, topic)`, `ListKnowledgeLayers()`. Filesystem-walked rather than index-driven, so front matter is optional and a layer can be a git submodule. Layer names are validated with `validation.NameComponent` because they're joined into paths; `safeio.ReadRegularFile` refuses symlinks; `.git` dirs are skipped.
5. `internal/config` — three-level config: global (`~/.llmwiki/config.yaml`), per-client (`~/.llmwiki/clients/<customer>.yaml`), and per-project (`llmwiki.yaml`). `Merge` resolves them project > client > global. Slice-valued fields (`Extraction.Sections`, `Knowledge`) replace rather than append.
6. `internal/tracker` — change tracking via git history. `cochange.go` clusters files that change together (union-find, 30% co-occurrence threshold) into `Area`s; `area.go` computes a content-addressed hash from `git ls-tree HEAD` output; `freshness.go` compares a stored hash against the current one. `GitRunner` is the injectable git-subprocess interface (real impl shells to `git`, `fakeGitRunner` in tests). At ingest time `ingestion.buildTracking` writes the resulting `wiki.TrackingMeta` into entry front matter; `cmd/check.go` re-checks it.

**check:** `wiki` (read entries) → `tracker.CheckFreshness` → fresh/stale report. The `check` command powers three triggers: manual `llmwiki check`, the Git pre-commit hook (`init --hooks`), and the graymatter Stop hook.

**mcp:** `cmd/mcp.go` (`llmwiki mcp`) → `internal/mcpserver` (built on `github.com/modelcontextprotocol/go-sdk`) → `wiki.Store`. Runs a stdio MCP server exposing four tools so agents can read extracted info without an LLM: `search_projects(client?, project?)`, `get_project(project, client?, service?)`, `search_knowledge(query?, layer?)`, and `get_knowledge(topic, layer?)`. When no layer is named the knowledge tools fall back to `ListKnowledgeLayers()`, so `search_knowledge` with no arguments doubles as layer discovery. `mcpserver.New(store)` registers the tools; `mcpserver.Serve(ctx, store)` runs over `StdioTransport`. The handlers are thin wrappers over `wiki.Store` and are unit-tested directly.

**knowledge layers:** `cmd/ingest.go --layer <name> [--topic <name>]` → `ingestion.IngestKnowledge` → `<wiki_root>/knowledge/<layer>/<topic>.md`. Reuses the project pipeline but skips service detection, `UpsertIndex`, and client/multi-project index generation. `cmd/context.go` appends `renderKnowledgeContext` output (per-entry + total char budgets) under `## Knowledge: <layer>/<topic>` headings; `--no-knowledge` opts out. `cmd/query.go` needs no changes — `loadAllWikiContent` already walks the whole wiki root.

## Wiki Storage Layout

Files land at `~/llmwiki/wiki/` (configurable via `wiki_root`):

```
wiki/
├── clients/{customer}/{project}.md           # single-service client project
├── clients/{customer}/{project}/{svc}.md     # multi-service (one file per service)
├── personal/{project}.md
├── opensource/{project}.md
├── knowledge/{layer}/*.md                    # knowledge layers (global, a team, a client)
└── _index.md                                 # YAML front matter listing all projects
```

`ingestion.TypeToDir` handles the type→directory mapping (`client`→`clients`, `personal`→`personal`, `oss`→`opensource`).

`knowledge/` is the second axis: knowledge that belongs to no single repo. It is deliberately **not** indexed in `_index.md` — `wiki.Store` reads it by walking the filesystem, which is what makes hand-written files and git submodules work with no bookkeeping. A layer is just a directory name; `config.Merged.Knowledge` holds the resolved lookup order (most specific first, default `["global"]`).

## Per-project Config

Drop `llmwiki.yaml` in the project root to override LLM backend and set metadata:

```yaml
type: client       # client | personal | oss
customer: acme
llm: ollama
ollama_model: llama3.2
output_mode: both        # central (default) | local | both
local_docs_dir: docs/llmwiki
knowledge: [acme, global]  # knowledge layers, most specific first (default: [global])
```

Absent fields fall back to the global config. The `llm` field accepts `claude-code` (default, uses Claude Code subscription), `claude-api` (requires `ANTHROPIC_API_KEY`), or `ollama`. `output_mode` controls where wiki files are written: `central` (`~/llmwiki/wiki/` only), `local` (`<project>/<local_docs_dir>/` only), or `both`.

For non-technical projects (notes, research, articles), set `extraction.preset` to `notes` or `research` (prose-oriented sections instead of code-oriented), and configure document converters via `extractors` (global config + per-project override). `config.DefaultExtractors()` ships the macOS/Linux defaults (`pdftotext`, `pandoc`); they merge key-by-key (`mergeExtractors` in `merge.go`).

## Interactive Setup

`llmwiki setup` runs an interactive wizard for the global `~/.llmwiki/config.yaml` (LLM backend, wiki root, memory, extractor detection). Running `llmwiki init` with no flags in a terminal launches a per-project wizard (type, customer, extraction preset, output mode, optional hooks). Passing any flag (`--customer`/`--type`/`--hooks`/`--no-graymatter`) or running without a TTY (CI) keeps the original non-interactive behaviour. Both wizards load an existing config as defaults, so they double as editors. The prompt helpers live in `internal/wizard` (a dependency-free `Prompter` with injectable I/O); TTY detection is the package-level `isInteractive` var in `internal/cmd/setup.go`.

## CLAUDE.md Injection

The `context` command is designed to inject wiki content into CLAUDE.md files:

```bash
llmwiki context myproject --inject CLAUDE.md
```

The target file must contain `<!-- llmwiki:start -->` and `<!-- llmwiki:end -->` markers. The command replaces content between them with the project's Domain + Services + Flows sections.
