package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// shellKind is the user's detected shell family, which decides which rc file
// gets the source line for wrapper-based hooks (gemini-cli).
type shellKind int

const (
	shellBash shellKind = iota
	shellZsh
	shellFish
	shellOther
)

// detectShell inspects $SHELL and returns the family. Unknown shells fall
// back to shellOther, which wrapper installers treat as "use ~/.profile and
// warn that the syntax may not match".
func detectShell(shellEnv string) shellKind {
	switch filepath.Base(strings.TrimSpace(shellEnv)) {
	case "bash":
		return shellBash
	case "zsh":
		return shellZsh
	case "fish":
		return shellFish
	default:
		return shellOther
	}
}

// rcFilePath returns the absolute path to the rc file for a given shell,
// rooted at home. Fish uses ~/.config/fish/config.fish; bash/zsh use
// ~/.bashrc / ~/.zshrc; shellOther falls back to ~/.profile.
func rcFilePath(kind shellKind, home string) string {
	switch kind {
	case shellBash:
		return filepath.Join(home, ".bashrc")
	case shellZsh:
		return filepath.Join(home, ".zshrc")
	case shellFish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return filepath.Join(home, ".profile")
	}
}

// markerBlock is a section of rc-file content delimited by begin/end
// comments. Used to install/uninstall wrapper source lines idempotently.
type markerBlock struct {
	// Begin and End are the exact sentinel lines that bracket the block.
	// Example: "# llmwiki:begin gemini-wrapper" / "# llmwiki:end gemini-wrapper".
	Begin string
	End   string
	// Body is the content between the markers (no leading/trailing newline —
	// the writer adds them).
	Body string
}

// upsertMarkerBlock inserts the block into the target file, or replaces the
// existing instance if the same Begin/End sentinels are already present.
// Missing target file is created. Returns true if the file was modified.
func upsertMarkerBlock(target string, block markerBlock) (bool, error) {
	existing, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	stripped, had := removeMarkerRegion(string(existing), block.Begin, block.End)

	// Reassemble: keep the rest untouched, then append our block fresh.
	var out strings.Builder
	out.WriteString(stripped)
	if len(stripped) > 0 && !strings.HasSuffix(stripped, "\n") {
		out.WriteString("\n")
	}
	out.WriteString(block.Begin)
	out.WriteString("\n")
	if block.Body != "" {
		out.WriteString(block.Body)
		if !strings.HasSuffix(block.Body, "\n") {
			out.WriteString("\n")
		}
	}
	out.WriteString(block.End)
	out.WriteString("\n")

	if string(existing) == out.String() {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301 -- rc file dirs are user-owned
		return false, err
	}
	if err := os.WriteFile(target, []byte(out.String()), 0o644); err != nil { // #nosec G306 -- rc files are user-owned
		return false, err
	}
	_ = had // silence ineffassign: we don't distinguish insert vs replace in the return value
	return true, nil
}

// removeMarkerRegion returns the content with any region between Begin and
// End (inclusive) stripped. had is true when at least one complete region
// was removed. Multiple stray begin/end pairs are all stripped.
func removeMarkerRegion(content, begin, end string) (stripped string, had bool) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	var kept strings.Builder
	inRegion := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == begin {
			inRegion = true
			had = true
			continue
		}
		if inRegion {
			if strings.TrimSpace(line) == end {
				inRegion = false
			}
			continue
		}
		kept.WriteString(line)
		kept.WriteString("\n")
	}
	// If the file didn't end with newline and we flushed a trailing newline,
	// trim it so round-trip is clean.
	out := kept.String()
	if !strings.HasSuffix(content, "\n") && strings.HasSuffix(out, "\n") {
		out = strings.TrimSuffix(out, "\n")
	}
	return out, had
}

// removeMarkerBlock is the inverse of upsertMarkerBlock: strips the region
// and writes the file back. Returns whether anything was removed.
func removeMarkerBlock(target, begin, end string) (bool, error) {
	existing, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	stripped, had := removeMarkerRegion(string(existing), begin, end)
	if !had {
		return false, nil
	}
	if err := os.WriteFile(target, []byte(stripped), 0o644); err != nil { // #nosec G306 -- rc files are user-owned
		return false, err
	}
	return true, nil
}

// shellPostInstallHint returns the command a user should run after the
// installer writes to an rc file, so the new function / source line takes
// effect in the current shell.
func shellPostInstallHint(kind shellKind, home string) string {
	rc := rcFilePath(kind, home)
	switch kind {
	case shellFish:
		return fmt.Sprintf("source %s", rc)
	default:
		return fmt.Sprintf(". %s", rc)
	}
}
