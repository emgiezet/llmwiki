package tracker

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// Area is a cluster of source files that change together in git history.
type Area struct {
	Name          string   // longest common directory prefix of its files
	Files         []string // sorted list of file paths relative to project root
	Hash          string   // 16-char hex prefix of SHA256 over git ls-tree output
	ClusterMethod string   // "git-cochange" or "scanner-heuristic"
}

// ComputeHash returns a deterministic 16-character hash derived from the
// git ls-tree output of each file. Files not tracked by git are skipped.
// Order of the input files does not affect the result.
func ComputeHash(runner GitRunner, projectRoot string, files []string) (string, error) {
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)

	h := sha256.New()
	for _, file := range sorted {
		line, err := runner.LSTree(projectRoot, file)
		if err != nil {
			return "", fmt.Errorf("ls-tree %s: %w", file, err)
		}
		if line == "" {
			// untracked — skip
			continue
		}
		fmt.Fprintf(h, "%s\n", line)
	}

	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

// areaName returns the longest common directory prefix shared by all files.
// Falls back to "." when no shared directory exists.
func areaName(files []string) string {
	if len(files) == 0 {
		return "."
	}

	parts := strings.Split(files[0], "/")

	// Try progressively shorter prefixes, from longest to shortest
	for n := len(parts) - 1; n >= 1; n-- {
		prefix := strings.Join(parts[:n], "/") + "/"
		allMatch := true
		for _, f := range files {
			if !strings.HasPrefix(f, prefix) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return strings.TrimSuffix(prefix, "/")
		}
	}

	return "."
}
