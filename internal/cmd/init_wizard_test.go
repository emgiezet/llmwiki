package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteProjectConfig_MinimalNoForce(t *testing.T) {
	dir := t.TempDir()
	err := writeProjectConfig(dir, initOptions{customer: "acme", projectType: "client"}, false)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "acme", cfg.Customer)
	assert.Equal(t, "client", cfg.Type)
	assert.Equal(t, "", cfg.Extraction.Preset)
	assert.Equal(t, "", cfg.OutputMode)
}

func TestWriteProjectConfig_NoForce_Errors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeProjectConfig(dir, initOptions{projectType: "client"}, false))
	err := writeProjectConfig(dir, initOptions{projectType: "personal"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestWriteProjectConfig_Force_Overwrites(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeProjectConfig(dir, initOptions{projectType: "client"}, false))
	err := writeProjectConfig(dir, initOptions{projectType: "personal"}, true)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "personal", cfg.Type)
}

func TestWriteProjectConfig_WritesPresetAndOutputMode(t *testing.T) {
	dir := t.TempDir()
	err := writeProjectConfig(dir, initOptions{
		projectType:  "personal",
		preset:       "notes",
		outputMode:   "both",
		localDocsDir: "docs/llmwiki",
	}, false)
	require.NoError(t, err)

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "notes", cfg.Extraction.Preset)
	assert.Equal(t, "both", cfg.OutputMode)
	assert.Equal(t, "docs/llmwiki", cfg.LocalDocsDir)
}

func TestInstallIntegrations_PreCommitOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))

	installIntegrations(dir, false, true)

	if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit")); err != nil {
		t.Errorf("pre-commit hook not installed: %v", err)
	}
}

func TestInstallIntegrations_None(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))

	installIntegrations(dir, false, false)

	if _, err := os.Stat(filepath.Join(dir, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook should not exist; err=%v", err)
	}
}
