package config_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGlobalConfig_Defaults(t *testing.T) {
	cfg, err := config.LoadGlobalConfig("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, "claude-code", cfg.LLM)
	assert.Equal(t, "http://localhost:11434", cfg.OllamaHost)
	assert.Equal(t, filepath.Join(os.Getenv("HOME"), "llmwiki", "wiki"), cfg.WikiRoot)
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	content := "wiki_root: /custom/wiki\nllm: ollama\nollama_host: http://remote:11434\n"
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := config.LoadGlobalConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "/custom/wiki", cfg.WikiRoot)
	assert.Equal(t, "ollama", cfg.LLM)
	assert.Equal(t, "http://remote:11434", cfg.OllamaHost)
}

func TestLoadProjectConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	content := "llm: ollama\nollama_model: llama3.2\ncustomer: insly\ntype: client\n"
	path := filepath.Join(dir, "llmwiki.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "ollama", cfg.LLM)
	assert.Equal(t, "llama3.2", cfg.OllamaModel)
	assert.Equal(t, "insly", cfg.Customer)
}

func TestLoadProjectConfig_Missing(t *testing.T) {
	cfg, err := config.LoadProjectConfig(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, "", cfg.LLM) // empty means "use global default"
}

func TestMerge_ProjectOverridesGlobal(t *testing.T) {
	global := config.GlobalConfig{LLM: "claude-code", WikiRoot: "~/wiki", OllamaHost: "http://localhost:11434"}
	project := config.ProjectConfig{LLM: "ollama", OllamaModel: "llama3.2"}
	merged := config.Merge(global, config.ClientConfig{}, project)
	assert.Equal(t, "ollama", merged.LLM)
	assert.Equal(t, "llama3.2", merged.OllamaModel)
	assert.Equal(t, "~/wiki", merged.WikiRoot)
}

func TestMerge_GlobalFillsEmptyProject(t *testing.T) {
	global := config.GlobalConfig{LLM: "claude-code", WikiRoot: "~/wiki", OllamaHost: "http://localhost:11434"}
	project := config.ProjectConfig{Customer: "acme"}
	merged := config.Merge(global, config.ClientConfig{}, project)
	assert.Equal(t, "claude-code", merged.LLM)
	assert.Equal(t, "acme", merged.Customer)
}

func TestLoadGlobalConfig_WarnsPlainterxtAPIKey(t *testing.T) {
	dir := t.TempDir()
	content := "anthropic_api_key: sk-ant-test123\n"
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	// Capture stderr by redirecting os.Stderr.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	cfg, loadErr := config.LoadGlobalConfig(path)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, loadErr)
	assert.Equal(t, "sk-ant-test123", cfg.AnthropicAPIKey)
	assert.Contains(t, buf.String(), "anthropic_api_key stored in plaintext")
}

func TestLoadProjectConfig_ExtractionBlock(t *testing.T) {
	dir := t.TempDir()
	content := `llm: claude-api
extraction:
  preset: software
  sections: [domain, architecture]
  max_tokens: 4000
`
	path := filepath.Join(dir, "llmwiki.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "software", cfg.Extraction.Preset)
	assert.Equal(t, []string{"domain", "architecture"}, cfg.Extraction.Sections)
	assert.Equal(t, 4000, cfg.Extraction.MaxTokens)
}

func TestMerge_ExtractionCarriedFromProject(t *testing.T) {
	global := config.GlobalConfig{LLM: "claude-code", WikiRoot: "~/wiki"}
	project := config.ProjectConfig{
		Extraction: config.ExtractionConfig{
			Preset:    "minimal",
			MaxTokens: 1500,
		},
	}
	merged := config.Merge(global, config.ClientConfig{}, project)
	assert.Equal(t, "minimal", merged.Extraction.Preset)
	assert.Equal(t, 1500, merged.Extraction.MaxTokens)
}

func TestLoadProjectConfig_ExtractionOmitted(t *testing.T) {
	dir := t.TempDir()
	content := "llm: claude-code\n"
	path := filepath.Join(dir, "llmwiki.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := config.LoadProjectConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Extraction.Preset)
	assert.Nil(t, cfg.Extraction.Sections)
	assert.Equal(t, 0, cfg.Extraction.MaxTokens)
}

func TestLoadGlobalConfig_NoWarningWithoutAPIKey(t *testing.T) {
	dir := t.TempDir()
	content := "llm: claude-code\n"
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	_, loadErr := config.LoadGlobalConfig(path)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	require.NoError(t, loadErr)
	assert.NotContains(t, buf.String(), "anthropic_api_key")
}
