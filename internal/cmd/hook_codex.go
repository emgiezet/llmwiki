package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed hook_codex_wrapper.js
var codexWrapperJS []byte

const (
	codexMarkerBegin = "# llmwiki:begin codex-notify"
	codexMarkerEnd   = "# llmwiki:end codex-notify"
)

// codexHook installs a `notify` entry in ~/.codex/config.toml that points at
// a small Node wrapper script which forwards the agent-turn payload (sent by
// codex as a JSON argv tail) to `llmwiki absorb`.
//
// The TOML edit is done with marker-delimited comments so we can
// install/uninstall idempotently without pulling in a full TOML parser.
// If the user already has a top-level `notify = ...` outside our markers,
// install refuses rather than silently overwriting.
type codexHook struct{}

func (c *codexHook) Name() string { return "codex" }

func (c *codexHook) wrapperPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".llmwiki", "hooks", "codex-absorb.js")
}

func (c *codexHook) configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "config.toml")
}

// existingTopLevelNotify matches a `notify = ...` assignment at the start of
// a line. Used to detect user-defined notify entries outside our marker
// block before we install our own.
var existingTopLevelNotify = regexp.MustCompile(`(?m)^\s*notify\s*=`)

func (c *codexHook) Install() error {
	// Write the Node wrapper.
	if err := os.MkdirAll(filepath.Dir(c.wrapperPath()), 0o755); err != nil { // #nosec G301
		return err
	}
	if err := os.WriteFile(c.wrapperPath(), codexWrapperJS, 0o755); err != nil { // #nosec G306 -- must be executable-readable
		return err
	}

	// Load existing codex config (or start empty).
	cfgPath := c.configPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil { // #nosec G301
		return err
	}

	existing, err := os.ReadFile(cfgPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", cfgPath, err)
	}

	// Before we add our marker block, check that the user hasn't already
	// defined a `notify` key outside our markers — TOML disallows duplicate
	// top-level keys, and silently clobbering user config is a no-go.
	stripped, _ := removeMarkerRegion(string(existing), codexMarkerBegin, codexMarkerEnd)
	if existingTopLevelNotify.MatchString(stripped) {
		return fmt.Errorf(
			"refusing to install: %s already has a top-level `notify = ...` entry. "+
				"remove or comment it out, then re-run `llmwiki hook install codex`",
			cfgPath,
		)
	}

	body := fmt.Sprintf("notify = [\"node\", %q]", c.wrapperPath())

	_, err = upsertMarkerBlock(cfgPath, markerBlock{
		Begin: codexMarkerBegin,
		End:   codexMarkerEnd,
		Body:  body,
	})
	if err != nil {
		return fmt.Errorf("update %s: %w", cfgPath, err)
	}

	if strings.TrimSpace(string(existing)) == "" {
		fmt.Fprintln(os.Stderr, "codex config created at "+cfgPath)
	}
	fmt.Fprintln(os.Stderr, "codex hook installed. New codex sessions will forward turn-end payloads to `llmwiki absorb`.")
	return nil
}

func (c *codexHook) Uninstall() error {
	// Strip our block from the TOML config.
	if _, err := removeMarkerBlock(c.configPath(), codexMarkerBegin, codexMarkerEnd); err != nil {
		return fmt.Errorf("clean %s: %w", c.configPath(), err)
	}
	// Remove wrapper script (ignore missing).
	if err := os.Remove(c.wrapperPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", c.wrapperPath(), err)
	}
	fmt.Fprintln(os.Stderr, "codex hook uninstalled.")
	return nil
}

func (c *codexHook) Status() (bool, string, error) {
	if _, err := os.Stat(c.wrapperPath()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, c.wrapperPath(), nil
}
