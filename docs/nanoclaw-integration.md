# NanoClaw Integration

[NanoClaw](https://nanoclaw.com) is a Discord bot that can answer questions about your projects directly in your Discord server. When combined with llmwiki, it becomes a team-wide knowledge assistant — any developer on your server can ask "how does the billing service handle retries?" and get an answer grounded in your actual codebase.

## How it works

1. llmwiki scans your projects and writes structured wiki entries to `~/llmwiki/wiki/`
2. You point NanoClaw at your wiki directory (or expose it via a path it can read)
3. Team members ask questions in Discord — NanoClaw reads the wiki and answers

## Setup

### 1. Install llmwiki and ingest your projects

```bash
go install github.com/emgiezet/llmwiki@latest

# Ingest each project you want NanoClaw to know about
llmwiki ingest ~/workspace/billing-api
llmwiki ingest ~/workspace/notification-service
llmwiki ingest ~/workspace/auth-service
```

### 2. Configure NanoClaw to read your wiki

In your NanoClaw bot configuration, point it at your wiki root:

```
wiki_path: ~/llmwiki/wiki
```

NanoClaw will index the markdown files and make them queryable in Discord.

### 3. Ask questions in Discord

In any channel where NanoClaw is active:

```
tell me how the billing service handles retries
which services use gRPC?
what does the auth service expose?
```

NanoClaw reads the wiki entries llmwiki generated and answers in plain language.

## Keeping the wiki fresh

Re-run `llmwiki ingest` after significant changes to keep NanoClaw's answers current:

```bash
llmwiki ingest ~/workspace/billing-api
```

For ongoing sessions, use the Claude Code hook to accumulate facts automatically, then materialize:

```bash
# Install the Claude Code hook (captures analytical sessions automatically)
llmwiki hook install

# After several sessions, rebuild the wiki without a full re-scan
llmwiki materialize billing-api
```

## Tips

- Add a `llmwiki.yaml` to each project with `customer` and `type` fields — NanoClaw can filter by client or project type
- Use `llmwiki remember` to seed facts that the scanner can't detect (team decisions, migration history, quirks)
- The `llmwiki query` command gives you the same cross-project search locally, useful for verifying what NanoClaw will see
