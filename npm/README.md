# llmwiki (npm wrapper)

Run [llmwiki](https://github.com/emgiezet/llmwiki) without installing anything:

```bash
npx llmwiki ingest ~/workspace/my-api
```

Or install it globally:

```bash
npm install -g llmwiki
llmwiki setup
```

This package is a thin launcher. On first run it downloads the matching
prebuilt binary for your platform from the [GitHub release](https://github.com/emgiezet/llmwiki/releases),
verifies its SHA256 against `checksums.txt`, and execs it. Binaries are
published for macOS and Linux (amd64 / arm64); on other platforms use the
[install script or Docker image](https://github.com/emgiezet/llmwiki#quick-start).

The npm version tracks the llmwiki release version, so `npx llmwiki@2.6.0`
fetches the `v2.6.0` binary.

Full docs → **https://github.com/emgiezet/llmwiki**

MIT
