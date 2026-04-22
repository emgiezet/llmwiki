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


def run_hook(script_path, event, env=None):
    return subprocess.run(
        ["python3", script_path],
        input=json.dumps(event),
        text=True,
        capture_output=True,
        timeout=10,
        env=env,
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
    # With D14 applied, the exception is also logged to ~/.llmwiki/hook.log.
    result = subprocess.run(
        ["python3", script_path],
        input="{not json",
        text=True,
        capture_output=True,
        timeout=10,
    )
    if result.returncode != 0:
        failures.append(f"malformed JSON exited {result.returncode}, expected 0")

    # Case 4 (D14): malformed JSON triggers exception logging.
    # Run the hook with HOME pointed at a temp dir so we can inspect the log.
    with tempfile.TemporaryDirectory() as tmpHome:
        env = os.environ.copy()
        env["HOME"] = tmpHome
        result = subprocess.run(
            ["python3", script_path],
            input="{not json",
            text=True,
            capture_output=True,
            timeout=10,
            env=env,
        )
        if result.returncode != 0:
            failures.append(f"D14 logging case exited {result.returncode}, expected 0")
        else:
            log_path = os.path.join(tmpHome, ".llmwiki", "hook.log")
            if not os.path.exists(log_path):
                failures.append("D14: hook.log not created after JSON parse exception")
            else:
                log_content = open(log_path).read()
                if "JSONDecodeError" not in log_content:
                    failures.append(f"D14: hook.log missing JSONDecodeError, got: {log_content!r}")

    # Case 5: verify the hook still loads and parses correctly after the
    # --fast-fail change.  We can't run a real llmwiki binary in this harness,
    # but we can confirm the script syntax is valid and that the busy-warning
    # branch string is present in the embedded script.
    if "--fast-fail" not in script:
        failures.append("Case 5: '--fast-fail' flag missing from embedded hook script")
    if "memory db busy" not in script:
        failures.append("Case 5: 'memory db busy' string missing from embedded hook script")

    os.unlink(script_path)

    if failures:
        for f in failures:
            print(f"FAIL: {f}", file=sys.stderr)
        sys.exit(1)
    print("all transcript-path hardening cases pass (including D14 exception logging and fast-fail flag)")


if __name__ == "__main__":
    main()
