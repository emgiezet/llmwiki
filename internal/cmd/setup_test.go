package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/wizard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSetupWizard_BuildsConfig(t *testing.T) {
	// backend=ollama(3), host=Enter(default), wiki_root=/w, memory=y, mode=global(2), final save=y
	input := "3\n\n/w\ny\n2\ny\n"
	p := wizard.New(strings.NewReader(input), &bytes.Buffer{})
	cfg := config.GlobalConfig{LLM: "claude-code", OllamaHost: "http://localhost:11434", WikiRoot: "~/llmwiki/wiki"}

	save := runSetupWizard(p, &cfg)

	assert.True(t, save)
	assert.Equal(t, "ollama", cfg.LLM)
	assert.Equal(t, "http://localhost:11434", cfg.OllamaHost)
	assert.Equal(t, "/w", cfg.WikiRoot)
	assert.True(t, cfg.MemoryEnabled)
	assert.Equal(t, "global", cfg.MemoryMode)
}

func TestRunSetupWizard_CancelReturnsFalse(t *testing.T) {
	// backend=Enter(claude-code default), wiki_root=Enter, memory=n, final save=n
	input := "\n\nn\nn\n"
	p := wizard.New(strings.NewReader(input), &bytes.Buffer{})
	cfg := config.GlobalConfig{LLM: "claude-code", WikiRoot: "~/llmwiki/wiki"}

	save := runSetupWizard(p, &cfg)

	assert.False(t, save)
}

func TestSaveGlobalConfig_PreservesUnmanagedFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := config.GlobalConfig{
		LLM:              "ollama",
		WikiRoot:         "/w",
		ClaudeBinaryPath: "/opt/claude",
	}
	require.NoError(t, saveGlobalConfig(path, cfg))

	reloaded, err := config.LoadGlobalConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "ollama", reloaded.LLM)
	assert.Equal(t, "/opt/claude", reloaded.ClaudeBinaryPath)
}

func TestSetupCmd_NoTTY_Errors(t *testing.T) {
	orig := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = orig }()

	cmd := NewSetupCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interactive terminal")
}
