"use strict";

// Download + verify + extract the llmwiki release binary. The pure helpers
// (parseChecksums / expectedHash / sha256File / verifyChecksum) are unit-tested;
// ensureBinary() does the network + filesystem I/O and is covered by a smoke
// test. Dependency-free: only node:* builtins, and `tar` (present on macOS and
// Linux, the only platforms we ship binaries for) — same approach as install.sh.

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const crypto = require("node:crypto");
const { spawnSync } = require("node:child_process");

const platform = require("./platform");

// Parse a `checksums.txt` (sha256sum format: "<hex>  <filename>") into a Map.
function parseChecksums(text) {
  const map = new Map();
  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const parts = trimmed.split(/\s+/);
    if (parts.length < 2) continue;
    const [hash, name] = parts;
    map.set(name, hash);
  }
  return map;
}

function expectedHash(checksumsText, assetName) {
  const hash = parseChecksums(checksumsText).get(assetName);
  if (!hash) {
    throw new Error(`no checksum entry for ${assetName} in checksums.txt`);
  }
  return hash;
}

function sha256File(filePath) {
  const buf = fs.readFileSync(filePath);
  return crypto.createHash("sha256").update(buf).digest("hex");
}

function verifyChecksum(filePath, checksumsText, assetName) {
  const want = expectedHash(checksumsText, assetName);
  const got = sha256File(filePath);
  if (want !== got) {
    throw new Error(`checksum mismatch for ${assetName}: got ${got}, want ${want}`);
  }
  return true;
}

async function fetchBuffer(url) {
  // Node >=18 has global fetch, which follows GitHub's redirect to the CDN.
  const res = await fetch(url, { redirect: "follow" });
  if (!res.ok) {
    throw new Error(`download failed (${res.status} ${res.statusText}): ${url}`);
  }
  return Buffer.from(await res.arrayBuffer());
}

async function fetchText(url) {
  return (await fetchBuffer(url)).toString("utf8");
}

// Resolve the llmwiki binary, downloading it on first use. Returns its path.
// Honors $LLMWIKI_NPM_BINARY (an absolute path to an existing binary) so the
// launcher can be smoke-tested without hitting the network.
async function ensureBinary(opts = {}) {
  const override = opts.binaryOverride || process.env.LLMWIKI_NPM_BINARY;
  if (override) {
    if (!fs.existsSync(override)) {
      throw new Error(`LLMWIKI_NPM_BINARY points at a missing file: ${override}`);
    }
    return override;
  }

  const version = opts.version || require("../package.json").version;
  const { os: targetOS, arch } = platform.detectPlatform(
    opts.nodePlatform || process.platform,
    opts.nodeArch || process.arch,
  );

  const vendorDir = opts.vendorDir || path.join(__dirname, "..", "vendor");
  const binPath = path.join(vendorDir, platform.BIN_NAME);
  if (fs.existsSync(binPath)) {
    return binPath;
  }

  const urls = platform.downloadURLs(version, targetOS, arch);
  const asset = platform.assetName(version, targetOS, arch);

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "llmwiki-dl-"));
  try {
    const archivePath = path.join(tmp, asset);
    process.stderr.write(`llmwiki: downloading ${asset} (${version})\n`);
    fs.writeFileSync(archivePath, await fetchBuffer(urls.archive));

    const checksums = await fetchText(urls.checksums);
    verifyChecksum(archivePath, checksums, asset);

    const res = spawnSync("tar", ["-xzf", archivePath, "-C", tmp], {
      stdio: "inherit",
    });
    if (res.status !== 0) {
      throw new Error("tar extraction failed");
    }

    const extracted = path.join(tmp, platform.BIN_NAME);
    if (!fs.existsSync(extracted)) {
      throw new Error(`archive did not contain expected binary: ${platform.BIN_NAME}`);
    }

    fs.mkdirSync(vendorDir, { recursive: true });
    fs.copyFileSync(extracted, binPath);
    fs.chmodSync(binPath, 0o755);
    return binPath;
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

module.exports = {
  parseChecksums,
  expectedHash,
  sha256File,
  verifyChecksum,
  ensureBinary,
};
