package ingestion

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emgiezet/llmwiki/internal/memory"
)

// ErrNothingToAbsorb is returned when no git history is found and no --note is provided.
var ErrNothingToAbsorb = errors.New("nothing to absorb: no git history found and no --note provided")

// BuildSessionContent collects git log, git diff, and optional note into the
// single blob that RememberIngestion would have stored. Returns
// ErrNothingToAbsorb if everything is empty.
func BuildSessionContent(projectDir, note string) (string, error) {
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
		return "", ErrNothingToAbsorb
	}
	return strings.Join(parts, "\n\n"), nil
}

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
	content, err := BuildSessionContent(projectDir, note)
	if err != nil {
		return err
	}
	return mem.RememberIngestion(ctx, projectName, customer, content, nil)
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
