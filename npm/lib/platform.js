"use strict";

// Pure, network-free helpers for resolving which release asset to download.
// Mirrors the naming used by install.sh and .goreleaser.yaml.

const REPO = "emgiezet/llmwiki";
const BIN_NAME = "llmwiki";

// Map Node's process.platform / process.arch to llmwiki's release os/arch.
function detectPlatform(nodePlatform, nodeArch) {
  let os;
  switch (nodePlatform) {
    case "darwin":
      os = "darwin";
      break;
    case "linux":
      os = "linux";
      break;
    default:
      throw new Error(
        `unsupported OS: ${nodePlatform}. llmwiki ships binaries for macOS and Linux only — ` +
          `on other systems use install.sh or the Docker image (ghcr.io/emgiezet/llmwiki).`,
      );
  }

  let arch;
  switch (nodeArch) {
    case "x64":
      arch = "amd64";
      break;
    case "arm64":
      arch = "arm64";
      break;
    default:
      throw new Error(
        `unsupported architecture: ${nodeArch} (need x64/amd64 or arm64).`,
      );
  }

  return { os, arch };
}

function stripV(version) {
  return version.startsWith("v") ? version.slice(1) : version;
}

function releaseTag(version) {
  return version.startsWith("v") ? version : `v${version}`;
}

// goreleaser's .Version is already v-stripped; install.sh uses ${version#v}.
function assetName(version, os, arch) {
  return `${BIN_NAME}_${stripV(version)}_${os}_${arch}.tar.gz`;
}

function downloadURLs(version, os, arch) {
  const base = `https://github.com/${REPO}/releases/download/${releaseTag(version)}`;
  return {
    archive: `${base}/${assetName(version, os, arch)}`,
    checksums: `${base}/checksums.txt`,
  };
}

module.exports = {
  REPO,
  BIN_NAME,
  detectPlatform,
  stripV,
  releaseTag,
  assetName,
  downloadURLs,
};
