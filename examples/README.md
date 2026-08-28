# Examples

Real output, not hand-written. [`llmwiki.md`](llmwiki.md) is what `llmwiki ingest`
produced when run on **this repository itself** (a clean `git archive HEAD`
snapshot) with the default `claude-code` backend — domain, architecture, a
service map, Mermaid system/data-model diagrams, an integration map, a tech-stack
table, and auto-generated tags in the YAML front matter.

It's a single markdown file with no database and no SaaS — drop it in Obsidian,
render it on GitHub, or let an agent read it over MCP (`llmwiki mcp`).

Generate one for your own project:

```bash
llmwiki ingest /path/to/your/project
```

> The `path:` and `last_ingested:` fields were generated during ingest; `path`
> has been normalized here since the sample was produced from a temporary
> checkout.
