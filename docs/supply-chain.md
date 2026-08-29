# Supply Chain Risk Assessment

_Last updated: 2026-08-29 (dependency audit itself dates from 2026-04-22)_

This document summarizes the results of the pre-release dependency / supply chain
audit for `llmwiki`. It is generated from:

- `sbom.cdx.json` (CycloneDX SBOM — `cyclonedx-gomod mod`)
- `osv-scanner --lockfile=go.mod`
- `go-licenses report ./...`
- `go mod graph`

## Tooling snapshot

| Tool | Version / Source | Purpose |
|---|---|---|
| `cyclonedx-gomod` | latest (installed to `$GOPATH/bin`) | SBOM generation |
| `osv-scanner` | v2, latest (installed to `$GOPATH/bin`) | CVE lookup against OSV.dev |
| `go-licenses` | v1.6.0 | License classification |
| Go toolchain | 1.26.7 (module pins `go 1.26.7`) | Build + `go mod graph` |

SBOM file: `sbom.cdx.json` (~20 KB, 22 components including the root module).

The committed SBOM had drifted: it still described 15 components from the
2026-04-22 audit, so regenerating it here (policy: on every PR that touches
`go.mod`) picks up seven modules that landed with the MCP server and the update
checker — `modelcontextprotocol/go-sdk`, `google/jsonschema-go`,
`yosida95/uritemplate/v3`, `segmentio/asm`, `segmentio/encoding`,
`golang.org/x/oauth2` and `golang.org/x/mod`. None of them come from graymatter;
the v0.18.0 bump adds no modules of its own.

## Direct dependencies

These are the modules explicitly required by `go.mod`.

| Module | Version | License | Purpose | Trust rationale |
|---|---|---|---|---|
| `github.com/angelnicolasc/graymatter` | v0.18.0 | MIT | Agent memory substrate (`internal/memory/memory.go`) | Pre-1.0, single-author project. See "pre-1.0 risks" below. |
| `github.com/spf13/cobra` | v1.10.2 | Apache-2.0 | CLI command framework (`internal/cmd/*`, `main.go`) | De facto standard Go CLI lib; >37k GitHub stars; Kubernetes/Hugo/etc. depend on it. |
| `github.com/stretchr/testify` | v1.11.1 | MIT | Test assertions (test-only) | Ubiquitous Go testing lib; vendored by the entire ecosystem. |
| `gopkg.in/yaml.v3` | v3.0.1 | MIT / Apache-2.0 | YAML front matter + config parsing (`internal/wiki`, `internal/config`) | Canonical Go YAML implementation (`go-yaml`), maintained since 2011. |
| `golang.org/x/mod` | v0.35.0 | BSD-3-Clause | semver comparison in the update checker (`internal/update`) | Google-maintained; same SLA as stdlib. |

## Critical transitive dependencies

Pulled in indirectly via graymatter / cobra / testify but worth scrutinising.

| Module | Version | License | Pulled in by | Trust rationale |
|---|---|---|---|---|
| `github.com/anthropics/anthropic-sdk-go` | v1.36.0 | MIT | `graymatter` (and our own `internal/llm/claude-api` code path) | Official Anthropic SDK; published by the API vendor itself. |
| `github.com/philippgille/chromem-go` | v0.7.0 | **MPL-2.0** | `graymatter` (memory/embedding store) | Pre-1.0, single-author project. MPL adds license obligations; see below. |
| `go.etcd.io/bbolt` | v1.3.11 | MIT | `graymatter` (local KV store) | CNCF-adjacent, mature fork of BoltDB. |
| `github.com/tidwall/gjson`, `sjson`, `match`, `pretty` | v1.18.0 / v1.2.5 / v1.1.1 / v1.2.1 | MIT | `anthropic-sdk-go` (JSON manipulation) | Single author (@tidwall) but widely used, stable for years. |
| `github.com/spf13/pflag` | v1.0.10 | BSD-3-Clause | `cobra` | Paired with cobra; same trust tier. |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | Apache-2.0 | `cobra` (Windows-only no-op on Linux) | Tiny, unchanged since 2022. |
| `github.com/oklog/ulid/v2` | v2.1.0 | Apache-2.0 | `graymatter` (IDs) | Small, well-known, stable since 2019. |
| `golang.org/x/sync`, `golang.org/x/sys` | v0.16.0 / v0.47.0 | BSD-3-Clause | transitive | Google-maintained; same SLA as stdlib. |

