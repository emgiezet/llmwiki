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

// MemoryModeProject stores llmwiki facts in {projectDir}/.graymatter — one
// database per project, no cross-project lock contention. This is the
// default and aligns with how the graymatter MCP server works.
const MemoryModeProject = "project"

// MemoryModeGlobal stores all facts in a single shared store (cfg.MemoryDir).
// Enables cross-project recall but introduces a global bbolt lock that
// serialises concurrent agents / MCP servers.
const MemoryModeGlobal = "global"

type GlobalConfig struct {
	WikiRoot          string `yaml:"wiki_root"`
	LLM               string `yaml:"llm"`
	OllamaHost        string `yaml:"ollama_host"`
	AllowRemoteOllama bool   `yaml:"allow_remote_ollama"`
	AnthropicAPIKey   string `yaml:"anthropic_api_key"`
	MemoryEnabled     bool   `yaml:"memory_enabled"`
	MemoryDir         string `yaml:"memory_dir"`
	// MemoryMode controls where facts are stored: "project" (default, per-project
	// .graymatter/ directory) or "global" (single shared MemoryDir store).
	MemoryMode string `yaml:"memory_mode"`
	// ClaudeBinaryPath overrides the default PATH lookup for the 'claude' binary.
	// Useful when the Claude Code CLI is installed in a non-standard location or
	// when PATH hijacking is a concern. Empty (default) = look up 'claude' via PATH.
	ClaudeBinaryPath string `yaml:"claude_binary_path"`
	// Optional PATH overrides for the agentic-coder CLIs added in v1.1.0.
	// Empty = look up by basename.
	GeminiBinaryPath   string `yaml:"gemini_binary_path"`
	CodexBinaryPath    string `yaml:"codex_binary_path"`
	OpencodeBinaryPath string `yaml:"opencode_binary_path"`
	PiBinaryPath       string `yaml:"pi_binary_path"`
}

type ProjectConfig struct {
	LLM         string           `yaml:"llm"`
	OllamaModel string           `yaml:"ollama_model"`
	Customer    string           `yaml:"customer"`
	Type        string           `yaml:"type"` // client | personal | oss
	Extraction  ExtractionConfig `yaml:"extraction,omitempty"`
	// v1.3.0 richer metadata — all optional, inherited from ClientConfig.
	Status ProjectStatus `yaml:"status,omitempty"`
	Links  LinksConfig   `yaml:"links,omitempty"`
	Team   TeamConfig    `yaml:"team,omitempty"`
	Cost   CostConfig    `yaml:"cost,omitempty"`
	// v1.4.0 per-project memory dir override. When set, this path is used
	// instead of the mode-derived default. Useful for git worktrees that want
	// to share memory with the main checkout:
	//   memory_dir: /path/to/main-checkout/.graymatter
	MemoryDir string `yaml:"memory_dir,omitempty"`
}

// ProjectStatus classifies where a project sits in its lifecycle. It
// influences which wiki sections the LLM is asked to produce (production
// projects get Architecture + Services; discovery projects get
// Requirements + Open Questions; POCs get a lighter shape). Empty is
// treated as "production" to preserve pre-v1.3.0 behaviour.
type ProjectStatus string

const (
	StatusProduction ProjectStatus = "production"
	StatusPOC        ProjectStatus = "poc"
	StatusDiscovery  ProjectStatus = "discovery"
)

// LinksConfig is a flat map of label → URL. Well-known keys
// (github/gitlab/jira/confluence/clickup/trello/notion/linear/slack/wiki)
// get nicer labels + icons when rendered to markdown; unknown keys pass
// through as generic links. The rendering lives in internal/wiki.
type LinksConfig map[string]string

// TeamConfig captures who owns / supports / escalates a project. Every
// field is optional; the wiki section is rendered only if at least one
// field is set.
type TeamConfig struct {
	Lead          string `yaml:"lead,omitempty"`
	OncallChannel string `yaml:"oncall_channel,omitempty"`
	Escalation    string `yaml:"escalation,omitempty"`
	Notes         string `yaml:"notes,omitempty"`
}

// CostConfig carries infra + team cost figures for a project. Numbers are
// optional; when partial or absent, the wiki renders a how-to-estimate
// template rather than a calculation.
type CostConfig struct {
	InfraMonthlyUSD       float64 `yaml:"infra_monthly_usd,omitempty"`
	TeamFTE               float64 `yaml:"team_fte,omitempty"`
	TeamFTERateMonthlyUSD float64 `yaml:"team_fte_rate_usd_monthly,omitempty"`
	Notes                 string  `yaml:"notes,omitempty"`
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

// Merged holds resolved config (global defaults + client baseline + project overrides)
type Merged struct {
	WikiRoot           string
	LLM                string
	OllamaHost         string
	OllamaModel        string
	AllowRemoteOllama  bool
	AnthropicAPIKey    string
	Customer           string
	Type               string
	MemoryEnabled      bool
	MemoryMode         string // "project" | "global"; default "project"
	MemoryDir          string // global fallback store path
	ProjectMemoryDir   string // per-project override from llmwiki.yaml (worktree use-case)
	ClaudeBinaryPath   string
	GeminiBinaryPath   string
	CodexBinaryPath    string
	OpencodeBinaryPath string
	PiBinaryPath       string
	Extraction         ExtractionConfig
	// v1.3.0 richer project metadata, three-way merged.
	Status ProjectStatus
	Links  LinksConfig
	Team   TeamConfig
	Cost   CostConfig
	// Source tracks, per field, whether the merged value came from the
	// client baseline. The wiki renderer uses this to annotate inherited
	// values with "*(inherited from client)*" so users can tell project-
	// specific settings from org-wide defaults.
	Source MergedSource
}

// MergedSource flags which fields of Merged came from the client baseline
// (vs global or project). Only the fields that the wiki renderer surfaces
// are tracked; extend as new inherited fields are added.
type MergedSource struct {
	LinksFromClient      map[string]bool // per-key: true = inherited from client
	TeamLeadFromClient   bool
	TeamOncallFromClient bool
	TeamEscFromClient    bool
	TeamNotesFromClient  bool
	CostFromClient       bool // true if any cost field came from client
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
	if err := ValidateMemoryMode(cfg.MemoryMode); err != nil {
		return cfg, fmt.Errorf("~/.llmwiki/config.yaml: %w", err)
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
	if err := ValidateStatus(cfg.Status); err != nil {
		return cfg, fmt.Errorf("llmwiki.yaml: %w", err)
	}
	if err := ValidateCost(cfg.Cost); err != nil {
		return cfg, fmt.Errorf("llmwiki.yaml: %w", err)
	}
	for _, w := range ValidateLinks(cfg.Links) {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return cfg, nil
}

// Merge lives in merge.go — see that file for the three-way resolver.

// DefaultGlobalConfigPath returns ~/.llmwiki/config.yaml
func DefaultGlobalConfigPath() string {
	return filepath.Join(homeDir(), ".llmwiki", "config.yaml")
}
