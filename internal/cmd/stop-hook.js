#!/usr/bin/env node
// llmwiki Claude Code Stop Hook (Node.js rewrite of stop-hook.py).
//
// Reads Stop hook JSON from stdin, extracts the last analytical assistant
// response, and pipes it to 'llmwiki absorb <cwd> --note-stdin'.
// Always exits 0 — never blocks Claude.
//
// Only Node stdlib is used (fs, path, os, child_process). Requires Node ≥ 18.

"use strict";

const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const { spawnSync } = require("node:child_process");

const MIN_RESPONSE_CHARS = 300;
const MAX_NOTE_CHARS = 2000;
const ANALYTICAL_TOOLS = new Set(["Read", "Grep", "Glob", "Bash"]);
const RECENT_WINDOW = 20;

const ALLOWED_TRANSCRIPT_PREFIX = fs.realpathSync(
    path.resolve(os.homedir(), ".claude/projects"),
);

function isSafeTranscriptPath(p) {
    try {
        const real = fs.realpathSync(p);
        return (
            real === ALLOWED_TRANSCRIPT_PREFIX ||
            real.startsWith(ALLOWED_TRANSCRIPT_PREFIX + path.sep)
        );
    } catch {
        return false;
    }
}

function readStdin() {
    // Sync stdin read — hook payload is small (a few KB).
    try {
        return fs.readFileSync(0, "utf8");
    } catch {
        return "";
    }
}

function extractLastResponse(transcriptPath) {
    let data;
    try {
        data = fs.readFileSync(transcriptPath, "utf8");
    } catch {
        return { lastText: null, recentTools: new Set() };
    }
    const lines = data.split("\n");
    const window =
        lines.length > RECENT_WINDOW
            ? lines.slice(-RECENT_WINDOW)
            : lines;

    let lastText = null;
    const recentTools = new Set();

    for (const raw of window) {
        const trimmed = raw.trim();
        if (!trimmed) continue;
        let msg;
        try {
            msg = JSON.parse(trimmed);
        } catch {
            continue;
        }
        const entryType = msg.type || "";
        const inner =
            msg.message && typeof msg.message === "object"
                ? msg.message
                : msg;
        const content = inner.content || "";
        const isAssistant =
            entryType === "assistant" || inner.role === "assistant";

        if (Array.isArray(content)) {
            for (const block of content) {
                if (!block || typeof block !== "object") continue;
                const blockType = block.type || "";
                if (blockType === "text" && isAssistant) {
                    lastText = block.text || lastText;
                } else if (blockType === "tool_use") {
                    const name = block.name || "";
                    if (ANALYTICAL_TOOLS.has(name)) recentTools.add(name);
                }
            }
        } else if (typeof content === "string" && isAssistant) {
            lastText = content;
        }
    }

    return { lastText, recentTools };
}

function logError(msg) {
    try {
        const logPath = path.join(os.homedir(), ".llmwiki", "hook.log");
        fs.mkdirSync(path.dirname(logPath), { recursive: true });
        fs.appendFileSync(logPath, `${path.basename(__filename)}: ${msg}\n`);
    } catch {
        // Logging is best-effort; never let it propagate.
    }
}

function hasAnalyticalIntersection(tools) {
    for (const t of tools) if (ANALYTICAL_TOOLS.has(t)) return true;
    return false;
}

function main() {
    let event;
    try {
        event = JSON.parse(readStdin());
    } catch (exc) {
        logError(`${exc && exc.name ? exc.name : "ParseError"}: ${exc}`);
        process.exit(0);
    }

    const cwd = event.cwd || "";
    const transcriptPath = event.transcript_path || "";
    if (!cwd || !transcriptPath) process.exit(0);

    if (!isSafeTranscriptPath(transcriptPath)) process.exit(0);

    const { lastText, recentTools } = extractLastResponse(transcriptPath);
    if (!lastText || lastText.length < MIN_RESPONSE_CHARS) process.exit(0);
    if (!hasAnalyticalIntersection(recentTools)) process.exit(0);

    const note = lastText.slice(0, MAX_NOTE_CHARS);

    try {
        const result = spawnSync(
            "llmwiki",
            ["absorb", cwd, "--note-stdin", "--fast-fail"],
            {
                input: note,
                encoding: "utf8",
                timeout: 30_000,
            },
        );
        if (
            result.stderr &&
            typeof result.stderr === "string" &&
            result.stderr.includes("memory db busy")
        ) {
            logError(result.stderr);
        }
    } catch (exc) {
        // Swallow — hook must never fail Claude's run.
        void exc;
    }

    process.exit(0);
}

try {
    main();
} catch (exc) {
    logError(
        `${exc && exc.name ? exc.name : "Error"}: ${exc && exc.message ? exc.message : exc}`,
    );
    process.exit(0);
}
