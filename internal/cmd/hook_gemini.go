package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed hook_gemini_wrapper.sh
var geminiWrapperPosix []byte

//go:embed hook_gemini_wrapper.fish
var geminiWrapperFish []byte

const (
	geminiMarkerBegin = "# llmwiki:begin gemini-wrapper"
	geminiMarkerEnd   = "# llmwiki:end gemini-wrapper"
)

// geminiCLIHook installs a shell-function wrapper around `gemini` that
// intercepts non-interactive (`-p`/`--prompt`) calls and forwards their
// stdout to `llmwiki absorb`. Interactive TUI sessions pass through
// unchanged — capturing them would produce ANSI-garbled output.
//
// Gemini CLI has no native hook / plugin API (only MCP server tools),
// so the wrapper is the only available integration surface.
type geminiCLIHook struct{}

func (g *geminiCLIHook) Name() string { return "gemini-cli" }

func (g *geminiCLIHook) wrapperDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".llmwiki", "shell")
}

func (g *geminiCLIHook) posixWrapperPath() string {
	return filepath.Join(g.wrapperDir(), "gemini-wrapper.sh")
}

func (g *geminiCLIHook) fishWrapperPath() string {
	return filepath.Join(g.wrapperDir(), "gemini-wrapper.fish")
}

// Install writes the wrapper files under ~/.llmwiki/shell/ and appends a
// marker-delimited source block to the user's rc file. Idempotent — the
// marker block is replaced rather than duplicated on re-install.
func (g *geminiCLIHook) Install() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}

	if err := os.MkdirAll(g.wrapperDir(), 0o755); err != nil { // #nosec G301 -- user-owned dir
		return err
	}
	if err := os.WriteFile(g.posixWrapperPath(), geminiWrapperPosix, 0o644); err != nil { // #nosec G306
		return err
	}
	if err := os.WriteFile(g.fishWrapperPath(), geminiWrapperFish, 0o644); err != nil { // #nosec G306
		return err
	}

	kind := detectShell(os.Getenv("SHELL"))
	rc := rcFilePath(kind, home)
	body := g.sourceLineFor(kind)

	_, err = upsertMarkerBlock(rc, markerBlock{
		Begin: geminiMarkerBegin,
		End:   geminiMarkerEnd,
		Body:  body,
	})
	if err != nil {
		return fmt.Errorf("update rc file %s: %w", rc, err)
	}

	hint := shellPostInstallHint(kind, home)
	fmt.Fprintf(os.Stderr, "gemini-cli wrapper installed. Run `%s` or open a new shell to activate.\n", hint)
	if kind == shellOther {
		fmt.Fprintln(os.Stderr, "note: could not detect shell family from $SHELL; wrote to ~/.profile. If your shell uses a different rc file, source the wrapper manually.")
	}
	return nil
}

// sourceLineFor returns the rc-file line that makes the wrapper active in
// the given shell. Fish requires a different source syntax.
func (g *geminiCLIHook) sourceLineFor(kind shellKind) string {
	if kind == shellFish {
		return fmt.Sprintf(`[ -f %q ] && source %q`, g.fishWrapperPath(), g.fishWrapperPath())
	}
	return fmt.Sprintf(`[ -f %q ] && . %q`, g.posixWrapperPath(), g.posixWrapperPath())
}

// Uninstall strips the marker block from the rc file and removes the
// wrapper files. Leaves ~/.llmwiki/shell/ directory untouched — it may be
// shared with other hook tools later.
func (g *geminiCLIHook) Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}

	// We don't know which rc file was touched at install time (user may
	// have changed shells since). Strip the block from every possible rc,
	// so uninstall cleans up regardless.
	for _, kind := range []shellKind{shellBash, shellZsh, shellFish, shellOther} {
		rc := rcFilePath(kind, home)
		if _, err := removeMarkerBlock(rc, geminiMarkerBegin, geminiMarkerEnd); err != nil {
			return fmt.Errorf("clean rc file %s: %w", rc, err)
		}
	}

	// Remove wrapper files. Ignore not-exist.
	for _, p := range []string{g.posixWrapperPath(), g.fishWrapperPath()} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	fmt.Fprintln(os.Stderr, "gemini-cli wrapper uninstalled. Restart your shell for the change to take effect.")
	return nil
}

// Status reports installed when the posix wrapper file is present on disk.
// We don't check the rc file contents — the wrapper file is the
// authoritative signal and a user can always re-install if they're unsure.
func (g *geminiCLIHook) Status() (bool, string, error) {
	p := g.posixWrapperPath()
	if _, err := os.Stat(p); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, p, nil
}
