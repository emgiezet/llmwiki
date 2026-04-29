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

// ClientConfig is the per-customer baseline that every project under
// `customer: <name>` inherits from. It lives at
// ~/.llmwiki/clients/<customer>.yaml and is loaded by LoadClientConfig.
//
// A missing file is not an error — consumers get a zero-valued ClientConfig
// and the Merge call falls through to global defaults.
//
// v1.3.0 only wires up LLM + Extraction inheritance here; status/links/
// team/cost arrive in a follow-up commit that extends the YAML schema.
type ClientConfig struct {
	// LLM lets a customer pick a default backend for every project without
	// needing each llmwiki.yaml to repeat the same line.
	LLM string `yaml:"llm,omitempty"`
	// Extraction inherits the same way — client-level presets + max_tokens
	// apply by default; projects override per-field.
	Extraction ExtractionConfig `yaml:"extraction,omitempty"`
	// v1.3.0 richer metadata — client-wide defaults for every project.
	Status ProjectStatus `yaml:"status,omitempty"`
	Links  LinksConfig   `yaml:"links,omitempty"`
	Team   TeamConfig    `yaml:"team,omitempty"`
	Cost   CostConfig    `yaml:"cost,omitempty"`
}

// LoadClientConfig looks up the per-customer config file. Returns a
// zero-value ClientConfig (no error) when the file doesn't exist so callers
// don't branch on missing clients.
//
// The `customer` argument is validated like every other path component to
// prevent an empty or traversal-laden value from resolving to an unexpected
// location.
func LoadClientConfig(customer string) (ClientConfig, error) {
	var cfg ClientConfig
	if customer == "" {
		return cfg, nil
	}
	if err := validation.NameComponent("customer", customer); err != nil {
		return cfg, fmt.Errorf("LoadClientConfig: %w", err)
	}
	data, err := os.ReadFile(DefaultClientConfigPath(customer))
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse client config for %q: %w", customer, err)
	}
	if err := ValidateStatus(cfg.Status); err != nil {
		return cfg, fmt.Errorf("client config for %q: %w", customer, err)
	}
	if err := ValidateCost(cfg.Cost); err != nil {
		return cfg, fmt.Errorf("client config for %q: %w", customer, err)
	}
	for _, w := range ValidateLinks(cfg.Links) {
		fmt.Fprintf(os.Stderr, "warning: client config %q: %s\n", customer, w)
	}
	return cfg, nil
}

// DefaultClientConfigPath returns ~/.llmwiki/clients/<customer>.yaml.
// Exported so `llmwiki client init` (v1.3.0) can use the same resolver.
func DefaultClientConfigPath(customer string) string {
	return filepath.Join(homeDir(), ".llmwiki", "clients", customer+".yaml")
}
