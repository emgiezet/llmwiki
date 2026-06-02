# Installation

How to install, update, and understand llmwiki releases.

[← Back to README](../README.md)

## Install

Builds are published for macOS (arm64, amd64) and Linux (amd64, arm64).

### Quick download (pre-built binary)

Pick your platform and grab the tarball straight from the [latest release](https://github.com/emgiezet/llmwiki/releases/latest):

| Platform        | Asset                                            |
|-----------------|--------------------------------------------------|
| macOS Apple Silicon | `llmwiki_<version>_darwin_arm64.tar.gz`     |
| macOS Intel     | `llmwiki_<version>_darwin_amd64.tar.gz`          |
| Linux x86_64    | `llmwiki_<version>_linux_amd64.tar.gz`           |
| Linux arm64     | `llmwiki_<version>_linux_arm64.tar.gz`           |

With the [GitHub CLI](https://cli.github.com/) — downloads + extracts in one step:

```bash
gh release download --repo emgiezet/llmwiki --pattern '*darwin_arm64.tar.gz' -O - | tar -xz
sudo mv llmwiki /usr/local/bin/        # or: mv llmwiki ~/.local/bin/
```

Each release also ships `checksums.txt` (SHA256). The one-liner installer below verifies it for you.

### One-liner installer (recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh | sh
```
Installs the latest release to `~/.local/bin/llmwiki`, auto-detects OS/arch, and verifies the SHA256 checksum. If that directory isn't on your `$PATH`, the installer prints the exact `export PATH=…` line to add.

**Pinned version**
```bash
curl -fsSL https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh | VERSION=v0.5.0 sh
```

**Custom install directory**
```bash
curl -fsSL https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh | INSTALL_DIR=/usr/local/bin sh
```

**Manual download** — grab `llmwiki_<version>_<os>_<arch>.tar.gz` from the [Releases page](https://github.com/emgiezet/llmwiki/releases), verify the SHA256 against `checksums.txt`, extract, and move `llmwiki` into your `$PATH`.

**Go users**
```bash
go install github.com/emgiezet/llmwiki@latest
```

### Updating

`llmwiki` checks GitHub once every 24 hours (cached, non-blocking) and prints a one-line notice on stderr when a newer release exists:
```
llmwiki 0.5.1 available (you have 0.5.0) — run 'llmwiki update' to install.
```

Run `llmwiki update` to upgrade. The subcommand detects how the binary was installed and re-runs the installer script for release binaries, or `go install` for Go-managed installs. Supports `--version vX.Y.Z` to pin and `--dry-run` to preview.

The notice is suppressed in CI, non-TTY output, dev builds, and when `LLMWIKI_NO_UPDATE_CHECK=1` is set.

## Releases

Releases are cut automatically from commit history on `main`:

- **`feat:`** — new feature → minor bump (`0.4.0 → 0.5.0`)
- **`fix:`** — bug fix → patch bump (`0.5.0 → 0.5.1`)
- **`feat!:`** or `BREAKING CHANGE:` in the body → major bump (`0.5.0 → 1.0.0`)
- Anything else (`docs:`, `chore:`, `refactor:`, …) ships silently with the next tagged release.

[release-please](https://github.com/googleapis/release-please) maintains a running "Release PR" that accumulates unreleased commits. Merging that PR creates the git tag and GitHub Release; the release workflow then builds binaries for all four target platforms and attaches them along with `checksums.txt` and `install.sh`.

See all releases: [github.com/emgiezet/llmwiki/releases](https://github.com/emgiezet/llmwiki/releases).
