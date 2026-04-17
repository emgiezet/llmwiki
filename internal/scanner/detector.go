package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServiceDir represents a detected service subdirectory.
type ServiceDir struct {
	Name string
	Path string
}

// DetectServices detects services in a project directory.
// Returns empty slice for single-service projects.
// Detection order: docker-compose.yml > subdirectories with code.
func DetectServices(dir string) ([]ServiceDir, error) {
	// 1. Try docker-compose
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml"} {
		services, err := servicesFromCompose(filepath.Join(dir, name))
		if err == nil && len(services) > 0 {
			return services, nil
		}
	}

	// 2. Fall back to subdirectories that look like services
	var services []ServiceDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || skipDirs[e.Name()] || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		subdir := filepath.Join(dir, e.Name())
		if looksLikeService(subdir) {
			services = append(services, ServiceDir{Name: e.Name(), Path: subdir})
		}
	}
	return services, nil
}

func servicesFromCompose(path string) ([]ServiceDir, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var compose struct {
		Services map[string]interface{} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	var result []ServiceDir
	for name := range compose.Services {
		subdir := filepath.Join(dir, name)
		if _, err := os.Stat(subdir); err == nil {
			result = append(result, ServiceDir{Name: name, Path: subdir})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func looksLikeService(dir string) bool {
	indicators := []string{
		"go.mod", "package.json", "Cargo.toml",
		"main.go", "main.py", "*.proto",
		"composer.json", "Dockerfile",
		"requirements.txt", "Gemfile",
		"build.gradle", "pom.xml",
	}
	for _, pattern := range indicators {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(matches) > 0 {
			return true
		}
	}
	// Fallback: src/ directory is a strong service indicator
	if info, err := os.Stat(filepath.Join(dir, "src")); err == nil && info.IsDir() {
		return true
	}
	return false
}
