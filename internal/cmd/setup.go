package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/wizard"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// isInteractive reports whether stdin is a terminal. It is a package-level var
// so tests can override it. Detection uses ModeCharDevice (no extra deps).
var isInteractive = func() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func toolStatus(name string) string {
	if _, err := exec.LookPath(name); err == nil {
		return "✓ found"
	}
	return "✗ not found"
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func firstWord(s string) string {
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}

// NewSetupCmd builds the `llmwiki setup` command — an interactive wizard for
// the global ~/.llmwiki/config.yaml.
func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactively configure global llmwiki settings (~/.llmwiki/config.yaml)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isInteractive() {
				return errors.New("setup requires an interactive terminal; edit ~/.llmwiki/config.yaml manually")
			}
			path := config.DefaultGlobalConfigPath()
			cfg, err := config.LoadGlobalConfig(path)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			p := wizard.New(cmd.InOrStdin(), cmd.OutOrStdout())
			if !runSetupWizard(p, &cfg) {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes made")
				return nil
			}
			if err := saveGlobalConfig(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ %s\n", path)
			return nil
		},
	}
}

// runSetupWizard mutates cfg via prompts and returns true if the user
// confirmed the final save.
func runSetupWizard(p *wizard.Prompter, cfg *config.GlobalConfig) bool {
	p.Note("Detected tools:")
	p.Note("  claude: %s", toolStatus("claude"))
	p.Note("  ollama: %s", toolStatus("ollama"))

	cfg.LLM = p.Choice("LLM backend?", []wizard.Option{
		{Value: "claude-code", Label: "claude-code (Claude Code subscription)"},
		{Value: "claude-api", Label: "claude-api (uses ANTHROPIC_API_KEY)"},
		{Value: "ollama", Label: "ollama (local models)"},
	}, orDefault(cfg.LLM, "claude-code"))

	switch cfg.LLM {
	case "claude-api":
		p.Note("Prefer the ANTHROPIC_API_KEY env var; a value entered here is stored in plaintext.")
		cfg.AnthropicAPIKey = p.Text("Anthropic API key (empty = use env)", cfg.AnthropicAPIKey)
	case "ollama":
		cfg.OllamaHost = p.Text("Ollama host", orDefault(cfg.OllamaHost, "http://localhost:11434"))
		p.Note("The Ollama model is set per-project via `llmwiki init`.")
	}

	cfg.WikiRoot = p.Text("Wiki root", cfg.WikiRoot)

	cfg.MemoryEnabled = p.Confirm("Enable memory (graymatter)?", cfg.MemoryEnabled)
	if cfg.MemoryEnabled {
		cfg.MemoryMode = p.Choice("Memory mode?", []wizard.Option{
			{Value: "project", Label: "project (per-project store, default)"},
			{Value: "global", Label: "global (single shared store)"},
		}, orDefault(cfg.MemoryMode, "project"))
	}

	p.Note("Document extractors (detection only — edit `extractors` to change):")
	for ext, cmdTmpl := range cfg.Extractors {
		tool := firstWord(cmdTmpl)
		p.Note("  %-6s %s [%s]", ext, tool, toolStatus(tool))
	}

	p.Note("")
	p.Note("Summary:")
	p.Note("  llm:            %s", cfg.LLM)
	p.Note("  wiki_root:      %s", cfg.WikiRoot)
	p.Note("  memory_enabled: %v", cfg.MemoryEnabled)
	if cfg.MemoryEnabled {
		p.Note("  memory_mode:    %s", orDefault(cfg.MemoryMode, "project"))
	}
	return p.Confirm("Save to ~/.llmwiki/config.yaml?", true)
}

// saveGlobalConfig marshals the whole GlobalConfig (preserving unmanaged
// fields such as binary paths) and writes it to path.
func saveGlobalConfig(path string, cfg config.GlobalConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // #nosec G301
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
