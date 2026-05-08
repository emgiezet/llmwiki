#!/bin/bash
# Passive graymatter memory capture on Claude Code Stop event.
# Reads transcript from stdin JSON, extracts last exchanges, stores in graymatter.

GRAYMATTER_DIR="${GRAYMATTER_HOOK_DIR:-.graymatter}"
AGENT_ID="claude-code"

INPUT=$(cat)
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript_path // ""')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"' | cut -c1-8)

[ -z "$TRANSCRIPT" ] || [ ! -f "$TRANSCRIPT" ] && exit 0

SUMMARY=$(python3 - "$TRANSCRIPT" <<'PYEOF'
import sys, json

transcript = sys.argv[1]
messages = []

try:
    with open(transcript) as f:
        for line in f:
            try:
                d = json.loads(line.strip())
                t = d.get('type', '')
                if t not in ('user', 'assistant'):
                    continue
                msg = d.get('message', {})
                role = msg.get('role', '')
                content = msg.get('content', '')
                text = ''
                if isinstance(content, list):
                    for c in content:
                        if isinstance(c, dict) and c.get('type') == 'text':
                            text = c['text'].strip()
                            break
                elif isinstance(content, str):
                    text = content.strip()
                # skip XML tags (system messages, commands)
                if text and not text.startswith('<') and len(text) > 10:
                    messages.append(f"[{role}]: {text[:300]}")
            except Exception:
                pass
except Exception:
    pass

# Last 6 exchanges
print('\n'.join(messages[-6:]))
PYEOF
)

[ -z "$SUMMARY" ] && exit 0

graymatter remember "$AGENT_ID" "session=$SESSION_ID
$SUMMARY" --dir "$GRAYMATTER_DIR" --quiet 2>/dev/null

exit 0
