# Configuration

How llmwiki resolves config (global → client → project), section presets, non-code projects, and the NDA local-LLM recipe.

[← Back to README](../README.md)

## Configuration

llmwiki resolves effective config by merging three layers, with each later layer overriding the previous one **per field**:

```
global  →  client  →  project
```

### Per-project: `llmwiki.yaml` (at project root)

Drop this in the project root (or let `llmwiki init` create it) to set its type, customer, and LLM backend:

```yaml
type: client                # client | personal | oss
customer: acme
status: discovery           # production | poc | discovery — drives section visibility
llm: codex                  # claude-code | claude-api | ollama | gemini-cli | codex | opencode | pi
ollama_model: llama3.2

# Where generated wiki files land (see "Docs alongside code" above).
output_mode: central        # central (default) | local | both
local_docs_dir: docs/llmwiki

# External systems for MCP-connected AI agents to crawl and for humans to click.
# Any key is allowed; well-known ones (github/gitlab/jira/confluence/slack/…)
# get nicer rendering. Project keys override client keys one at a time.
links:
  jira: https://acme.atlassian.net/jira/software/projects/BILL

# Team template — optional; well-known scalar keys + free-form notes.
team:
  oncall_channel: "#bill-api-oncall"

# Cost perspective — optional; drives a calculation table when the numbers
# are there, otherwise renders a how-to-estimate framework in the wiki.
cost:
  infra_monthly_usd: 1200
  team_fte: 2.5
```

### Per-client: `~/.llmwiki/clients/<customer>.yaml` (v1.3.0+)

Baseline every project with `customer: <name>` inherits from. Projects override any field per-key. Scaffold with `llmwiki client init <customer>`.

```yaml
# Client baseline — every project under customer: acme inherits these.
status: production
llm: codex

links:
  github: https://github.com/acme
  confluence: https://acme.atlassian.net/wiki/spaces/ACME
  slack: https://acme.slack.com/archives/C01ACME

team:
  lead: "jane.doe@acme.com"
  oncall_channel: "#acme-ops"
  escalation: "ops-manager@acme.com"

cost:
  # Team rate flows down to every project unless the project sets its own.
  team_fte_rate_usd_monthly: 18000
  notes: "Fully loaded (salary + benefits + overhead)."

# LLM / extraction defaults also inherit client-wide.
extraction:
  preset: software
  max_tokens: 4000
```

Inspect the effective config for any project with:
```bash
llmwiki client show acme --project ~/workspace/billing-api
llmwiki client list      # customers with a client config file
```

### Global: `~/.llmwiki/config.yaml`

```yaml
wiki_root: ~/llmwiki/wiki
llm: claude-code
ollama_host: http://localhost:11434
anthropic_api_key: ""   # or set ANTHROPIC_API_KEY env var
memory_enabled: false   # enable graymatter persistent memory
memory_mode: project    # project (default) | global — see Memory section
memory_dir: ~/.llmwiki/memory   # used when memory_mode: global

# Optional PATH overrides for agentic-coder CLIs (empty = look up by name):
claude_binary_path: ""
gemini_binary_path: ""
codex_binary_path: ""
opencode_binary_path: ""
pi_binary_path: ""

# Document text extraction (see "Non-technical projects" below). Maps a file
# extension to a command template; {{input}} is replaced with the file path and
# the command must write extracted text to stdout. Defaults target macOS/Linux;
# override per-OS or per-project. A missing tool is skipped, not fatal.
extractors:
  .pdf:  "pdftotext {{input}} -"
  .docx: "pandoc {{input}} -t plain"
  .odt:  "pandoc {{input}} -t plain"
  .epub: "pandoc {{input}} -t plain"
```

Per-project config overrides client config overrides global. If none exist, defaults to `claude-code` with wiki at `~/llmwiki/wiki/`.

### Project status & section presets

`status:` picks the default section list when no explicit preset/sections override is set:

| Status | Section shape |
|---|---|
| `production` (default) | Domain, Architecture, Services, Features, Flows, System/Data diagrams, Integrations, Tech Stack, Configuration, Notes, Bug Summary *(v1.4.0)*, Tags |
| `discovery` | Domain, **Open Questions**, **Requirements**, **Scope**, **Assumptions**, **Stakeholders**, Integrations, Notes, Tags |
| `poc` | Domain, **Scope & Assumptions**, Architecture (light), Tech Stack, **Success Criteria**, Notes, Tags |

Override per-run with `llmwiki ingest <path> --status discovery` or per-project via `status:` in `llmwiki.yaml`. Discovery projects automatically also get `docs/*.md`, `notes/*.md`, `PRD.md`, and `requirements.md` pulled into the scanner input.

## Links, Team, Cost rendering

