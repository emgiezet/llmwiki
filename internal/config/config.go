package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mgz/llmwiki/internal/validation"
	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	WikiRoot        string `yaml:"wiki_root"`
	LLM             string `yaml:"llm"`
	OllamaHost      string `yaml:"ollama_host"`
	AnthropicAPIKey string `yaml:"anthropic_api_key"`
	MemoryEnabled   bool   `yaml:"memory_enabled"`
	MemoryDir       string `yaml:"memory_dir"`
}

type ProjectConfig struct {
	LLM         string `yaml:"llm"`
	OllamaModel string `yaml:"ollama_model"`
	Customer    string `yaml:"customer"`
	Type        string `yaml:"type"` // client | personal | oss
}

// Merged holds resolved config (global defaults + project overrides)
type Merged struct {
	WikiRoot        string
	LLM             string
	OllamaHost      string
	OllamaModel     string
	AnthropicAPIKey string
	Customer        string
	Type            string
	MemoryEnabled   bool
	MemoryDir       string
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
	return cfg, yaml.Unmarshal(data, &cfg)
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

func Merge(g GlobalConfig, p ProjectConfig) Merged {
	memDir := g.MemoryDir
	if memDir == "" {
		memDir = filepath.Join(homeDir(), ".llmwiki", "memory")
	}
	m := Merged{
		WikiRoot:        g.WikiRoot,
		LLM:             g.LLM,
		OllamaHost:      g.OllamaHost,
		AnthropicAPIKey: g.AnthropicAPIKey,
		Customer:        p.Customer,
		Type:            p.Type,
		OllamaModel:     p.OllamaModel,
		MemoryEnabled:   g.MemoryEnabled,
		MemoryDir:       memDir,
	}
	if p.LLM != "" {
		m.LLM = p.LLM
	}
	return m
}

// DefaultGlobalConfigPath returns ~/.llmwiki/config.yaml
func DefaultGlobalConfigPath() string {
	return filepath.Join(homeDir(), ".llmwiki", "config.yaml")
}
