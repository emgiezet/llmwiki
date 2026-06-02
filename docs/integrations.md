# Integrations

Supported AI coding tools and session-capture hooks, plus Obsidian and NanoClaw.

[← Back to README](../README.md)

## Supported AI coding tools

llmwiki supports seven LLM backends and five hook-based session-capture integrations:

| Tool         | Backend (`llm:` value) | Hook install         | Capture mechanism                                      |
|--------------|------------------------|----------------------|--------------------------------------------------------|
| Claude Code  | `claude-code` (default) | ✅ native             | Plugin at `~/.claude/plugins/llmwiki/` (Node hook)      |
| Claude API   | `claude-api`           | —                    | SDK-only, no session concept                           |
| Ollama       | `ollama`               | —                    | REST, no session concept                               |
| codex        | `codex`                | ✅ native `notify` TOML | `~/.codex/config.toml` → `notify = ["node", …]`         |
| opencode     | `opencode`             | ✅ native plugin       | `~/.config/opencode/plugins/llmwiki.ts` (`session.idle`) |
| pi           | `pi`                   | ✅ native extension    | `~/.pi/agent/extensions/llmwiki.ts` (`agent_end`)       |
| gemini-cli   | `gemini-cli`           | ⚠ wrapper fallback   | Shell function in `~/.bashrc` / `.zshrc` / fish config  |

Hook-based capture is optional — all seven can be used as a plain ingest backend without any hook setup.

## Hook install / uninstall / status

```bash
# Install for one tool
llmwiki hook install claude-code
llmwiki hook install codex
llmwiki hook install opencode
llmwiki hook install pi
llmwiki hook install gemini-cli     # shell-wrapper fallback; source your rc after

# Install for everything detected
llmwiki hook install all

# Remove
llmwiki hook uninstall claude-code  # or any tool name, or 'all'

# See what's installed
llmwiki hook status
```

Each tool's install is idempotent. Uninstall preserves user-authored config (TOML keys outside our marker block, unrelated rc-file lines, etc.).

## How the captures work

- **Claude Code**: `Stop` hook fires after every qualifying turn (assistant response >300 chars with at least one analytical tool call). A Node script reads the transcript, extracts the last response, and pipes to `llmwiki absorb`.
- **codex**: top-level `notify = ["node", "~/.llmwiki/hooks/codex-absorb.js"]` in `~/.codex/config.toml`. Codex appends a JSON payload (with `last-assistant-message`, `cwd`) as the final argv on every turn end; the wrapper unpacks it and forwards.
- **opencode**: TS plugin subscribes to `session.idle`, grabs the last assistant message via `client.session.messages(...)`, pipes it through Bun's `$` shell helper to `llmwiki absorb`.
- **pi**: TS extension calls `pi.on("agent_end", …)`, which fires once per user prompt after tools complete.
- **gemini-cli**: no native hook API. The installer writes a shell function wrapper that intercepts `gemini -p "…"` invocations only (interactive TUI passes through unchanged); it tees stdout and pipes to `llmwiki absorb` on success.

## Hook requirements

- `memory_enabled: true` in `~/.llmwiki/config.yaml` (otherwise `absorb` is a no-op).
- `llmwiki` on `$PATH` — every hook shells out to the binary.
- **Node.js ≥ 18** for the Claude Code and codex hooks. You already have it if you use any of the new agents (all ship as npm packages); the installers fail fast with a clear message otherwise.

## Obsidian compatibility

![llmwiki graph view in Obsidian](obsidian2.png)

The wiki directory works as an [Obsidian](https://obsidian.md/) vault out of the box:

1. Open Obsidian, choose "Open folder as vault", select `~/llmwiki/wiki/`
2. Mermaid diagrams render natively in preview mode
3. Cross-file links are clickable — navigate from client to project to service
4. YAML front matter shows as properties
5. Tags are searchable via the tag pane
6. Graph view visualizes your entire knowledge base

## NanoClaw

llmwiki works with [NanoClaw](https://nanoclaw.com) — a Discord bot that can query your wiki knowledge base and answer project questions directly in your Discord server.

![NanoClaw answering a question about llmwiki in Discord](nanoclaw-discord.png)

Ask NanoClaw questions about any of your tracked projects and it draws on the wiki entries llmwiki generated. See [nanoclaw-integration.md](nanoclaw-integration.md) for setup instructions.
