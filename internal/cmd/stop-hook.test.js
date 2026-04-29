#!/usr/bin/env node
// Node-side integration test for stop-hook.js.
//
// Spawns the hook as a subprocess with a synthesized Claude Code transcript
// on stdin and a fake `llmwiki` binary on PATH. Confirms that a qualifying
// transcript (long assistant text + Read tool_use in the window) triggers
// exactly one `llmwiki absorb` call with the expected stdin body.
//
// Run via: node plugin/hooks/stop-hook.test.js
// Exits 0 on pass, non-zero on fail. No test framework dependency.

"use strict";

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const { spawnSync } = require("node:child_process");

const HOOK = path.resolve(__dirname, "stop-hook.js");
const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "stop-hook-test-"));

// --- Build the fake `llmwiki` binary: a shell script that records its
// argv + stdin to a file so we can assert on them after the hook runs.
const FAKE_BIN_DIR = path.join(tmp, "bin");
const CAPTURE_FILE = path.join(tmp, "captured.json");
fs.mkdirSync(FAKE_BIN_DIR, { recursive: true });
const fakeBin = path.join(FAKE_BIN_DIR, "llmwiki");
fs.writeFileSync(
    fakeBin,
    `#!/bin/sh
# Records the invocation so the test can inspect argv + stdin.
note="$(cat)"
# Use python-style JSON escaping via sed since we can't depend on jq.
printf '{"args":["%s","%s","%s","%s","%s"],"note":%s}' \\
  "$0" "$1" "$2" "$3" "$4" \\
  "$(printf '%s' "$note" | awk 'BEGIN{printf "\\""}{gsub(/\\\\/,"\\\\\\\\");gsub(/"/,"\\\\\\"");printf "%s\\\\n",$0}END{printf "\\""}')" \\
  > "${CAPTURE_FILE}"
exit 0
`,
);
fs.chmodSync(fakeBin, 0o755);

// --- Build a fake ~/.claude/projects/... transcript path that passes the
// security gate (the hook only accepts transcript paths under that prefix).
const fakeHome = path.join(tmp, "home");
const projectsDir = path.join(fakeHome, ".claude", "projects", "proj1");
fs.mkdirSync(projectsDir, { recursive: true });
const transcriptPath = path.join(projectsDir, "session.jsonl");

// Transcript entries: an analytical tool_use (Read) in the window, followed
// by a qualifying assistant text block > 300 chars.
const longText = "x".repeat(400);
const lines = [
    JSON.stringify({
        type: "assistant",
        message: {
            role: "assistant",
            content: [{ type: "tool_use", name: "Read", input: {} }],
        },
    }),
    JSON.stringify({
        type: "assistant",
        message: {
            role: "assistant",
            content: [{ type: "text", text: longText }],
        },
    }),
];
fs.writeFileSync(transcriptPath, lines.join("\n") + "\n");

// --- Run the hook with HOME overridden so it resolves the fake transcript
// prefix, PATH rewritten so `llmwiki` points at our capture script.
const payload = JSON.stringify({
    cwd: "/some/project",
    transcript_path: transcriptPath,
});
const result = spawnSync(process.execPath, [HOOK], {
    input: payload,
    env: {
        ...process.env,
        HOME: fakeHome,
        PATH: `${FAKE_BIN_DIR}:${process.env.PATH || ""}`,
        CAPTURE_FILE,
    },
    encoding: "utf8",
    timeout: 10_000,
});

// --- Assertions.
let failed = 0;
function check(cond, msg) {
    if (!cond) {
        console.error(`FAIL: ${msg}`);
        failed++;
    } else {
        console.log(`ok: ${msg}`);
    }
}

check(result.status === 0, `hook exits 0 (got ${result.status}, stderr: ${result.stderr})`);
check(fs.existsSync(CAPTURE_FILE), "llmwiki was invoked (capture file exists)");

if (fs.existsSync(CAPTURE_FILE)) {
    const captured = JSON.parse(fs.readFileSync(CAPTURE_FILE, "utf8"));
    check(
        captured.args[1] === "absorb",
        `first arg is 'absorb' (got ${captured.args[1]})`,
    );
    check(
        captured.args[2] === "/some/project",
        `cwd passed through (got ${captured.args[2]})`,
    );
    check(
        captured.args[3] === "--note-stdin",
        `--note-stdin flag present (got ${captured.args[3]})`,
    );
    check(
        captured.args[4] === "--fast-fail",
        `--fast-fail flag present (got ${captured.args[4]})`,
    );
    check(
        captured.note.includes(longText),
        "captured note contains the long assistant text",
    );
    check(
        captured.note.length <= 2000 + 5, // + slack for JSON quoting
        `note was truncated to <= 2000 chars (got ${captured.note.length})`,
    );
}

// --- Negative test: short assistant text should NOT trigger absorb.
fs.writeFileSync(
    transcriptPath,
    JSON.stringify({
        type: "assistant",
        message: {
            role: "assistant",
            content: [
                { type: "tool_use", name: "Read", input: {} },
                { type: "text", text: "too short" },
            ],
        },
    }) + "\n",
);
fs.rmSync(CAPTURE_FILE, { force: true });
spawnSync(process.execPath, [HOOK], {
    input: payload,
    env: {
        ...process.env,
        HOME: fakeHome,
        PATH: `${FAKE_BIN_DIR}:${process.env.PATH || ""}`,
        CAPTURE_FILE,
    },
    encoding: "utf8",
    timeout: 10_000,
});
check(
    !fs.existsSync(CAPTURE_FILE),
    "short assistant text does NOT trigger llmwiki absorb",
);

// --- Negative test: missing analytical tool_use should NOT trigger absorb.
fs.writeFileSync(
    transcriptPath,
    JSON.stringify({
        type: "assistant",
        message: {
            role: "assistant",
            content: [{ type: "text", text: longText }],
        },
    }) + "\n",
);
fs.rmSync(CAPTURE_FILE, { force: true });
spawnSync(process.execPath, [HOOK], {
    input: payload,
    env: {
        ...process.env,
        HOME: fakeHome,
        PATH: `${FAKE_BIN_DIR}:${process.env.PATH || ""}`,
        CAPTURE_FILE,
    },
    encoding: "utf8",
    timeout: 10_000,
});
check(
    !fs.existsSync(CAPTURE_FILE),
    "long text without analytical tool_use does NOT trigger llmwiki absorb",
);

// --- Negative test: unsafe transcript path (outside ~/.claude/projects)
// should exit 0 without calling absorb.
const unsafeTranscript = path.join(tmp, "evil.jsonl");
fs.writeFileSync(unsafeTranscript, lines.join("\n") + "\n");
fs.rmSync(CAPTURE_FILE, { force: true });
const unsafeResult = spawnSync(process.execPath, [HOOK], {
    input: JSON.stringify({ cwd: "/some/project", transcript_path: unsafeTranscript }),
    env: {
        ...process.env,
        HOME: fakeHome,
        PATH: `${FAKE_BIN_DIR}:${process.env.PATH || ""}`,
        CAPTURE_FILE,
    },
    encoding: "utf8",
    timeout: 10_000,
});
check(
    unsafeResult.status === 0 && !fs.existsSync(CAPTURE_FILE),
    "transcript path outside ~/.claude/projects is rejected silently",
);

fs.rmSync(tmp, { recursive: true, force: true });
if (failed > 0) {
    console.error(`\n${failed} assertion(s) failed`);
    process.exit(1);
}
console.log(`\nall assertions passed`);
