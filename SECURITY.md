# Security Policy

## Supported Versions

llmwiki follows semver. Only the latest minor release is actively supported
with security fixes.

| Version | Supported |
|---------|-----------|
| 1.x.y   | Yes       |
| < 1.0   | No        |

## Reporting a Vulnerability

Please **do not** open a public issue for security problems. Instead:

- Open a private security advisory on GitHub:
  <https://github.com/emgiezet/llmwiki/security/advisories/new>
- Or email the maintainer (see `AUTHORS` or commit history).

You can expect an initial acknowledgment within 72 hours. We aim to publish
a fix within 14 days for high-severity issues.

## Threat Model (Summary)

llmwiki runs on developer machines. It:
- Reads files from arbitrary project directories
- Spawns subprocesses (`claude`, `git`, `python3` via the Claude Code hook)
- Makes outbound HTTP calls (Anthropic API, Ollama)
- Writes markdown files under `~/llmwiki/wiki/`
- Optionally installs a Claude Code plugin at `~/.claude/plugins/llmwiki/`

**Assets protected:**
1. The user's `ANTHROPIC_API_KEY`
2. The filesystem outside the wiki root (no unintended writes)
3. Integrity of wiki content that flows into CLAUDE.md injections
4. The user's session (no code execution beyond the intended CLI)

**Primary attackers considered:**
1. Malicious project repositories (crafted README, docker-compose, git history)
2. Compromised transitive dependencies
3. Local unprivileged users on a shared machine
4. Forged Claude Code Stop-hook events

For the full threat model, see [docs/threat-model.md](docs/threat-model.md).

## Known Residual Risks

Documented and accepted for v1.0:

- **Prompt injection** — content from project files, git history, and LLM
  responses is always treated as untrusted data. llmwiki fences it in prompts,
  instructs the model to treat it as data, and scrubs the response before
  writing. Prompt injection remains an open research problem, so a fully
  general defense is not possible. See `docs/threat-model.md` for details.
- **PATH-based `claude` lookup** — by default llmwiki invokes `claude` via
  `exec.LookPath`. Users on multi-user systems with writable PATH entries can
  pin the binary with `claude_binary_path` in `~/.llmwiki/config.yaml`.
- **API key in config** — if `anthropic_api_key` is stored in the YAML
  config, llmwiki prints a warning at startup. Prefer the
  `ANTHROPIC_API_KEY` environment variable.

## Supply Chain

- SBOM: [`sbom.cdx.json`](sbom.cdx.json) (CycloneDX format)
- Dependency review: [`docs/supply-chain.md`](docs/supply-chain.md)
- Secret scanning: gitleaks runs on every PR via `.github/workflows/security.yml`

## Security Scanning

Contributors can reproduce the full CI security gate locally:

```bash
make security-scan
```

This runs `go vet`, `staticcheck`, `gosec`, `govulncheck`, `osv-scanner`, and
`gitleaks`. All must pass before a release.
