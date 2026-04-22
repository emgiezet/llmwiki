#!/usr/bin/env python3
"""Sanity test for the embedded Stop hook script in internal/cmd/hook.go.

Not part of go test ./... — run standalone with `python3 scripts/test_stop_hook.py`.
Exits 0 if all assertions pass, 1 otherwise.
"""
import json
import os
import re
import subprocess
import sys
import tempfile

SCRIPT_CONST_RE = re.compile(r'const\s+stopHookScript\s*=\s*`(.*?)`', re.DOTALL)


def load_embedded_script():
    src = open(os.path.join(os.path.dirname(__file__), "..", "internal", "cmd", "hook.go")).read()
    m = SCRIPT_CONST_RE.search(src)
    if not m:
        raise RuntimeError("could not find embedded Python script")
    return m.group(1)


def run_hook(script_path, event):
    return subprocess.run(
        ["python3", script_path],
        input=json.dumps(event),
        text=True,
        capture_output=True,
        timeout=10,
    )


def main():
    script = load_embedded_script()
    with tempfile.NamedTemporaryFile("w", suffix=".py", delete=False) as f:
        f.write(script)
        script_path = f.name

    failures = []

    # Case 1: forged path to /etc/passwd — hook must exit 0 without reading.
    result = run_hook(script_path, {"cwd": "/tmp", "transcript_path": "/etc/passwd"})
    if result.returncode != 0:
        failures.append(f"forged /etc/passwd case exited {result.returncode}, expected 0")

    # Case 2: empty event — hook must exit 0.
    result = run_hook(script_path, {})
    if result.returncode != 0:
        failures.append(f"empty event exited {result.returncode}, expected 0")

    # Case 3: malformed JSON — hook must exit 0 (wrapped in try/except).
    result = subprocess.run(
        ["python3", script_path],
        input="{not json",
        text=True,
        capture_output=True,
        timeout=10,
    )
    if result.returncode != 0:
        failures.append(f"malformed JSON exited {result.returncode}, expected 0")

    os.unlink(script_path)

    if failures:
        for f in failures:
            print(f"FAIL: {f}", file=sys.stderr)
        sys.exit(1)
    print("all transcript-path hardening cases pass")


if __name__ == "__main__":
    main()
