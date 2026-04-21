package ingestion

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mgz/llmwiki/internal/memory"
)

// AbsorbSession extracts facts from a work session into graymatter memory without
// generating a wiki entry. Near-zero direct LLM cost — graymatter handles extraction async.
//
// projectName defaults to filepath.Base(projectDir) if empty.
// mem may be nil: safe no-op.
func AbsorbSession(ctx context.Context, projectDir, projectName, customer, note string, mem *memory.Store) error {
	if mem == nil || !mem.Enabled() {
		return nil
	}

	if projectName == "" {
		projectName = filepath.Base(projectDir)
	}

	var parts []string

	if log := gitCommand(projectDir, "log", "--oneline", "-20"); log != "" {
		parts = append(parts, "Recent commits:\n"+log)
	}
	if diff := gitCommand(projectDir, "diff", "--stat", "HEAD~5..HEAD"); diff != "" {
		parts = append(parts, "Recent file changes:\n"+diff)
	}
	if note != "" {
		parts = append(parts, "Session note: "+note)
	}

	if len(parts) == 0 {
		return nil
	}

	return mem.RememberIngestion(ctx, projectName, customer, strings.Join(parts, "\n\n"), nil)
}

// gitCommand runs a git subcommand in dir and returns trimmed stdout.
// Returns "" on any error (not a repo, not enough commits, etc.).
func gitCommand(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
