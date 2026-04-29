# llmwiki gemini-cli wrapper (fish).
#
# Same intent as the bash/zsh version: intercept `gemini -p ...` only,
# forward stdout to `llmwiki absorb`, never break gemini itself.

function gemini --description "llmwiki: capture non-interactive gemini output into memory"
    set -l has_p 0
    for arg in $argv
        if test "$arg" = "-p" -o "$arg" = "--prompt"
            set has_p 1
            break
        end
    end
    if test $has_p -eq 0
        command gemini $argv
        return $status
    end

    set -l tmp (mktemp); or begin
        command gemini $argv
        return $status
    end
    command gemini $argv | tee $tmp
    set -l rc $pipestatus[1]
    if test $rc -eq 0 -a -s $tmp
        llmwiki absorb (pwd) --note-stdin --fast-fail < $tmp 2>/dev/null; or true
    end
    rm -f $tmp
    return $rc
end
