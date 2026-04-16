package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	WikiRoot        string `yaml:"wiki_root"`
	LLM             string `yaml:"llm"`
	OllamaHost      string `yaml:"ollama_host"`
	AnthropicAPIKey string `yaml:"anthropic_api_key"`
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
}

func LoadGlobalConfig(path string) (GlobalConfig, error) {
	cfg := GlobalConfig{
		LLM:        "claude-code",
		OllamaHost: "http://localhost:11434",
		WikiRoot:   filepath.Join(os.Getenv("HOME"), "llmwiki", "wiki"),
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
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
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	return cfg, yaml.Unmarshal(data, &cfg)
}

func Merge(g GlobalConfig, p ProjectConfig) Merged {
	m := Merged{
		WikiRoot:        g.WikiRoot,
		LLM:             g.LLM,
		OllamaHost:      g.OllamaHost,
		AnthropicAPIKey: g.AnthropicAPIKey,
		Customer:        p.Customer,
		Type:            p.Type,
		OllamaModel:     p.OllamaModel,
	}
	if p.LLM != "" {
		m.LLM = p.LLM
	}
	return m
}

// DefaultGlobalConfigPath returns ~/.llmwiki/config.yaml
func DefaultGlobalConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".llmwiki", "config.yaml")
}
