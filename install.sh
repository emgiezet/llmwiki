#!/bin/sh
#
# llmwiki release installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh | sh
#
# Env overrides:
#   VERSION      - Pin to a specific release tag (e.g. v0.5.0). Default: latest.
#   INSTALL_DIR  - Where to install the binary. Default: $HOME/.local/bin.
#
# Exit codes are non-zero on any failure with a one-line reason printed to
# stderr. Downloads are checksum-verified against the release's checksums.txt.

set -eu

REPO="emgiezet/llmwiki"
BIN_NAME="llmwiki"

log() { printf '%s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }

# --- detect OS + arch -------------------------------------------------------

detect_os() {
    kernel="$(uname -s 2>/dev/null || echo unknown)"
    case "$kernel" in
        Darwin) echo darwin ;;
        Linux)  echo linux ;;
        *) die "unsupported OS: $kernel (need Darwin or Linux)" ;;
    esac
}

detect_arch() {
    machine="$(uname -m 2>/dev/null || echo unknown)"
    case "$machine" in
        x86_64|amd64) echo amd64 ;;
        arm64|aarch64) echo arm64 ;;
        *) die "unsupported architecture: $machine (need x86_64/amd64 or arm64/aarch64)" ;;
    esac
}

# --- resolve version --------------------------------------------------------

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        printf '%s\n' "$VERSION"
        return
    fi
    url="https://api.github.com/repos/${REPO}/releases/latest"
    tag="$(curl -fsSL "$url" 2>/dev/null | grep '"tag_name"' | head -n1 | cut -d'"' -f4 || true)"
    [ -n "$tag" ] || die "could not resolve latest release (check network access to GitHub)"
    printf '%s\n' "$tag"
}

# --- pick sha256 tool -------------------------------------------------------

sha256_tool() {
    if command -v sha256sum >/dev/null 2>&1; then
        echo "sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        echo "shasum -a 256"
    else
        die "neither sha256sum nor shasum available; cannot verify download"
    fi
}

# --- download + verify ------------------------------------------------------

download() {
    url="$1"
    out="$2"
    curl -fsSL --retry 3 --connect-timeout 10 "$url" -o "$out" \
        || die "download failed: $url"
}

verify_checksum() {
    archive="$1"
    checksums_file="$2"
    tool="$(sha256_tool)"
    expected="$(grep " $(basename "$archive")\$" "$checksums_file" | awk '{print $1}' || true)"
    [ -n "$expected" ] || die "no checksum entry for $(basename "$archive") in checksums.txt"
    actual="$($tool "$archive" | awk '{print $1}')"
    [ "$expected" = "$actual" ] \
        || die "checksum mismatch for $(basename "$archive"): got $actual, want $expected"
}

# --- install target ---------------------------------------------------------

install_dir() {
    if [ -n "${INSTALL_DIR:-}" ]; then
        printf '%s\n' "$INSTALL_DIR"
        return
    fi
    printf '%s\n' "${HOME}/.local/bin"
}

path_has() {
    dir="$1"
    case ":${PATH}:" in
        *":${dir}:"*) return 0 ;;
        *) return 1 ;;
    esac
}

shell_rc_hint() {
    dir="$1"
    shell_name="$(basename "${SHELL:-/bin/sh}")"
    case "$shell_name" in
        zsh)  rc="~/.zshrc" ;;
        bash) rc="~/.bashrc" ;;
        fish) rc="~/.config/fish/config.fish" ;;
        *)    rc="your shell's rc file" ;;
    esac
    cat >&2 <<EOF
note: $dir is not on your \$PATH.
      add it by appending this line to $rc:
        export PATH="$dir:\$PATH"
EOF
}

# --- main -------------------------------------------------------------------

os="$(detect_os)"
arch="$(detect_arch)"
version="$(resolve_version)"
dest="$(install_dir)"

archive="${BIN_NAME}_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${version}"

log "installing ${BIN_NAME} ${version} (${os}/${arch}) → ${dest}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

download "${base_url}/${archive}"        "${tmp}/${archive}"
download "${base_url}/checksums.txt"     "${tmp}/checksums.txt"
verify_checksum "${tmp}/${archive}" "${tmp}/checksums.txt"

tar -xzf "${tmp}/${archive}" -C "$tmp" || die "tar extraction failed"

[ -f "${tmp}/${BIN_NAME}" ] || die "archive did not contain expected binary: ${BIN_NAME}"

mkdir -p "$dest"
mv -f "${tmp}/${BIN_NAME}" "${dest}/${BIN_NAME}"
chmod 0755 "${dest}/${BIN_NAME}"

if ! path_has "$dest"; then
    shell_rc_hint "$dest"
fi

log "installed ${BIN_NAME} ${version} → ${dest}/${BIN_NAME}"
log "run '${BIN_NAME} --help' to get started."