Every wiki file rendered after v1.3.0 carries three optional body sections generated deterministically by llmwiki (not by the LLM — so the LLM can't hallucinate team members or cost figures):

- **`## Links`** — clickable list with well-known keys (github/jira/confluence/slack/…) getting nice labels and icons. Inherited-from-client entries annotated `*(inherited from client)*`.
- **`## Team`** — per-field markdown list with email addresses auto-linked as `mailto:` and `#channels` passed through verbatim.
- **`## Cost`** — calculation table when numbers are set, or a how-to-estimate framework (the framework IS the doc — shows the exact YAML to fill in) when empty.

Any of these sections is omitted entirely when the corresponding YAML block is empty, so unused metadata leaves no empty headers.

## Docs alongside code (output mode)

By default wiki files live in the central `~/llmwiki/wiki/` tree. Set `output_mode` in `llmwiki.yaml` to also (or only) write them into the project repository, so a single PR diff shows both the code change and the doc change:

```yaml
output_mode: both              # central (default) | local | both
local_docs_dir: docs/llmwiki   # where local docs land (default)
```

- `central` — existing behaviour, wiki only in `~/llmwiki/wiki/`
- `local` — wiki only inside `<project>/<local_docs_dir>/`
- `both` — written to both locations

## LLM backends

| Backend | Config | Best for |
|---------|--------|----------|
| Claude Code CLI | `claude-code` (default) | Uses your Claude Code subscription. No API key needed. |
| Claude API | `claude-api` | Fast bulk ingestion. Requires `ANTHROPIC_API_KEY`. |
| Ollama | `ollama` | NDA code, air-gapped environments, cost control. |

## Non-code projects & document extraction

llmwiki isn't only for code. Point it at a folder of notes, research, articles, or
lecture material and it builds a prose-oriented wiki the same way it documents a
service. Two pieces make this work — **document text extraction** and
**knowledge presets** — described below.

### Non-technical projects (notes, research, articles, lectures)

llmwiki can document corpora whose source material is prose in `.pdf`, `.docx`,
`.odt`, or `.epub` rather than code. Two pieces make this work:

**1. Document text extraction.** During a scan, files matching a configured
extension are run through an external converter and their text is folded into
the LLM input alongside any README/Markdown. The converters are configured in
`extractors:` (global config, overridable per-project) — defaults below target
macOS and Linux:

| Format | Default tool | Install |
|---|---|---|
| `.pdf` | `pdftotext` | `poppler-utils` (apt) / `poppler` (brew) |
| `.docx` / `.odt` / `.epub` | `pandoc` | `pandoc` (apt/brew) |

Extraction is always on when the tool is available; if a converter isn't on
`PATH` the file is skipped with a warning rather than failing the run. At most
50 documents are extracted per scan, each truncated to ~50 KB. On Windows, set
`extractors` to whatever converters you have installed — no code change needed.

**2. Knowledge presets.** Set `extraction.preset` in `llmwiki.yaml` to a
prose-oriented section bundle instead of the code-oriented defaults:

```yaml
type: personal              # directory still: personal/ | clients/ | opensource/
extraction:
  preset: research          # or: notes
```

| Preset | Sections |
|---|---|
| `notes` | Summary, Key Topics, Key Points / Findings, Open Questions, Tags |
| `research` | Summary, Key Topics, Key Points / Findings, References / Sources, Glossary, Open Questions, Tags |

The project `type` still controls only where the file lands; the preset controls
its shape. Point `llmwiki ingest` at the folder of documents as usual.

## NDA projects: local-LLM override

Keep your default backend on the Claude Code subscription for everyday work, and
override **just the confidential project** to a local Ollama model — so that
project's code (and any extracted document text) never leaves your machine. No
cloud API is called for it.

**Global `~/.llmwiki/config.yaml`** (set once, e.g. via `llmwiki setup`):

```yaml
llm: claude-code        # default backend for every project
ollama_host: http://localhost:11434
```

**The NDA project's `llmwiki.yaml`** (at the repo root — overrides the global backend for this project only):

```yaml
type: client
customer: acme-confidential
llm: ollama                   # this project only — never calls the cloud
ollama_model: qwen3-coder:30b # any local model you've pulled
```

Then:

```bash
ollama pull qwen3-coder:30b   # one-time, on the machine that runs ingest
llmwiki ingest ~/work/acme-confidential
```

The entire pipeline — directory scan, prompt construction, and generation — runs
against the local model, so no client source or document text is sent to an
external API. `ollama_host` is a **global** setting (per-project override is the
model, not the host). Verify the effective backend for a project with
`llmwiki client show <customer> --project <path>`.

## Interactive setup (`llmwiki setup`)

Run `llmwiki setup` once to configure the global `~/.llmwiki/config.yaml` — it walks you through the LLM backend, wiki root, memory, and shows which document extractors are available:

```text
➜  llmwiki setup
Detected tools:
  claude: ✓ found
  ollama: ✓ found
LLM backend?
  * 1) claude-code (Claude Code subscription)
    2) claude-api (uses ANTHROPIC_API_KEY)
    3) ollama (local models)
Choice [1]: 1
Wiki root [/home/mgz/llmwiki/wiki]:
Enable memory (graymatter)? [Y/n]: y
Memory mode?
  * 1) project (per-project store, default)
    2) global (single shared store)
Choice [1]:
Document extractors (detection only — edit `extractors` to change):
  .docx  pandoc [✓ found]
  .epub  pandoc [✓ found]
  .odt   pandoc [✓ found]
  .pdf   pdftotext [✓ found]

Summary:
  llm:            claude-code
  wiki_root:      /home/mgz/llmwiki/wiki
  memory_enabled: true
  memory_mode:    project
Save to ~/.llmwiki/config.yaml? [Y/n]: y
✓ /home/mgz/.llmwiki/config.yaml
```

Running `llmwiki init` with **no flags** in a terminal launches the equivalent per-project wizard. Passing any flag (or running in CI / non-TTY) keeps the non-interactive behaviour shown below. Both wizards load an existing config as defaults, so they double as editors.

Tab completion is built in — enable it for your shell with `source <(llmwiki completion zsh)` (or `bash`/`fish`), then `llmwiki s<Tab>` suggests `setup`.
