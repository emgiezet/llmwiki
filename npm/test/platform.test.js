"use strict";

const test = require("node:test");
const assert = require("node:assert");
const platform = require("../lib/platform");

test("detectPlatform maps supported os/arch", () => {
  assert.deepStrictEqual(platform.detectPlatform("linux", "x64"), {
    os: "linux",
    arch: "amd64",
  });
  assert.deepStrictEqual(platform.detectPlatform("linux", "arm64"), {
    os: "linux",
    arch: "arm64",
  });
  assert.deepStrictEqual(platform.detectPlatform("darwin", "arm64"), {
    os: "darwin",
    arch: "arm64",
  });
  assert.deepStrictEqual(platform.detectPlatform("darwin", "x64"), {
    os: "darwin",
    arch: "amd64",
  });
});

test("detectPlatform rejects unsupported OS with a helpful message", () => {
  assert.throws(() => platform.detectPlatform("win32", "x64"), /install\.sh|Docker/i);
});

test("detectPlatform rejects unsupported arch", () => {
  assert.throws(() => platform.detectPlatform("linux", "ia32"), /architecture/i);
});

test("stripV / releaseTag normalize the leading v", () => {
  assert.strictEqual(platform.stripV("v2.6.0"), "2.6.0");
  assert.strictEqual(platform.stripV("2.6.0"), "2.6.0");
  assert.strictEqual(platform.releaseTag("2.6.0"), "v2.6.0");
  assert.strictEqual(platform.releaseTag("v2.6.0"), "v2.6.0");
});

test("assetName matches the install.sh / goreleaser naming (no leading v)", () => {
  // install.sh:129 uses ${version#v}; goreleaser .Version is already v-stripped.
  assert.strictEqual(
    platform.assetName("v2.6.0", "linux", "amd64"),
    "llmwiki_2.6.0_linux_amd64.tar.gz",
  );
  assert.strictEqual(
    platform.assetName("2.6.0", "darwin", "arm64"),
    "llmwiki_2.6.0_darwin_arm64.tar.gz",
  );
});

test("downloadURLs point at the tagged release assets", () => {
  const urls = platform.downloadURLs("2.6.0", "linux", "amd64");
  assert.strictEqual(
    urls.archive,
    "https://github.com/emgiezet/llmwiki/releases/download/v2.6.0/llmwiki_2.6.0_linux_amd64.tar.gz",
  );
  assert.strictEqual(
    urls.checksums,
    "https://github.com/emgiezet/llmwiki/releases/download/v2.6.0/checksums.txt",
  );
});