## Vulnerability scan (OSV)

`osv-scanner` scanned 21 packages. **Zero CVEs were reported against any
third-party module** in our dependency graph. Every advisory it surfaced applied
to the Go standard library, matched against the `go` directive in `go.mod` (which
was `go 1.23.0` at the time of the audit):

- 25 advisories against `stdlib@1.23.0` in the "called" table.
- 13 further "uncalled" stdlib advisories (present in code paths we do not
  import).

### Mitigation — current stdlib policy

The `go` directive pins a full patch release (currently `go 1.26.5`) rather than a
minor version, so release binaries are built with a toolchain whose stdlib CVEs
are known-patched. Bump it whenever `govulncheck` reports a reachable stdlib
advisory.

Two scanners cover the stdlib from different angles, and they are wired
differently on purpose:

- **`govulncheck`** does call-graph reachability analysis, so it only reports
  advisories our code can actually reach. It is the hard gate. In `security.yml`
  it runs under `go-version: stable` (the newest release), because stdlib CVEs
  are patched upstream within days and a toolchain pin would otherwise turn every
  new advisory into a red weekly scan for something we cannot fix any faster.
- **`ci.yml`** runs `govulncheck` against the pinned toolchain from `go.mod` with
  `continue-on-error: true`. That is where an aging pin surfaces — as a warning on
  the next pull request, not a broken scheduled job.
- **`osv-scanner`** covers third-party modules. From v2 on it ignores the `go`
  directive by default, on the same reasoning: the directive is a minimum
  language version, not the toolchain a binary was built with, so matching
  advisories against it produces false positives. Leave `ScanGoModVersion` unset
  — turning it on re-adds stdlib findings with no reachability filter, which is
  what `govulncheck` is for. Note that osv-scanner **must** be installed from the
  `github.com/google/osv-scanner/v2/...` path; `@latest` on the pre-v2 path
  resolves to v1.9.2 (Dec 2024) forever, and v1 does scan the `go` directive.

No fix-version actions are required for third-party packages at this time.

## Licenses

`go-licenses report ./...` — 15 modules classified. Summary:

| License | Count | Modules |
|---|---|---|
| MIT | 9 | graymatter, anthropic-sdk-go, gjson, match, pretty, sjson, bbolt, yaml.v3, llmwiki (self) |
| Apache-2.0 | 3 | cobra, oklog/ulid/v2, (mousetrap via graph) |
| BSD-3-Clause | 2 | pflag, golang.org/x/sync, golang.org/x/sys |
| **MPL-2.0** | 1 | **philippgille/chromem-go** |

### Non-permissive findings

**`github.com/philippgille/chromem-go` is MPL-2.0.** MPL-2.0 is file-level
copyleft: any modification to a chromem-go source file must be released under
MPL-2.0, but static/dynamic linking from our own MIT-licensed code is
explicitly permitted without license contamination (MPL-2.0 §3.3).

**Action:** As long as we consume chromem-go unmodified via `go get`, there is
no distribution obligation beyond preserving the upstream LICENSE file. If we
ever fork/patch it, the patched files must remain MPL-2.0.

All other licenses are MIT / Apache-2.0 / BSD-3-Clause — unconditionally
compatible with our MIT release.

## Pre-1.0 / smaller-project risks

Two dependencies live under the `<1.0` single-maintainer risk class.

### 1. `github.com/angelnicolasc/graymatter@v0.18.0`

