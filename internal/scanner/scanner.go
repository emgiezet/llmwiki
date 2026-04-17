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

// fileTarget defines a file pattern and its max allowed content size.
type fileTarget struct {
	Pattern  string
	MaxChars int
}

// highValueTargets are files that deserve a larger truncation budget.
var highValueTargets = []fileTarget{
	{"main.go", 6000},
	{"Dockerfile", 6000},
	{"CLAUDE.md", 8000},
	{"AGENTS.md", 8000},
	{"*.swagger.json", 8000},
	{"*.swagger.yaml", 8000},
	{"swagger.json", 8000},
	{"swagger.yaml", 8000},
	{"openapi*.json", 8000},
	{"openapi*.yaml", 8000},
}

// standardTargets are config/manifest files with standard truncation.
var standardTargets = []fileTarget{
	{"README*", 4000},
	{"readme*", 4000},
	{"go.mod", 4000},
	{"package.json", 4000},
	{"Cargo.toml", 4000},
	{"pyproject.toml", 4000},
	{"docker-compose*.yml", 4000},
	{"docker-compose*.yaml", 4000},
	{"*.proto", 4000},
	{".env.example", 4000},
	{"Makefile", 4000},
	{"Jenkinsfile", 3000},
	{"CHANGELOG.md", 3000},
	{".golangci.yml", 2000},
	{".eslintrc*", 2000},
	{"go.work", 2000},
}

// pathTargets match on relative path (not just filename).
var pathTargets = []fileTarget{
	{"cmd/*/main.go", 6000},
	{".github/workflows/*.yml", 3000},
	{".github/workflows/*.yaml", 3000},
	{"docs/swagger.json", 8000},
	{"docs/swagger.yaml", 8000},
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
	"__pycache__":  true,
	".venv":        true,
	".tox":         true,
	"coverage":     true,
	".next":        true,
	".nuxt":        true,
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
		rel, _ := filepath.Rel(dir, path)
		matched, maxChars := matchTarget(d.Name())
		if !matched {
			matched, maxChars = matchPathTarget(rel)
		}
		if matched {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			content := string(data)
			if len(content) > maxChars {
				content = content[:maxChars] + "\n[truncated]"
			}
			parts = append(parts, fmt.Sprintf("=== %s ===\n%s", rel, content))
		}
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}

	// Append directory tree for structural context
	tree, _ := ScanDirectoryTree(dir, 3)
	if tree != "" {
		parts = append(parts, fmt.Sprintf("=== DIRECTORY STRUCTURE ===\n%s", tree))
	}

	return ScanResult{
		Dir:     dir,
		Summary: strings.Join(parts, "\n\n"),
	}, nil
}

// matchTarget checks filename against high-value and standard targets.
func matchTarget(name string) (bool, int) {
	for _, t := range highValueTargets {
		if ok, _ := filepath.Match(t.Pattern, name); ok {
			return true, t.MaxChars
		}
	}
	for _, t := range standardTargets {
		if ok, _ := filepath.Match(t.Pattern, name); ok {
			return true, t.MaxChars
		}
	}
	return false, 0
}

// matchPathTarget checks relative path against path-based targets.
func matchPathTarget(relPath string) (bool, int) {
	for _, t := range pathTargets {
		if ok, _ := filepath.Match(t.Pattern, relPath); ok {
			return true, t.MaxChars
		}
	}
	return false, 0
}

// ScanDirectoryTree returns a tree listing of dir up to maxDepth levels.
func ScanDirectoryTree(dir string, maxDepth int) (string, error) {
	var lines []string
	base := filepath.Base(dir)
	lines = append(lines, base+"/")

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator)) + 1
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		indent := strings.Repeat("  ", depth)
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		lines = append(lines, indent+name)
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}
