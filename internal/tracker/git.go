package tracker

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNoGit = errors.New("git not found in PATH")

// CommitFiles holds a commit hash and the files changed in that commit.
type CommitFiles struct {
	Hash  string
	Files []string
}

// GitRunner is the interface for querying git history.
type GitRunner interface {
	LogFiles(projectRoot string) ([]CommitFiles, error)
	LSTree(projectRoot, file string) (string, error)
}

// RealGitRunner executes real git subprocesses.
type RealGitRunner struct{}

// LogFiles runs git log with --name-only to get per-commit file lists.
func (r RealGitRunner) LogFiles(projectRoot string) ([]CommitFiles, error) {
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--name-only", "--format=COMMIT:%H", "--diff-filter=ACMR")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return parseLogOutput(string(out)), nil
}

// LSTree runs git ls-tree HEAD for a single file and returns the trimmed output.
// Returns an empty string if the exit code is 128 (not tracked / not a repo).
func (r RealGitRunner) LSTree(projectRoot, file string) (string, error) {
	cmd := exec.Command("git", "-C", projectRoot, "ls-tree", "HEAD", "--", file)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// NewGitRunner verifies git is available in PATH and returns a RealGitRunner.
func NewGitRunner() (GitRunner, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrNoGit
	}
	return RealGitRunner{}, nil
}

// parseLogOutput converts the raw text output of git log --name-only --format=COMMIT:%H
// into a slice of CommitFiles.
func parseLogOutput(output string) []CommitFiles {
	var result []CommitFiles
	var current *CommitFiles

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "COMMIT:"):
			if current != nil {
				result = append(result, *current)
			}
			hash := strings.TrimPrefix(line, "COMMIT:")
			current = &CommitFiles{Hash: hash, Files: []string{}}
		case line == "":
			// blank separator — skip
		default:
			if current != nil {
				current.Files = append(current.Files, line)
			}
		}
	}

	if current != nil {
		result = append(result, *current)
	}

	return result
}
