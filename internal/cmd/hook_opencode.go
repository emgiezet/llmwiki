package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed hook_opencode_plugin.ts
var opencodePluginTS []byte

// opencodeHook installs a TypeScript plugin to
// ~/.config/opencode/plugins/llmwiki.ts (global). The plugin subscribes to
// opencode's `session.idle` event and forwards the last assistant message
// to `llmwiki absorb` via the plugin API's Bun-provided `$` shell helper.
//
// Opencode auto-loads .ts files from the plugins directory; no build /
// install step is required beyond writing the file.
type opencodeHook struct{}

func (o *opencodeHook) Name() string { return "opencode" }

func (o *opencodeHook) pluginPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode", "plugins", "llmwiki.ts")
}

func (o *opencodeHook) Install() error {
	target := o.pluginPath()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301
		return err
	}
	if err := os.WriteFile(target, opencodePluginTS, 0o644); err != nil { // #nosec G306
		return err
	}
	fmt.Fprintln(os.Stderr, "opencode plugin installed at "+target)
	fmt.Fprintln(os.Stderr, "restart opencode for the plugin to load.")
	return nil
}

func (o *opencodeHook) Uninstall() error {
	if err := os.Remove(o.pluginPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", o.pluginPath(), err)
	}
	fmt.Fprintln(os.Stderr, "opencode plugin removed.")
	return nil
}

func (o *opencodeHook) Status() (bool, string, error) {
	if _, err := os.Stat(o.pluginPath()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, o.pluginPath(), nil
}
