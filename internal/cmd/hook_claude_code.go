package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed stop-hook.js
var stopHookJS []byte

const claudeCodePluginManifest = `{
  "name": "llmwiki",
  "description": "Captures analytical Claude Code sessions into graymatter memory via llmwiki absorb",
  "author": {
    "name": "Max Małecki"
  },
  "version": "1.1.0"
}
`

// claudeCodeHooksConfig uses `node` instead of `python3` (the v1.0.x default).
// Migration from a pre-1.1.0 install is handled by Install(), which deletes
// any stale stop-hook.py and rewrites hooks.json to this content.
const claudeCodeHooksConfig = `{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node ${CLAUDE_PLUGIN_ROOT}/hooks/stop-hook.js",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
`

type claudeCodeHook struct{}

func (c *claudeCodeHook) Name() string { return "claude-code" }

func (c *claudeCodeHook) pluginDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plugins", "llmwiki")
}

// Install writes (or overwrites) the llmwiki Claude Code plugin. It also
// migrates any pre-1.1.0 install: removes stop-hook.py and rewrites
// hooks.json to invoke `node` instead of `python3`. Idempotent — a second
// call is a no-op beyond re-writing identical content.
func (c *claudeCodeHook) Install() error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("node not found in PATH — install Node ≥ 18 (https://nodejs.org); downgrade to llmwiki 1.0.x if you need the legacy Python hook")
	}

	dir := c.pluginDir()

	manifestDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil { // #nosec G301 -- plugin dirs must be 0755 for Claude Code discovery
		return err
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(claudeCodePluginManifest), 0o644); err != nil { // #nosec G306 -- plugin manifest is world-readable by design
		return err
	}

	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil { // #nosec G301 -- plugin dirs must be 0755 for Claude Code discovery
		return err
	}

	// Migration: pre-1.1.0 installs carry stop-hook.py. Remove it before
	// writing the new hooks.json so no stale python3 invocation survives.
	if legacy := filepath.Join(hooksDir, "stop-hook.py"); fileExists(legacy) {
		if err := os.Remove(legacy); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove legacy python hook: %w", err)
		}
	}

	if err := os.WriteFile(filepath.Join(hooksDir, "hooks.json"), []byte(claudeCodeHooksConfig), 0o644); err != nil { // #nosec G306 -- hook config is world-readable by design
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "stop-hook.js"), stopHookJS, 0o755); err != nil { // #nosec G306 -- hook script must be executable
		return err
	}

	if _, err := exec.LookPath("llmwiki"); err != nil {
		fmt.Fprintln(os.Stderr, "warning: 'llmwiki' not found in PATH — add it before using the hook")
	}
	return nil
}

// Uninstall removes the plugin directory wholesale.
func (c *claudeCodeHook) Uninstall() error {
	dir := c.pluginDir()
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove plugin dir: %w", err)
	}
	return nil
}

// Status reports the install location when the plugin manifest is present.
// A plugin dir that contains only a stale stop-hook.py (no hooks.json) is
// treated as not-installed — the user should run `install` to migrate.
func (c *claudeCodeHook) Status() (bool, string, error) {
	dir := c.pluginDir()
	manifest := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(manifest); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, dir, nil
}

// fileExists is a small helper reused across hook installers.
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
