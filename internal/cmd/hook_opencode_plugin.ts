// llmwiki opencode plugin — captures session-idle events and forwards the
// assistant output to `llmwiki absorb` so the insight lands in graymatter
// memory.
//
// Listens for `event.type === "session.idle"` (opencode's "session
// completed" signal per the @opencode-ai/plugin docs). Opencode plugins run
// in a TS-aware runtime so no compile step is needed.
//
// Imports are kept type-only so this file is valid as plain TS even when
// the @opencode-ai/plugin package isn't installed locally (opencode resolves
// it at load time).

import type { Plugin } from "@opencode-ai/plugin";

const LlmwikiPlugin: Plugin = async ({ client, $, directory }) => {
    return {
        event: async ({ event }) => {
            if (event.type !== "session.idle") return;

            try {
                // Grab the last assistant message for this session via the
                // opencode SDK. The exact messages shape is SDK-versioned;
                // we reach for the last element and its text body.
                const sessionID: string | undefined = (event as any)
                    .properties?.sessionID;
                if (!sessionID) return;

                const messages = await client.session.messages({
                    path: { id: sessionID },
                });
                const list = (messages as any)?.data ?? [];
                if (!Array.isArray(list) || list.length === 0) return;

                const last = list[list.length - 1];
                // Messages can carry text in either `content` (string) or
                // `parts` (array of typed blocks). Accept both.
                let text = "";
                if (typeof last?.content === "string") {
                    text = last.content;
                } else if (Array.isArray(last?.parts)) {
                    for (const p of last.parts) {
                        if (p?.type === "text" && typeof p.text === "string") {
                            text += p.text;
                        }
                    }
                }
                text = text.slice(0, 2000);
                if (!text) return;

                // Bun's $ is a template-tag shell; .quiet() suppresses
                // stdout in opencode's UI. Absorb errors are swallowed —
                // the plugin must never interrupt opencode's flow.
                await $`sh -c 'echo "$0" | llmwiki absorb "$1" --note-stdin --fast-fail 2>/dev/null || true' ${text} ${directory}`.quiet();
            } catch {
                // Best-effort capture only.
            }
        },
    };
};

export default LlmwikiPlugin;
