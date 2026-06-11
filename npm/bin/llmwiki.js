#!/usr/bin/env node
"use strict";

// Thin launcher: ensure the platform binary is present (downloading it from
// the matching GitHub release on first run), then exec it with our argv and
// propagate the exit code. The heavy lifting lives in ../lib/install.js.

const { spawnSync } = require("node:child_process");
const { ensureBinary } = require("../lib/install");

async function main() {
  const bin = await ensureBinary();
  const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });

  if (result.error) {
    throw result.error;
  }
  if (typeof result.status === "number") {
    process.exit(result.status);
  }
  // Killed by a signal.
  process.exit(1);
}

main().catch((err) => {
  process.stderr.write(`llmwiki: ${err.message}\n`);
  process.exit(1);
});
