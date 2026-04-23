// llmwiki pi extension — subscribes to pi's `agent_end` lifecycle event
// (fires once per user prompt, after all tool calls complete) and forwards
// the last assistant message to `llmwiki absorb`.
//
// pi auto-loads .ts files from ~/.pi/agent/extensions/ (global) or
// .pi/extensions/ (project). The `pi.on()` registration API mirrors the
// @mariozechner/pi-coding-agent docs.
//
// The `pi` object is provided by the extension runtime at load time; we
// declare it loosely here so the file type-checks in isolation.

declare const pi: {
    on: (
        event: string,
        handler: (event: any, ctx: any) => Promise<void> | void,
    ) => void;
};

pi.on("agent_end", async (event: any, ctx: any) => {
    try {
        const text: string =
            (event && typeof event.last_assistant_message === "string"
                ? event.last_assistant_message
                : ctx && typeof ctx.last_assistant_message === "string"
                  ? ctx.last_assistant_message
                  : "") || "";
        if (!text) return;

        const cwd: string =
            (ctx && typeof ctx.cwd === "string"
                ? ctx.cwd
                : process && typeof process.cwd === "function"
                  ? process.cwd()
                  : "") || "";
        if (!cwd) return;

        const note = text.slice(0, 2000);

        // Use Node stdlib child_process — available because pi runs on Node.
        const { spawnSync } = require("node:child_process");
        spawnSync(
            "llmwiki",
            ["absorb", cwd, "--note-stdin", "--fast-fail"],
            {
                input: note,
                encoding: "utf8",
                timeout: 30_000,
                stdio: ["pipe", "ignore", "ignore"],
            },
        );
    } catch {
        // Best-effort capture; the extension must never break pi.
    }
});
