#!/usr/bin/env python3
"""
llmwiki Claude Code Stop Hook
Reads Stop hook JSON from stdin, extracts the last analytical assistant
response, and pipes it to 'llmwiki absorb <cwd> --note-stdin'.
Always exits 0 — never blocks Claude.
"""
import json
import os
import os.path
import subprocess
import sys

MIN_RESPONSE_CHARS = 300
MAX_NOTE_CHARS = 2000
ANALYTICAL_TOOLS = {"Read", "Grep", "Glob", "Bash"}
RECENT_WINDOW = 20

ALLOWED_TRANSCRIPT_PREFIX = os.path.realpath(os.path.expanduser("~/.claude/projects"))


def _is_safe_transcript_path(path):
    try:
        real = os.path.realpath(path)
    except OSError:
        return False
    return real == ALLOWED_TRANSCRIPT_PREFIX or real.startswith(ALLOWED_TRANSCRIPT_PREFIX + os.sep)


def extract_last_response(transcript_path):
    try:
        with open(transcript_path, "r", encoding="utf-8") as f:
            lines = f.readlines()
    except OSError:
        return None, set()

    last_text = None
    recent_tools = set()
    window = lines[-RECENT_WINDOW:] if len(lines) > RECENT_WINDOW else lines

    for raw in window:
        raw = raw.strip()
        if not raw:
            continue
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            continue

        entry_type = msg.get("type", "")
        inner = msg.get("message")
        if not isinstance(inner, dict):
            inner = msg

        content = inner.get("content", "")
        is_assistant = entry_type == "assistant" or inner.get("role") == "assistant"

        if isinstance(content, list):
            for block in content:
                if not isinstance(block, dict):
                    continue
                block_type = block.get("type", "")
                if block_type == "text" and is_assistant:
                    last_text = block.get("text", last_text)
                elif block_type == "tool_use":
                    tname = block.get("name", "")
                    if tname in ANALYTICAL_TOOLS:
                        recent_tools.add(tname)
        elif isinstance(content, str) and is_assistant:
            last_text = content

    return last_text, recent_tools


def main():
    try:
        event = json.loads(sys.stdin.read())
        cwd = event.get("cwd", "")
        transcript_path = event.get("transcript_path", "")

        if not cwd or not transcript_path:
            sys.exit(0)

        if not _is_safe_transcript_path(transcript_path):
            sys.exit(0)

        last_text, recent_tools = extract_last_response(transcript_path)

        if not last_text or len(last_text) < MIN_RESPONSE_CHARS:
            sys.exit(0)

        if not recent_tools.intersection(ANALYTICAL_TOOLS):
            sys.exit(0)

        note = last_text[:MAX_NOTE_CHARS]
        subprocess.run(
            ["llmwiki", "absorb", cwd, "--note-stdin"],
            input=note,
            text=True,
            timeout=30,
            capture_output=True,
        )
    except Exception as exc:
        try:
            log_path = os.path.join(os.path.expanduser("~"), ".llmwiki", "hook.log")
            os.makedirs(os.path.dirname(log_path), exist_ok=True)
            with open(log_path, "a", encoding="utf-8") as logf:
                logf.write(f"{os.path.basename(__file__)}: {type(exc).__name__}: {exc}\n")
        except Exception:
            pass
    sys.exit(0)


if __name__ == "__main__":
    main()
