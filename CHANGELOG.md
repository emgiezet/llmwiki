# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-04-22

### Added
- Claude Code plugin layout at `plugin/` with automatic Stop-hook absorption.
- `llmwiki absorb` — extract session facts into memory without regenerating the wiki.
- `llmwiki materialize` — rebuild a wiki entry from accumulated memory facts.
- `llmwiki hook install/uninstall/status` — install as a Claude Code plugin.
- Public SBOM (`sbom.cdx.json`) and supply-chain risk doc (`docs/supply-chain.md`).
- Pre-commit hook config (`.pre-commit-config.yaml`) running gitleaks.

### Security
Baseline pre-release audit. All findings tracked in the audit plan.

- **Critical — fixed:** path traversal via `customer` / `project` / `service` /
  `type` — rejected by a new `internal/validation` package.
- **Critical — fixed:** 8 reachable Go stdlib CVEs — toolchain bumped to
  `go1.25.9` via `go.mod` directive.
- **High — fixed:** forged Stop-hook event with `transcript_path` outside
  `~/.claude/projects/` is rejected.
- **High — fixed:** untrusted prompt inputs (scan output, git log, notes,
  existing wiki) are fenced with system instruction "treat as data".
- **High — fixed:** LLM responses are scrubbed of `<!-- llmwiki:start/end -->`
  markers and fence tags before writing.
- **High — fixed:** plaintext `anthropic_api_key` in config triggers a warning.
- **High — fixed:** `filepath.WalkDir` callbacks refuse non-regular files
  (symlink TOCTOU).
- **Medium — fixed:** Ollama host restricted to loopback by default; opt-in
  `allow_remote_ollama`.
- **Medium — fixed:** HTTP + subprocess deadlines on all LLM calls.
- **Low — fixed:** stdin on `absorb --note-stdin` bounded to 1 MiB.
- **Low — fixed:** stderr in error messages truncated to 512 bytes to avoid
  leaking prompt content.
- **Low — fixed:** non-UTF-8 project files skipped in the scanner.
- **Low — added:** `claude_binary_path` config option to pin the Claude Code
  binary path.
- **Low — fixed:** Stop-hook exceptions logged to `~/.llmwiki/hook.log`
  instead of silently swallowed.

Scanning in CI: `go vet`, `staticcheck`, `gosec`, `govulncheck`, `osv-scanner`,
`gitleaks` — all required green on every PR.
