# Contributing to llmwiki

Thanks for helping out! llmwiki is a single Go binary with a small dependency-free
npm wrapper. This guide covers how to build, test, and submit changes.

## Prerequisites

- Go (version per [`go.mod`](go.mod))
- Node ≥ 18 (only to run the `npm/` wrapper tests)
- Optional: `golangci`/`staticcheck`, `goreleaser` (for release config changes), Docker

## Build & test

```bash
go build ./...                        # build everything
go build -o llmwiki .                 # build the binary
go test ./...                         # run all tests
go test ./internal/ingestion/... -v   # run a specific package
go vet ./...                          # static analysis
```

The npm wrapper has its own dependency-free test suite (Node's built-in runner):

```bash
cd npm && node --test "test/*.test.js"
```

CI mirrors these (`.github/workflows/ci.yml`): `go vet`, `go build`, `go test -race`,
`staticcheck`, and the npm wrapper tests. The security gate
(`.github/workflows/security.yml`) additionally runs `gosec`, `govulncheck`,
`osv-scanner`, and `gitleaks`.

## Tests are required

Every new feature or bugfix must ship with tests. The codebase favors
test-driven development — the `internal/llm` package provides `NewFakeLLM` as a
test double so you can exercise the pipeline without calling a real model.

## Commit messages

Releases are automated with [release-please](https://github.com/googleapis/release-please),
which reads [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — new feature → minor bump
- `fix:` — bug fix → patch bump
- `feat!:` or a `BREAKING CHANGE:` footer → major bump
- `docs:`, `chore:`, `refactor:`, `test:`, … → no release on their own

Keep the subject in the imperative mood and scoped to one logical change.

## Pull requests

1. Branch off `main`.
2. Make the change with tests; ensure `go test ./...` and the npm tests pass.
3. Run `go vet ./...` and `staticcheck ./...`.
4. Open the PR with a clear description. CI must be green before merge.

## Architecture

See [CLAUDE.md](CLAUDE.md) for the full pipeline (`scanner` → `ingestion` → `llm`
→ `wiki`), the storage layout, and how the `check`, `mcp`, and `context` commands
fit together.

## License

By contributing you agree your contributions are licensed under the project's
[MIT license](LICENSE).
