# Commands & Wiki Structure

Full command reference, plus where files land and how freshness tracking works.

[← Back to README](../README.md)

## Commands

| Command | Description |
|---------|-------------|
| `init [path]` | Create `llmwiki.yaml` and optionally wire up graymatter hooks |
| `init [path] --hooks` | Also install a Git pre-commit hook that checks for stale docs |
| `ingest <path>` | Scan a project and generate/update wiki entries |
| `check [path]` | Report which wiki entries are stale relative to source code |
| `check --json / --exit-code / --files` | Machine output / CI exit code / restrict to given files |
| `ingest <path> --no-memory` | Ingest without memory recall/storage |
| `absorb <path>` | Extract session facts into memory (near-zero token cost) |
| `absorb <path> --note "..."` | Absorb with an explicit session note |
| `absorb <path> --note-stdin` | Absorb note piped from stdin (used by the Claude Code hook) |
| `absorb-drain` | Drain queued absorb sessions (created when the memory DB was busy) |
| `materialize <project>` | Rebuild wiki from accumulated memory facts (~10× cheaper than ingest) |
| `list` | List all tracked projects |
| `context <project>` | Print wiki context (pipe into CLAUDE.md) |
| `query "<question>"` | Ask a question across all wiki entries (LLM-backed) |
| `mcp` | Run an MCP server exposing the wiki to agents over stdio (see [Integrations](integrations.md#mcp-server)) |
| `docs <path>` | Generate/update project documentation from wiki + memory |
| `docs <path> --write` | Write the updated doc to the project directory |
| `docs <path> --target FILE` | Update a specific file (default: README.md) |
| `index [customer]` | Generate client and project index files |
| `link` | Add cross-reference links between wiki files |
| `remember --project <name> "<fact>"` | Store a fact in memory |
| `recall "<query>"` | Recall facts from memory |
| `recall --project <name> "<query>"` | Recall facts for a specific project |
| `hook install` | Install llmwiki as a Claude Code plugin |
| `hook uninstall` | Remove the Claude Code plugin |
| `hook status` | Check if the Claude Code plugin is installed |

## Wiki Structure

```
~/llmwiki/wiki/
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
│           ├── _index.md
│           ├── auth-service.md
│           └── ...
├── personal/
│   └── my-tool.md
└── opensource/
    └── some-lib.md
```

Plain markdown with YAML front matter. No proprietary format. Works with git, grep, and any text editor.

## AI coding integration

Inject wiki context directly into `CLAUDE.md` (or any file) with marker-based replacement:

```markdown
<!-- llmwiki:start -->
<!-- llmwiki:end -->
```

```bash
llmwiki context my-project --inject CLAUDE.md
```

Your AI assistant starts every session with Domain, Architecture, Services, and Flows already in context. No more "can you look at the codebase and figure out what this does."

## Change tracking & freshness

Documentation drifts the moment code changes. `llmwiki` tracks which source files each wiki entry describes, and tells you when those files have changed but the docs haven't.

At ingest time, each wiki entry's front matter gains an `llmwiki_tracking` block:

```yaml
llmwiki_tracking:
  area: internal/auth
  files:
    - internal/auth/handler.go
    - internal/auth/middleware.go
  hash: a3f9bc1d2e4f6a8b
  cluster_method: git-cochange
  updated_at: "2026-05-25"
```

llmwiki figures out areas from git co-change history: files that keep landing in the same commits get grouped together (union-find clustering, 30% co-occurrence threshold). On projects with fewer than 20 commits it falls back to top-level directory heuristics. The `hash` is a SHA256 over `git ls-tree HEAD` output for each tracked file, so it changes when file contents change. Timestamps don't enter into it.

Run the check anytime:

```bash
llmwiki check                       # report fresh/stale entries for the current project
llmwiki check --json                # machine-readable output
llmwiki check --exit-code           # exit 1 if anything is stale (for CI)
llmwiki check --files a.go,b.go     # restrict to areas containing these files
```

```
✓ clients/acme/billing-api.md   fresh   area: internal/auth   updated: 2026-05-25
✗ clients/acme/billing-api.md   STALE   area: internal/billing
```

Staleness shows up through three paths:

- **Manual / agent** — run `llmwiki check` yourself, or register it as a slash command so an AI agent runs it before handing work back.
- **Git pre-commit hook** — `llmwiki init --hooks` installs a `.git/hooks/pre-commit` that blocks the commit (`--exit-code`) when staged files belong to a stale area. If the doc update is deliberately deferred, `git commit --no-verify` gets you through. An existing pre-commit hook is left alone.
- **AI session Stop hook** — the graymatter Stop hook (see below) runs a non-blocking `llmwiki check` on the files touched during the session and records the result in memory.

## Client & project indexes

For consultants and agencies managing multiple clients:

```bash
llmwiki index acme    # executive summary across all acme projects
```

Generates a client-level `_index.md` with executive summary, C4 diagram, architecture overview, and a projects table — useful for onboarding, handoffs, and architecture reviews.
