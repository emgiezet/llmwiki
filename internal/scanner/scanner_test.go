package scanner_test

import (
	"os"
	"path/filepath"
	"strings"
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
	// Create service subdirectories so they are detected
	require.NoError(t, os.Mkdir(filepath.Join(dir, "api-gateway"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "policy-service"), 0755))

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

func TestDetectServices_PHPComposer(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "api-service")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "composer.json"), []byte(`{"name":"test"}`), 0644))

	services, err := scanner.DetectServices(dir)
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, "api-service", services[0].Name)
}

func TestDetectServices_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "worker")
	require.NoError(t, os.MkdirAll(svcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(svcDir, "Dockerfile"), []byte("FROM php:8.3"), 0644))

	services, err := scanner.DetectServices(dir)
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, "worker", services[0].Name)
}

func TestDetectServices_SrcDir(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "audit.service")
	require.NoError(t, os.MkdirAll(filepath.Join(svcDir, "src"), 0755))

	services, err := scanner.DetectServices(dir)
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, "audit.service", services[0].Name)
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

func TestScanProject_CollectsMainGo(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package main\nfunc main() {}"), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "main.go")
}

func TestScanProject_CollectsDockerfile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"),
		[]byte("FROM golang:1.23\nEXPOSE 8080"), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "Dockerfile")
}

func TestScanProject_CollectsCLAUDEmd(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"),
		[]byte("# Project context for Claude"), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "CLAUDE.md")
}

func TestScanProject_CollectsCmdMainGo(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "cmd", "server")
	require.NoError(t, os.MkdirAll(cmdDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(cmdDir, "main.go"),
		[]byte("package main\nfunc main() { serve() }"), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "cmd/server/main.go")
}

func TestScanProject_HighValueFilesGetMoreChars(t *testing.T) {
	dir := t.TempDir()
	bigContent := strings.Repeat("x", 7000)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"),
		[]byte(bigContent), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.NotContains(t, result.Summary, "[truncated]")
}

func TestScanProject_IncludesDirectoryTree(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "server"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal", "api"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	result, err := scanner.ScanProject(dir)
	require.NoError(t, err)
	assert.Contains(t, result.Summary, "DIRECTORY STRUCTURE")
	assert.Contains(t, result.Summary, "cmd/")
	assert.Contains(t, result.Summary, "internal/")
}

func TestScanDirectoryTree_RespectsMaxDepth(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "d")
	require.NoError(t, os.MkdirAll(deep, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "file.go"), []byte("package d"), 0644))

	tree, err := scanner.ScanDirectoryTree(dir, 2)
	require.NoError(t, err)
	assert.Contains(t, tree, "a/")
	assert.Contains(t, tree, "b/")
	assert.NotContains(t, tree, "d/")
}

func TestScanDirectoryTree_SkipsDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))

	tree, err := scanner.ScanDirectoryTree(dir, 3)
	require.NoError(t, err)
	assert.NotContains(t, tree, "node_modules")
	assert.Contains(t, tree, "src/")
}
