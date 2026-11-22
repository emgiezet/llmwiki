package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed hook_omp_extension.ts
var ompExtensionTS []byte

// ompHook installs a TypeScript extension to <agentDir>/extensions/llmwiki.ts
// (global). omp auto-loads extensions from that directory; unlike pi 0.79's
// global-injection contract, omp requires a default-exported factory. The
// extension buffers the last assistant text and, on `agent_end`, forwards it
// to `llmwiki absorb`.
type ompHook struct{}

func (o *ompHook) Name() string { return "omp" }

// ompAgentDir resolves omp's agent directory: PI_CODING_AGENT_DIR when set
// (omp honours it for the default profile), else ~/.omp/agent.
func ompAgentDir() string {
	if d := os.Getenv("PI_CODING_AGENT_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".omp", "agent")
}

func (o *ompHook) extensionPath() string {
	return filepath.Join(ompAgentDir(), "extensions", "llmwiki.ts")
}

// Install uses owner-only permissions: omp runs as the installing user, so
// nothing else needs to read the extension.
func (o *ompHook) Install() error {
	target := o.extensionPath()
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(target, ompExtensionTS, 0o600); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "omp extension installed at "+target)
	fmt.Fprintln(os.Stderr, "restart omp for the extension to load.")
	return nil
}

func (o *ompHook) Uninstall() error {
	if err := os.Remove(o.extensionPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", o.extensionPath(), err)
	}
	fmt.Fprintln(os.Stderr, "omp extension removed.")
	return nil
}

func (o *ompHook) Status() (bool, string, error) {
	if _, err := os.Stat(o.extensionPath()); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, o.extensionPath(), nil
}
