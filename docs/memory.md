# Memory (graymatter)

Persistent, compounding memory across ingestion runs — modes, seeding, and the absorb queue.

[← Back to README](../README.md)

## Graymatter integration

`llmwiki init` can wire up [graymatter](https://github.com/gdgvda/graymatter) — a local vector memory store — as a passive layer on top of Claude Code sessions.

After initialisation each Claude Code session automatically saves a compact summary to `.graymatter/` when the session ends (via the `Stop` hook). The graymatter MCP server exposes those memories back to Claude Code, so context from past sessions is available without manual effort. The same Stop hook also runs a non-blocking `llmwiki check` on the files touched during the session and stores any staleness signal in memory, so you find out about drifting docs without going looking for them.

```
project/
├── .claude/
│   ├── settings.local.json     # Stop hook registered here
│   └── graymatter_stop.sh      # captures session summary → graymatter
├── .graymatter/                # local vector DB (git-ignored)
│   ├── gray.db
│   └── vectors/
└── .mcp.json                   # graymatter MCP server wired here
```

The memory capture is **passive and async** — it never blocks a session and produces no visible output.

## Memory configuration

Set `memory_enabled: true` in your global config to activate [graymatter](https://github.com/angelnicolasc/graymatter) integration. It reuses your existing `anthropic_api_key` for embeddings, and falls back to Ollama or keyword-only search if no key is available.

### Memory modes

| `memory_mode` | Store location | Lock scope | Cross-project recall |
|---|---|---|---|
| `project` *(default)* | `{projectDir}/.graymatter/` | per-project | no |
| `global` | `memory_dir` (`~/.llmwiki/memory/` by default) | process-wide | yes |

**Project mode** (default) gives each project its own isolated `gray.db`. Multiple agents working on different projects never block each other. This aligns with how the `graymatter` MCP server works by default.

**Global mode** keeps a single shared store — useful when you want `llmwiki recall` to search across all projects at once. Concurrent agents hitting the same store will contend on the bbolt file lock; llmwiki degrades gracefully (logs a warning, skips memory) rather than crashing.

### Worktree pattern

When a git worktree should share memory with its parent checkout, add to the worktree's `llmwiki.yaml`:

```yaml
memory_dir: /path/to/main-checkout/.graymatter
```

This per-project `memory_dir` override takes priority over the global mode, so the worktree reads/writes the same store as the main checkout.

### Seeding tribal knowledge

```bash
llmwiki remember --project my-api "billing service was rewritten from PHP to Go in Q1 2025"
llmwiki remember --project my-api "uses custom auth middleware in pkg/auth, not standard library"
llmwiki recall "which projects use gRPC?"
```

## Regenerating from captured facts

After a few hook-captured sessions, you can refresh a wiki entry without re-scanning the whole codebase:

```bash
llmwiki materialize my-project   # ~5–15K tokens vs 50–100K for full ingest
```

## Lock contention & the absorb queue

If the Stop hook fires while another process holds the memory DB (for example, you have `graymatter tui` open), llmwiki appends the session to a local queue file (`~/.llmwiki/memory/absorb-queue.jsonl` by default). The queue is drained the next time `llmwiki absorb` runs successfully, or explicitly via `llmwiki absorb-drain`.

## Incremental wiki building

The hook + materialize workflow is designed for ongoing sessions where a full `ingest` run would be too expensive. Facts accumulate silently across sessions; you run `materialize` when you want a refreshed wiki entry.

You can also capture explicit insights during a session:

```bash
llmwiki remember --project my-api "retry logic uses exponential back-off with jitter in pkg/retry"
```
