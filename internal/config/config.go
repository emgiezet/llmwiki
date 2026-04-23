package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/emgiezet/llmwiki/internal/validation"
	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	WikiRoot          string `yaml:"wiki_root"`
	LLM               string `yaml:"llm"`
	OllamaHost        string `yaml:"ollama_host"`
	AllowRemoteOllama bool   `yaml:"allow_remote_ollama"`
	AnthropicAPIKey   string `yaml:"anthropic_api_key"`
	MemoryEnabled     bool   `yaml:"memory_enabled"`
	MemoryDir         string `yaml:"memory_dir"`
	// ClaudeBinaryPath overrides the default PATH lookup for the 'claude' binary.
	// Useful when the Claude Code CLI is installed in a non-standard location or
	// when PATH hijacking is a concern. Empty (default) = look up 'claude' via PATH.
	ClaudeBinaryPath string `yaml:"claude_binary_path"`
}

type ProjectConfig struct {
	LLM         string           `yaml:"llm"`
	OllamaModel string           `yaml:"ollama_model"`
	Customer    string           `yaml:"customer"`
	Type        string           `yaml:"type"` // client | personal | oss
	Extraction  ExtractionConfig `yaml:"extraction,omitempty"`
}

// ExtractionConfig controls which markdown sections the LLM is asked to
// produce and how many output tokens the call is allowed to spend.
//
// Sections wins over Preset when both are set; empty means "preset default".
type ExtractionConfig struct {
	// Preset names a bundle from ingestion.Presets (e.g. "minimal", "software",
	// "feature", "full"). Empty means the "default" preset.
	Preset string `yaml:"preset,omitempty"`
	// Sections lists section IDs explicitly; when non-empty it replaces Preset.
	Sections []string `yaml:"sections,omitempty"`
	// MaxTokens caps LLM output per call. 0 means backend default (Claude API
	// falls back to 8192; Ollama and claude-code leave the limit to the backend).
	MaxTokens int `yaml:"max_tokens,omitempty"`
}

// Merged holds resolved config (global defaults + project overrides)
type Merged struct {
	WikiRoot          string
	LLM               string
	OllamaHost        string
	OllamaModel       string
	AllowRemoteOllama bool
	AnthropicAPIKey   string
	Customer          string
	Type              string
	MemoryEnabled     bool
	MemoryDir         string
	ClaudeBinaryPath  string
	Extraction        ExtractionConfig
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return home
}

func LoadGlobalConfig(path string) (GlobalConfig, error) {
	cfg := GlobalConfig{
		LLM:        "claude-code",
		OllamaHost: "http://localhost:11434",
		WikiRoot:   filepath.Join(homeDir(), "llmwiki", "wiki"),
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.AnthropicAPIKey != "" {
		fmt.Fprintln(os.Stderr, "warning: anthropic_api_key stored in plaintext in ~/.llmwiki/config.yaml — prefer the ANTHROPIC_API_KEY environment variable")
	}
	return cfg, nil
}

func LoadProjectConfig(projectDir string) (ProjectConfig, error) {
	var cfg ProjectConfig
	data, err := os.ReadFile(filepath.Join(projectDir, "llmwiki.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if err := validation.NameComponentOptional("customer", cfg.Customer); err != nil {
		return cfg, fmt.Errorf("llmwiki.yaml: %w", err)
	}
	if err := validation.NameComponentOptional("type", cfg.Type); err != nil {
		return cfg, fmt.Errorf("llmwiki.yaml: %w", err)
	}
	return cfg, nil
}

// Merge lives in merge.go — see that file for the three-way resolver.

// DefaultGlobalConfigPath returns ~/.llmwiki/config.yaml
func DefaultGlobalConfigPath() string {
	return filepath.Join(homeDir(), ".llmwiki", "config.yaml")
}