- **Status:** v0.x, single primary author (`angelnicolasc`).
- **Pulled in because:** provides the agent memory substrate; imported directly
  by `internal/memory/memory.go`. Also pulls in `chromem-go` and `bbolt`.
- **Risk:** API churn until 1.0; low bus factor; no published release cadence.
  The release cadence turned out to be fast rather than absent — v0.5.0 to
  v0.18.0 landed inside four months.
- **Decision:** _accept risk, version-pin_. We consume it via `go.mod` exact
  version (`v0.18.0`) with a pinned `go.sum` hash, which is sufficient to
  prevent surprise upgrades. Revisit at v1.0 or if we see unmaintained
  signals (stale issues, abandoned repo).
- **Fallback if upstream disappears:** the surface we use is small; vendor the
  module under `vendor/` and cut a local fork.
- **Version skew is a data-integrity risk, not just an API risk.** The library
  we link and the `graymatter` binary the generated Stop hook shells out to
  read and write the same `gray.db`. Fields added to the on-disk `Fact` after
  our pinned version (`superseded_by` v0.10.0, `confidence` v0.12.0, `pinned`
  v0.14.0) are dropped by an older decoder and then written back, resurrecting
  retired facts and clearing pins. Keep the pin at or above the binary users
  are told to install, and re-check the `Fact` struct on every bump.
- **The CLI is a separate module.** `go install
  github.com/angelnicolasc/graymatter/cmd/graymatter@vX.Y.Z` resolves the
  nested `cmd/graymatter` module (tag `cmd/graymatter/vX.Y.Z`), which carries
  its own `go.mod` and toolchain pin. Its version is not constrained by ours.

### 2. `github.com/philippgille/chromem-go@v0.7.0`

- **Status:** v0.x, single maintainer (`philippgille`), but actively developed
  with many releases through 2025.
- **Pulled in because:** embedded vector store used by graymatter.
- **Risk:** pre-1.0 API stability; MPL-2.0 obligation on any local modifications.
- **Decision:** _accept risk, version-pin via `go.sum`_. The module is
  pure-Go, has no cgo, and its blast radius is contained to graymatter's
  memory subsystem. If upstream stalls, we can vendor it (while respecting
  MPL-2.0).

### General mitigation for both

`go.sum` locks content hashes, so a compromised re-tag of an existing version
would be detected on `go mod download`. `GOPROXY=https://proxy.golang.org` (the
default) further guarantees tag immutability via the Go checksum database.

## Update policy

| Category | Cadence | Trigger |
|---|---|---|
| Go toolchain (`go` directive + `toolchain`) | Quarterly; always within 6 months of a stable Go release | New stdlib CVE in OSV report |
| Direct dependencies (cobra, testify, yaml.v3) | Monthly `go list -u -m all` review; upgrade on minor/patch | Any CVE matching a module we import |
| `anthropic-sdk-go` | Opportunistic — follow the SDK's release notes | Claude model / API change we want to consume |
| `graymatter`, `chromem-go` | Manual review before each bump; diff the release notes, re-run tests | Every tagged release |
| OSV + SBOM regeneration | On every PR that touches `go.mod`, and weekly in CI | Diff `sbom.cdx.json` in PR review |
| License re-scan | Same cadence as SBOM | New indirect dep appears |

### Suggested CI hook

```yaml
# .github/workflows/supply-chain.yml (sketch)
- run: go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
- run: go install github.com/google/osv-scanner/cmd/osv-scanner@latest
- run: cyclonedx-gomod mod -json -output sbom.cdx.json
- run: osv-scanner --lockfile=go.mod
- run: git diff --exit-code sbom.cdx.json   # fail PR if SBOM drifted un-committed
```

## Reproducing this audit

```bash
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
go install github.com/google/osv-scanner/cmd/osv-scanner@latest
go install github.com/google/go-licenses@latest

cd /path/to/llmwiki
cyclonedx-gomod mod -json -output sbom.cdx.json
osv-scanner --lockfile=go.mod
go-licenses report ./... 2>/dev/null
go mod graph
```
