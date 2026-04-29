# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.2.0](https://github.com/emgiezet/llmwiki/compare/v2.1.0...v2.2.0) (2026-04-29)


### Features

* v1.3.0 — rich project metadata + client-level config ([ed79612](https://github.com/emgiezet/llmwiki/commit/ed79612d8165f8fddc1fb8ebbc032dc3a9b3c176))

## [2.1.0](https://github.com/emgiezet/llmwiki/compare/v2.0.0...v2.1.0) (2026-04-23)


### Features

* **wiki:** prefix index files with client/project name ([28aabc4](https://github.com/emgiezet/llmwiki/commit/28aabc4a269fafbc071e492f98946819e6af1648))

## [2.0.0](https://github.com/emgiezet/llmwiki/compare/v1.0.0...v2.0.0) (2026-04-23)


### ⚠ BREAKING CHANGES

* **hook:** the Claude Code Stop hook is now Node.js instead of Python. Existing 1.0.x installs auto-migrate on `llmwiki hook install claude-code` (the legacy stop-hook.py is removed and hooks.json is rewritten to invoke `node` instead of `python3`), but node ≥ 18 must be on PATH.

### Features

* **hook:** per-tool hook dispatcher with native integrations, Node-migrated Claude Code hook ([9ba752a](https://github.com/emgiezet/llmwiki/commit/9ba752a0c970433f5e84aa06675bba0ebd0ff974))
* **llm:** add gemini-cli, codex, opencode, pi as LLM backends ([68cbd40](https://github.com/emgiezet/llmwiki/commit/68cbd4090ad92c11a94447a119c21d97619bea98))
* v1.1.0 — support opencode, codex, gemini-cli, pi + Node-migrated Claude Code hook ([19cdf06](https://github.com/emgiezet/llmwiki/commit/19cdf06c3b73b6871ebf219ab1d815473edaf373))

## 1.0.0 (2026-04-22)


### Features

* add --note-stdin flag to absorb for hook integration ([85008d5](https://github.com/emgiezet/llmwiki/commit/85008d5f7de16bd5c51f96a7532cc16c4bd1549c))
* add absorb and materialize commands for incremental wiki building ([2f46456](https://github.com/emgiezet/llmwiki/commit/2f464568acc6c5dd9834c02dee450ab5795459dc))
* add AbsorbSession for incremental memory-based fact extraction ([3ae5f07](https://github.com/emgiezet/llmwiki/commit/3ae5f076afe379356a69128a919096fd5b734a75))
* add anthropic-sdk-go dependency ([96ca6ff](https://github.com/emgiezet/llmwiki/commit/96ca6ffafbcee79c01b6943d652ad53f038b7623))
* add BuildMaterializePrompt for memory-only wiki generation ([624c247](https://github.com/emgiezet/llmwiki/commit/624c247c61b138dd1f960ea2a9f58fd1cac81212))
* add hook install/uninstall/status for Claude Code Stop hook integration ([caa748d](https://github.com/emgiezet/llmwiki/commit/caa748d84f4799d7f848dc721912ae7b4ee7c769))
* add MaterializeFromMemory for memory-only wiki generation ([439ab0c](https://github.com/emgiezet/llmwiki/commit/439ab0c4145b159d5e2a0b354be5a682a599a313))
* auto-generate tags from LLM output into front matter ([9f6fa07](https://github.com/emgiezet/llmwiki/commit/9f6fa07e98bcba03597e407547f73ad03c18749f))
* ClaudeAPILLM using Anthropic Go SDK ([e6ac811](https://github.com/emgiezet/llmwiki/commit/e6ac811472d527151473f85beaab902be1f19233))
* ClaudeCodeLLM shells out to claude -p ([d19015b](https://github.com/emgiezet/llmwiki/commit/d19015be57f36d53cc9545538169c0016637f5de))
* config loading with global + per-project YAML ([fbe3ce6](https://github.com/emgiezet/llmwiki/commit/fbe3ce6f8dd5499bfd2afe7cf028aea9ea2dc867))
* cross-file wiki links with auto-linking after ingest ([08b414c](https://github.com/emgiezet/llmwiki/commit/08b414c11459355a529ec5c4f6377448e932f638))
* detect PHP, Docker, Maven, Gradle, Ruby, and src/ services ([52ccf13](https://github.com/emgiezet/llmwiki/commit/52ccf13cca5485cf8e92d71fdc2f29aa92e3b3c1))
* foundation for client index — sections, meta types, summary reader ([eabf10c](https://github.com/emgiezet/llmwiki/commit/eabf10c8175cd8d8fe6b067792c1b59c9ca36def))
* ingestion pipeline — scan, prompt, LLM, write wiki ([285b699](https://github.com/emgiezet/llmwiki/commit/285b6992de84bd2bbd26c68bfd751a2059e648fa))
* install hook as Claude Code plugin instead of patching settings.json ([dae83b3](https://github.com/emgiezet/llmwiki/commit/dae83b3e5ba8681d8eedb3455ba2ca89f4b60589))
* integrate graymatter persistent memory layer ([d23b777](https://github.com/emgiezet/llmwiki/commit/d23b7772566ec41c6480591a4a4966f6d66f6928))
* list, context, and query commands ([7e1dc67](https://github.com/emgiezet/llmwiki/commit/7e1dc67eec6491a0fad8b954556390fd70b4d8a4))
* LLM interface, factory, and FakeLLM test stub ([4dcb0e2](https://github.com/emgiezet/llmwiki/commit/4dcb0e2315299cb07612ed3ae94631ca81db7b54))
* LLM prompt builders for project and service ingestion ([c495c7c](https://github.com/emgiezet/llmwiki/commit/c495c7c8ec0922175bf6f9ab1fd4bb239614875f))
* **memory:** queue absorb sessions when db is busy, drain on next successful run ([94ab114](https://github.com/emgiezet/llmwiki/commit/94ab1144804bbba86c8d391a45c425b2a6fe7982))
* mermaid diagrams, enhanced API docs, swagger scanner patterns ([b05974a](https://github.com/emgiezet/llmwiki/commit/b05974a9d68fc01b69adc8e33153b39d96bad5fe))
* OllamaLLM REST client with mock server test ([1f25bbb](https://github.com/emgiezet/llmwiki/commit/1f25bbb2b878c1d3a8bc4694d5fbc8fde0746446))
* per-client and multi-project index generation with C4 diagrams ([f92fbcf](https://github.com/emgiezet/llmwiki/commit/f92fbcff239493c55df92bd93bd56e7fbcacab79))
* project scanner and service detector ([2ece136](https://github.com/emgiezet/llmwiki/commit/2ece136bcf240affee3f7c4bc2e9bc30490087b3))
* register hook command in main ([0f2566b](https://github.com/emgiezet/llmwiki/commit/0f2566bcdab705859fdf379821af09bc3e5f039c))
* release pipeline, autoupdate, and configurable extraction presets ([0c4097b](https://github.com/emgiezet/llmwiki/commit/0c4097b60d43796c66e248026761d4150bcccbad))
* release pipeline, autoupdate, and configurable extraction presets ([d40c498](https://github.com/emgiezet/llmwiki/commit/d40c498d66b4775e76bdc5d790ca58dfb8d7c133))
* scaffold llmwiki CLI with cobra ([0b09477](https://github.com/emgiezet/llmwiki/commit/0b09477253c87c7614aa857ffbaed9af6a792514))
* wiki entry format with YAML front matter ([00e5ad0](https://github.com/emgiezet/llmwiki/commit/00e5ad0b7f8445ed8c543821c211b4db189934fb))
* wiki index (_index.md) read/write and upsert ([d8c57cd](https://github.com/emgiezet/llmwiki/commit/d8c57cd4cf53bee2685420be994f78e9a5fb7fb6))
* wire ingest command with --service flag ([9f28376](https://github.com/emgiezet/llmwiki/commit/9f2837695d9ac4b0cc35fe9503629434ba126bfb))


### Bug Fixes

* correct transcript parser to handle Claude Code JSONL envelope format ([f7fad06](https://github.com/emgiezet/llmwiki/commit/f7fad06db03333d028b09ad7d3be2c9ad6fc07b3))
* deterministic service ordering and skip missing service dirs ([42d0e80](https://github.com/emgiezet/llmwiki/commit/42d0e80abb01e0fd771a67eeb550da8c66a55c03))
* document missing Path field in MaterializeFromMemory ([3508747](https://github.com/emgiezet/llmwiki/commit/3508747eb46e1080c553b94a32e6ebcb18408592))
* fail-fast on memory guard before LLM init, add API key env fallback in absorb ([5e59262](https://github.com/emgiezet/llmwiki/commit/5e59262a49564265ee8bf1db7c5549bd9e526da6))
* improve BuildMaterializePrompt update branch clarity and test precision ([3853dc5](https://github.com/emgiezet/llmwiki/commit/3853dc504e46e4b0d859039f5c0b59def4b34062))
* **memory:** cap Close() wait + fast-fail on lock contention in hook path ([c8cf596](https://github.com/emgiezet/llmwiki/commit/c8cf59670d1087a90863ce81630b5461b7c4c0a0))
* **security:** allowlist ollama_host to loopback by default (D4) ([217aee4](https://github.com/emgiezet/llmwiki/commit/217aee408152a956dd7a0da7e2d9ad75d7109d67))
* **security:** bound HTTP + subprocess deadlines on LLM calls (D6) ([f7c35a1](https://github.com/emgiezet/llmwiki/commit/f7c35a1ceb4f155bda249020a575f09e31014e86))
* **security:** fence untrusted prompt input, scrub LLM response (D3) ([310ddf2](https://github.com/emgiezet/llmwiki/commit/310ddf265fd02b3d07ce0670f65536131bc99c8a))
* **security:** refuse to read non-regular files in walk callbacks (D15) ([8c9c9c9](https://github.com/emgiezet/llmwiki/commit/8c9c9c98cc6385dc6e371114f895fd179024c673))
* **security:** reject path traversal in customer/project/service/type + bump toolchain ([9582c63](https://github.com/emgiezet/llmwiki/commit/9582c63f5b3f8b17d615ffeb64dd028973f3c680))
* **security:** validate transcript_path in Stop hook (D2) ([95350f8](https://github.com/emgiezet/llmwiki/commit/95350f8e32490740eea0ef3d303631b941f89c23))
* **security:** warn when anthropic_api_key is stored in plaintext config (D7) ([560c0a1](https://github.com/emgiezet/llmwiki/commit/560c0a1d7b5abb5b05245a782ae37f2703982a6a))
* **security:** wave 4 low-severity hardening (stdin bound, stderr redaction, utf-8 check, claude path override, hook logging) ([aaffddc](https://github.com/emgiezet/llmwiki/commit/aaffddc8ec53c64c04001e525df7c4cd11c1e818))
* sentinel error for empty absorb, nil guard in materialize, API key comment ([d06ba16](https://github.com/emgiezet/llmwiki/commit/d06ba16fe3831e270368fd46bdf76d8a308398f7))
* **test:** pass context.TODO to DrainAbsorbQueue instead of nil (SA1012) ([d933c88](https://github.com/emgiezet/llmwiki/commit/d933c88d842a18367358b24ffb683d19818874c0))
* use correct directory names for personal/oss project types ([00f6eb3](https://github.com/emgiezet/llmwiki/commit/00f6eb36627838cc73c4dcfac059a7c112679975))
* use errors.Is and os.UserHomeDir in config ([bb5bfb7](https://github.com/emgiezet/llmwiki/commit/bb5bfb70edc7789003155fb7dd66b3568ef74db0))
* use errors.Is for fs errors and avoid slice aliasing in hook.go ([5acb181](https://github.com/emgiezet/llmwiki/commit/5acb181ad65962b553bcf7d4df5bb8986dc0bbd0))

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
