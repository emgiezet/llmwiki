package cmd

import (
	"bytes"
	"os"
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
	// Pre-existing file with a key the wizard never manages.
	require.NoError(t, os.WriteFile(path, []byte("claude_binary_path: /opt/claude\nllm: claude-code\n"), 0o600))

	cfg := config.GlobalConfig{LLM: "ollama", WikiRoot: "/w", OllamaHost: "http://h:11434"}
	require.NoError(t, saveGlobalConfig(path, cfg))

	reloaded, err := config.LoadGlobalConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "ollama", reloaded.LLM)                   // managed key updated
	assert.Equal(t, "/opt/claude", reloaded.ClaudeBinaryPath) // unmanaged key preserved
	assert.Equal(t, "/w", reloaded.WikiRoot)
}

func TestSaveGlobalConfig_WritesMinimalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := config.GlobalConfig{LLM: "claude-code", WikiRoot: "/w"} // memory disabled

	require.NoError(t, saveGlobalConfig(path, cfg))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(data)
	// A fresh claude-code config must not gain default extractor maps, empty
	// scalars, or backend-specific keys that don't apply.
	assert.NotContains(t, out, "extractors")
	assert.NotContains(t, out, "anthropic_api_key")
	assert.NotContains(t, out, "ollama_host")
	assert.NotContains(t, out, "memory_mode") // memory disabled → not written
	assert.Contains(t, out, "llm: claude-code")
	assert.Contains(t, out, "wiki_root: /w")
}

func TestRunSetupWizard_ExtractorsListedSorted(t *testing.T) {
	// claude-code default, wiki_root Enter, memory n, save n
	input := "\n\nn\nn\n"
	out := &bytes.Buffer{}
	p := wizard.New(strings.NewReader(input), out)
	cfg := config.GlobalConfig{
		LLM:      "claude-code",
		WikiRoot: "/w",
		Extractors: map[string]string{
			".pdf":  "pdftotext {{input}} -",
			".docx": "pandoc a",
			".odt":  "pandoc b",
		},
	}

	runSetupWizard(p, &cfg)

	s := out.String()
	iDocx := strings.Index(s, ".docx")
	iOdt := strings.Index(s, ".odt")
	iPdf := strings.Index(s, ".pdf")
	require.True(t, iDocx >= 0 && iOdt >= 0 && iPdf >= 0, "all extractors should be listed")
	assert.Less(t, iDocx, iOdt, "extractors should be listed in sorted order")
	assert.Less(t, iOdt, iPdf, "extractors should be listed in sorted order")
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
