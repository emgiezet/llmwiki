# llmwiki gemini-cli wrapper (bash/zsh).
#
# Intercepts `gemini -p "..."` non-interactive invocations and forwards stdout
# to `llmwiki absorb` after success. Interactive TUI calls (no -p / --prompt)
# are passed through unchanged, because capturing ANSI-laced TUI output
# would produce garbage.
#
# Safety:
#   - `|| true` on the absorb call: llmwiki failures never break gemini.
#   - PIPESTATUS[0] preserves gemini's real exit code (not tee's).
#   - Empty stdout skips absorb.

gemini() {
    local has_p=0
    for arg in "$@"; do
        case "$arg" in -p|--prompt) has_p=1; break ;; esac
    done
    if [ "$has_p" -eq 0 ]; then
        command gemini "$@"
        return $?
    fi

    local tmp
    tmp=$(mktemp) || { command gemini "$@"; return $?; }
    command gemini "$@" | tee "$tmp"
    local rc=${PIPESTATUS[0]:-$?}
    if [ "$rc" -eq 0 ] && [ -s "$tmp" ]; then
        llmwiki absorb "$(pwd)" --note-stdin --fast-fail < "$tmp" 2>/dev/null || true
    fi
    rm -f "$tmp"
    return "$rc"
}
