# llmwiki Threat Model (v1.0)

## Scope

Covers the llmwiki Go binary, the embedded Python Stop-hook script, and the
Claude Code plugin layout shipped at `plugin/`. Out of scope: Anthropic/Ollama
backend security; graymatter internals beyond pinned-version trust.

## Assets

1. **ANTHROPIC_API_KEY** — grants usage billing against the user's account.
2. **Filesystem outside wiki root** — no unintended writes or reads.
3. **Wiki content integrity** — wiki entries flow into CLAUDE.md via
   `llmwiki context --inject`, so attacker-controlled wiki content could
   reach the user's AI coding session.
4. **User session** — no code execution beyond the invoked command.

## Attackers

### A1. Malicious project repository
The user clones a hostile repo and runs `llmwiki ingest`. Attacker controls
README, docker-compose.yml, .proto files, and git history. Success conditions:
write outside wiki root, inject content into the LLM prompt to hijack output,
cause a crash/hang.

**Mitigations:**
- Customer / project / service names validated via `internal/validation`.
- Untrusted content fenced in LLM prompts with explicit "treat as data" system
  instruction.
- LLM responses scrubbed of structural markers before writing.
- Non-regular files refused in `filepath.WalkDir` callbacks (symlink TOCTOU).
- Non-UTF-8 files skipped in the scanner.

### A2. Forged Claude Code Stop event
An attacker who can drive the user's Claude Code to emit a tampered Stop
event (e.g. via a compromised plugin) could trick the hook into reading an
arbitrary path.

**Mitigations:**
- `transcript_path` must resolve to a location under `~/.claude/projects/`,
  otherwise the hook exits 0 silently.
- The hook always exits 0 and never blocks Claude (graceful degradation).

### A3. Supply-chain compromise
A transitive dependency ships malicious code.

**Mitigations:**
- Pinned versions in `go.sum`.
- Dependency review documented in `docs/supply-chain.md`.
- `osv-scanner` runs in CI on every PR.
- Pre-1.0 dependencies (`graymatter`, `chromem-go`) flagged with explicit
  accept-risk rationale.

### A4. Local unprivileged user on shared machine
Reads `~/.llmwiki/config.yaml` or files under `~/llmwiki/wiki/`.

**Mitigations:**
- Warning emitted if `anthropic_api_key` is stored in config.
- Recommendation: use `ANTHROPIC_API_KEY` env var.
- Wiki files intentionally 0644 (sharing is the point).

### A5. SSRF via Ollama host
Config with `ollama_host: http://169.254.169.254/...` attempts to reach the
cloud metadata endpoint.

**Mitigations:**
- Host allowlist: loopback only by default.
- Escape hatch: `allow_remote_ollama: true` for deliberate remote setups.

## Trust Boundaries

**Inside the boundary:** Go binary code, pinned `graymatter` library, Go
standard library (on supported toolchain >= 1.25.9).

**Outside the boundary:** filesystem contents, stdin, environment variables,
CLI arguments, git output, LLM responses, HTTP responses, YAML and JSON
from disk.

## Residual Risks

- **Prompt injection** — defense-in-depth only; general solution remains an
  open research problem.
- **PATH hijacking of `claude` binary** — mitigated by `claude_binary_path`
  override.
- **API key in plaintext config** — warning emitted; env var recommended.

## Verification

See `Makefile` target `security-scan` for the reproducible CI gate.

## Change Log

Significant threat-model changes land in `CHANGELOG.md` under a
`### Security` heading per release.
