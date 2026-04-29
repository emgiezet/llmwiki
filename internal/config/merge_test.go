package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_LLMPrecedence_ProjectWinsOverClientOverGlobal(t *testing.T) {
	cases := []struct {
		name         string
		global       string
		client       string
		project      string
		wantLLM      string
	}{
		{"all empty — leaves LLM empty", "", "", "", ""},
		{"global only", "claude-code", "", "", "claude-code"},
		{"client overrides global", "claude-code", "ollama", "", "ollama"},
		{"project overrides both", "claude-code", "ollama", "codex", "codex"},
		{"project fills over empty global/client", "", "", "gemini-cli", "gemini-cli"},
		{"client fills over empty global/project", "", "opencode", "", "opencode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := config.GlobalConfig{LLM: tc.global}
			c := config.ClientConfig{LLM: tc.client}
			p := config.ProjectConfig{LLM: tc.project}
			got := config.Merge(g, c, p)
			assert.Equal(t, tc.wantLLM, got.LLM)
		})
	}
}

func TestMerge_ExtractionPresetPrecedence(t *testing.T) {
	g := config.GlobalConfig{}
	c := config.ClientConfig{Extraction: config.ExtractionConfig{Preset: "software"}}
	p := config.ProjectConfig{Extraction: config.ExtractionConfig{Preset: "minimal"}}

	got := config.Merge(g, c, p)
	assert.Equal(t, "minimal", got.Extraction.Preset, "project preset wins")
}

func TestMerge_ExtractionPreset_ClientFillsEmptyProject(t *testing.T) {
	c := config.ClientConfig{Extraction: config.ExtractionConfig{Preset: "software"}}
	p := config.ProjectConfig{}

	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, "software", got.Extraction.Preset, "client preset fills when project unset")
}

func TestMerge_ExtractionMaxTokens_NonZeroWins(t *testing.T) {
	c := config.ClientConfig{Extraction: config.ExtractionConfig{MaxTokens: 4000}}
	p := config.ProjectConfig{Extraction: config.ExtractionConfig{MaxTokens: 8000}}
	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, 8000, got.Extraction.MaxTokens, "project max_tokens wins")

	// Project unset → falls through to client.
	p2 := config.ProjectConfig{}
	got2 := config.Merge(config.GlobalConfig{}, c, p2)
	assert.Equal(t, 4000, got2.Extraction.MaxTokens)
}

func TestMerge_ExtractionSections_SliceOverride(t *testing.T) {
	c := config.ClientConfig{Extraction: config.ExtractionConfig{Sections: []string{"a", "b"}}}
	p := config.ProjectConfig{Extraction: config.ExtractionConfig{Sections: []string{"c"}}}
	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, []string{"c"}, got.Extraction.Sections,
		"project sections replace client sections wholesale (no concat)")
}

func TestMerge_GlobalOnlyFieldsPassThrough(t *testing.T) {
	g := config.GlobalConfig{
		WikiRoot:        "/wiki",
		OllamaHost:      "http://localhost:11434",
		AnthropicAPIKey: "sk-x",
		MemoryEnabled:   true,
	}
	got := config.Merge(g, config.ClientConfig{}, config.ProjectConfig{})
	assert.Equal(t, "/wiki", got.WikiRoot)
	assert.Equal(t, "http://localhost:11434", got.OllamaHost)
	assert.Equal(t, "sk-x", got.AnthropicAPIKey)
	assert.True(t, got.MemoryEnabled)
}

func TestMerge_ProjectOnlyFieldsPassThrough(t *testing.T) {
	p := config.ProjectConfig{Customer: "acme", Type: "client", OllamaModel: "llama3.2"}
	got := config.Merge(config.GlobalConfig{}, config.ClientConfig{}, p)
	assert.Equal(t, "acme", got.Customer)
	assert.Equal(t, "client", got.Type)
	assert.Equal(t, "llama3.2", got.OllamaModel)
}

func TestLoadClientConfig_MissingFileReturnsZeroValue(t *testing.T) {
	// Use a customer name that definitely has no config file anywhere.
	got, err := config.LoadClientConfig("nonexistent-customer-for-testing")
	require.NoError(t, err, "missing client config must not be an error")
	assert.Equal(t, config.ClientConfig{}, got)
}

func TestLoadClientConfig_EmptyCustomerReturnsZero(t *testing.T) {
	got, err := config.LoadClientConfig("")
	require.NoError(t, err)
	assert.Equal(t, config.ClientConfig{}, got)
}

func TestLoadClientConfig_ParsesYAML(t *testing.T) {
	// Redirect HOME so LoadClientConfig reads from a temp dir instead of
	// the real user profile.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	clientsDir := filepath.Join(tmpHome, ".llmwiki", "clients")
	require.NoError(t, os.MkdirAll(clientsDir, 0o755))
	yaml := "llm: codex\nextraction:\n  preset: software\n  max_tokens: 4000\n"
	require.NoError(t, os.WriteFile(filepath.Join(clientsDir, "acme.yaml"), []byte(yaml), 0o644))

	got, err := config.LoadClientConfig("acme")
	require.NoError(t, err)
	assert.Equal(t, "codex", got.LLM)
	assert.Equal(t, "software", got.Extraction.Preset)
	assert.Equal(t, 4000, got.Extraction.MaxTokens)
}

func TestLoadClientConfig_RejectsInvalidCustomerName(t *testing.T) {
	_, err := config.LoadClientConfig("../etc/passwd")
	require.Error(t, err, "path traversal must be rejected")
}

func TestMerge_EndToEnd_WithClientYAML(t *testing.T) {
	// Full flow: write a client YAML, load it, merge with global + project,
	// verify precedence end-to-end.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	clientsDir := filepath.Join(tmpHome, ".llmwiki", "clients")
	require.NoError(t, os.MkdirAll(clientsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(clientsDir, "acme.yaml"),
		[]byte("llm: codex\nextraction:\n  preset: software\n"),
		0o644,
	))

	cc, err := config.LoadClientConfig("acme")
	require.NoError(t, err)

	global := config.GlobalConfig{LLM: "claude-code", WikiRoot: "/wiki"}
	project := config.ProjectConfig{Customer: "acme", Type: "client"}

	merged := config.Merge(global, cc, project)
	assert.Equal(t, "codex", merged.LLM, "client LLM overrides global")
	assert.Equal(t, "software", merged.Extraction.Preset, "client preset passes through when project unset")
	assert.Equal(t, "acme", merged.Customer)
	assert.Equal(t, "/wiki", merged.WikiRoot)
}
