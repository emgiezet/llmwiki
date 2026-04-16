package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanResult holds the collected text from a project or service directory.
type ScanResult struct {
	Dir     string
	Summary string // concatenated file contents, formatted for LLM prompt
}

// collectTargets lists file patterns worth scanning.
var collectTargets = []string{
	"README*", "readme*",
	"go.mod", "package.json", "Cargo.toml", "pyproject.toml",
	"docker-compose*.yml", "docker-compose*.yaml",
	"*.proto",
	".env.example",
	"Makefile",
}

// skipDirs are directory names to ignore.
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".idea":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

// ScanProject walks dir and collects relevant file contents.
func ScanProject(dir string) (ScanResult, error) {
	var parts []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldCollect(d.Name()) {
			rel, _ := filepath.Rel(dir, path)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			// Truncate very large files
			content := string(data)
			if len(content) > 4000 {
				content = content[:4000] + "\n[truncated]"
			}
			parts = append(parts, fmt.Sprintf("=== %s ===\n%s", rel, content))
		}
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}

	return ScanResult{
		Dir:     dir,
		Summary: strings.Join(parts, "\n\n"),
	}, nil
}

func shouldCollect(name string) bool {
	for _, pattern := range collectTargets {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}
