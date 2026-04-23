package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed hook_pi_extension.ts
var piExtensionTS []byte

// piHook installs a TypeScript extension to ~/.pi/agent/extensions/llmwiki.ts
// (global). Pi auto-loads extensions from that directory; the extension
// registers `pi.on("agent_end", ...)` (fires once per user prompt after
// tools complete) and forwards the last assistant message to
// `llmwiki absorb`.
type piHook struct{}

func (p *piHook) Name() string { return "pi" }

func (p *piHook) extensionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi", "agent", "extensions", "llmwiki.ts")
}

func (p *piHook) Install() error {
	target := p.extensionPath()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301
		return err
	}
	if err := os.WriteFile(target, piExtensionTS, 0o644); err != nil { // #nosec G306
		return err
	}
	fmt.Fprintln(os.Stderr, "pi extension installed at "+target)
	fmt.Fprintln(os.Stderr, "restart pi for the extension to load.")
	return nil
}

func (p *piHook) Uninstall() error {
	if err := os.Remove(p.extensionPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", p.extensionPath(), err)
	}
	fmt.Fprintln(os.Stderr, "pi extension removed.")
	return nil
}

func (p *piHook) Status() (bool, string, error) {
	if _, err := os.Stat(p.extensionPath()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, p.extensionPath(), nil
}
