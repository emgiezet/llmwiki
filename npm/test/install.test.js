"use strict";

const test = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const crypto = require("node:crypto");
const install = require("../lib/install");

function tmpFile(contents) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "llmwiki-npm-test-"));
  const file = path.join(dir, "llmwiki_1.0.0_linux_amd64.tar.gz");
  fs.writeFileSync(file, contents);
  return file;
}

test("parseChecksums maps filename -> hash for 'hash  name' lines", () => {
  const text = [
    "abc123  llmwiki_1.0.0_linux_amd64.tar.gz",
    "def456  llmwiki_1.0.0_darwin_arm64.tar.gz",
    "", // trailing blank line tolerated
  ].join("\n");
  const map = install.parseChecksums(text);
  assert.strictEqual(map.get("llmwiki_1.0.0_linux_amd64.tar.gz"), "abc123");
  assert.strictEqual(map.get("llmwiki_1.0.0_darwin_arm64.tar.gz"), "def456");
});

test("expectedHash returns the hash for a known asset", () => {
  const text = "abc123  llmwiki_1.0.0_linux_amd64.tar.gz\n";
  assert.strictEqual(
    install.expectedHash(text, "llmwiki_1.0.0_linux_amd64.tar.gz"),
    "abc123",
  );
});

test("expectedHash throws when the asset is absent", () => {
  const text = "abc123  some_other_file.tar.gz\n";
  assert.throws(
    () => install.expectedHash(text, "llmwiki_1.0.0_linux_amd64.tar.gz"),
    /no checksum entry/i,
  );
});

test("sha256File computes the hex digest of a file", () => {
  const file = tmpFile("hello llmwiki");
  const expected = crypto.createHash("sha256").update("hello llmwiki").digest("hex");
  assert.strictEqual(install.sha256File(file), expected);
});

test("verifyChecksum passes for a matching checksum", () => {
  const contents = "release-binary-bytes";
  const file = tmpFile(contents);
  const hash = crypto.createHash("sha256").update(contents).digest("hex");
  const text = `${hash}  ${path.basename(file)}\n`;
  assert.strictEqual(install.verifyChecksum(file, text, path.basename(file)), true);
});

test("verifyChecksum throws on a tampered file", () => {
  const file = tmpFile("real-bytes");
  const wrong = crypto.createHash("sha256").update("different-bytes").digest("hex");
  const text = `${wrong}  ${path.basename(file)}\n`;
  assert.throws(
    () => install.verifyChecksum(file, text, path.basename(file)),
    /checksum mismatch/i,
  );
});
