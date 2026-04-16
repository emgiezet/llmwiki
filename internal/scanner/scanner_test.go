package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgz/llmwiki/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanProject_CollectsREADME(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# My Project\nDoes things."), 0644))

	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "README.md")
	assert.Contains(t, result.Summary, "# My Project")
}

func TestScanProject_CollectsGoMod(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.22\n"), 0644))

	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "go.mod")
}

func TestScanProject_IgnoresNodeModules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("// ignore"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Top"), 0644))

	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.NotContains(t, result.Summary, "node_modules")
}

func TestDetectServices_FromDockerCompose(t *testing.T) {
	dir := t.TempDir()
	compose := `services:
  api-gateway:
    image: go-service
  policy-service:
    image: go-service
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644))

	services, err := scanner.DetectServices(dir)
	require.NoError(t, err)
	names := make([]string, len(services))
	for i, s := range services {
		names[i] = s.Name
	}
	assert.Contains(t, names, "api-gateway")
	assert.Contains(t, names, "policy-service")
}

func TestDetectServices_SingleService(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n"), 0644))

	services, err := scanner.DetectServices(dir)
	require.NoError(t, err)
	assert.Len(t, services, 0) // single-service: no subdirs = no service split
}

func TestScanService_CollectsProto(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "api-gateway")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "service.proto"), []byte("syntax = \"proto3\";\nservice Gateway {}"), 0644))

	result, err := scanner.ScanProject(svcDir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "service.proto")
}
