#!/usr/bin/env node
// llmwiki codex notify hook.
//
// Codex invokes `notify = ["node", "<this-file>"]` once per agent turn,
// appending its JSON payload as the final argv entry. The payload contains
// `last-assistant-message` and `cwd`; we pipe the message into
// `llmwiki absorb "$cwd" --note-stdin --fast-fail` so the insight lands in
// graymatter memory. Always exits 0 — never blocks codex.

"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const MAX_NOTE_CHARS = 2000;

function logError(msg) {
    try {
        const logPath = path.join(os.homedir(), ".llmwiki", "hook.log");
        fs.mkdirSync(path.dirname(logPath), { recursive: true });
        fs.appendFileSync(logPath, `codex-absorb: ${msg}\n`);
    } catch {
        // Logging is best-effort.
    }
}

function main() {
    // Payload is the final positional arg. argv = [node, script, ...(optional static args), <payload>]
    const raw = process.argv[process.argv.length - 1];
    if (!raw) process.exit(0);

    let payload;
    try {
        payload = JSON.parse(raw);
    } catch (err) {
        logError(`parse payload: ${err && err.message ? err.message : err}`);
        process.exit(0);
    }

    const msg = payload && payload["last-assistant-message"];
    const cwd = payload && payload.cwd;
    if (!msg || !cwd) process.exit(0);

    const note = typeof msg === "string" ? msg.slice(0, MAX_NOTE_CHARS) : "";
    if (!note) process.exit(0);

    try {
        const result = spawnSync(
            "llmwiki",
            ["absorb", cwd, "--note-stdin", "--fast-fail"],
            { input: note, encoding: "utf8", timeout: 30_000 },
        );
        if (
            result.stderr &&
            typeof result.stderr === "string" &&
            result.stderr.includes("memory db busy")
        ) {
            logError(result.stderr);
        }
    } catch (err) {
        logError(err && err.message ? err.message : String(err));
    }

    process.exit(0);
}

try {
    main();
} catch (err) {
    logError(err && err.message ? err.message : String(err));
    process.exit(0);
}
