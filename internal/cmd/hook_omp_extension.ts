import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

// Payloads are read defensively through `unknown` so a shape change in a
// future omp degrades to "nothing captured" instead of a thrown handler.
function assistantText(event: unknown): string {
	if (!event || typeof event !== "object" || !("message" in event)) return "";
	const message = event.message;
	if (!message || typeof message !== "object" || !("content" in message)) return "";
	const blocks = message.content;
	if (!Array.isArray(blocks)) return "";
	const parts: string[] = [];
	for (const block of blocks as unknown[]) {
		if (block && typeof block === "object" && "type" in block && block.type === "text" && "text" in block) {
			if (typeof block.text === "string") parts.push(block.text);
		}
	}
	return parts.join("\n").trim();
}

function hostCwd(ctx: unknown): string {
	if (ctx && typeof ctx === "object" && "cwd" in ctx && typeof ctx.cwd === "string") return ctx.cwd;
	return "";
}

export default function llmwiki(pi: ExtensionAPI) {
	pi.setLabel("llmwiki");

	// Buffer the latest finalized assistant text; `assistant_message` fires
	// per message (including the ones before tool dispatch), so flushing
	// there would call absorb several times per prompt.
	let last = "";

	pi.on("assistant_message", (event: unknown) => {
		const text = assistantText(event);
		if (text) last = text;
		// Returning a value here would rewrite history; stay notification-only.
	});

	pi.on("agent_end", async (_event: unknown, ctx: unknown) => {
		const note = last.slice(0, 2000);
		last = "";
		if (!note) return;
		const cwd = hostCwd(ctx);
		if (!cwd) return;
		try {
			await pi.exec("llmwiki", ["absorb", cwd, "--note", note, "--fast-fail"], { cwd });
		} catch {
			// Best-effort capture; the extension must never break the session.
		}
	});
}
