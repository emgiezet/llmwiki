package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/emgiezet/llmwiki/internal/wizard"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const preCommitHookScript = `#!/bin/sh
# Installed by: llmwiki init --hooks
# Checks if any wiki entries covering staged files are stale.
STAGED=$(git diff --cached --name-only | tr '\n' ',')
if [ -z "$STAGED" ]; then
  exit 0
fi
if command -v llmwiki >/dev/null 2>&1; then
  llmwiki check --exit-code --files "$STAGED" .
fi
`

const graymatterStopScript = `#!/bin/bash
# Passive graymatter memory capture on Claude Code Stop event.
# Installed by: llmwiki init --graymatter
GRAYMATTER_DIR="${GRAYMATTER_HOOK_DIR:-.graymatter}"
AGENT_ID="claude-code"

INPUT=$(cat)
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript_path // ""')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"' | cut -c1-8)

[ -z "$TRANSCRIPT" ] || [ ! -f "$TRANSCRIPT" ] && exit 0

SUMMARY=$(python3 - "$TRANSCRIPT" <<'PYEOF'
import sys, json

transcript = sys.argv[1]
messages = []
try:
    with open(transcript) as f:
        for line in f:
            try:
                d = json.loads(line.strip())
                if d.get('type') not in ('user', 'assistant'):
                    continue
                msg = d.get('message', {})
                role = msg.get('role', '')
                content = msg.get('content', '')
                text = ''
                if isinstance(content, list):
                    for c in content:
                        if isinstance(c, dict) and c.get('type') == 'text':
                            text = c['text'].strip()
                            break
                elif isinstance(content, str):
                    text = content.strip()
                if text and not text.startswith('<') and len(text) > 10:
                    messages.append(f"[{role}]: {text[:300]}")
            except Exception:
                pass
except Exception:
    pass
print('\n'.join(messages[-6:]))
PYEOF
)

[ -z "$SUMMARY" ] && exit 0

graymatter remember "$AGENT_ID" "session=$SESSION_ID
$SUMMARY" --dir "$GRAYMATTER_DIR" --quiet 2>/dev/null

# Run a non-blocking freshness check on files mentioned in this session.
TOUCHED_FILES=$(echo "$SUMMARY" | grep -oE '[a-zA-Z0-9_./-]+\.[a-zA-Z]+' | grep '/' | tr '\n' ',' | sed 's/,$//')
if [ -n "$TOUCHED_FILES" ] && command -v llmwiki >/dev/null 2>&1; then
  llmwiki check --json --files "$TOUCHED_FILES" . 2>/dev/null | \
    graymatter remember "$AGENT_ID" "llmwiki-staleness-check" --dir "$GRAYMATTER_DIR" --quiet 2>/dev/null || true
fi

exit 0
`

func NewInitCmd() *cobra.Command {
	var customer, projectType string
	var noGraymatter bool
	var installHooks bool

	c := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialise a project for llmwiki tracking",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir := "."
			if len(args) == 1 {
				projectDir = args[0]
			}
			abs, err := filepath.Abs(projectDir)
			if err != nil {
				return err
			}

			if err := writeProjectConfig(abs, initOptions{customer: customer, projectType: projectType}, false); err != nil {
				return err
			}
			fmt.Printf("✓ %s\n", filepath.Join(abs, "llmwiki.yaml"))

			installIntegrations(abs, !noGraymatter, installHooks)
			return nil
		},
	}

	c.Flags().StringVar(&customer, "customer", "", "customer name (e.g. acme)")
	c.Flags().StringVar(&projectType, "type", "client", "project type: client|personal|oss")
	c.Flags().BoolVar(&noGraymatter, "no-graymatter", false, "skip graymatter hook installation")
	c.Flags().BoolVar(&installHooks, "hooks", false, "Install a Git pre-commit hook that checks for stale docs")
	return c
}

func installPreCommitHook(projectDir string) error {
	hooksDir := filepath.Join(projectDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")

	if _, err := os.Stat(hookPath); err == nil {
		return nil // already exists — never overwrite
	}

	if err := os.MkdirAll(hooksDir, 0o755); err != nil { // #nosec G301
		return fmt.Errorf("create hooks dir: %w", err)
	}
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript), 0o755); err != nil { // #nosec G306
		return fmt.Errorf("write pre-commit hook: %w", err)
	}
	return nil
}

// initOptions carries the project-config fields the wizard / flags collect.
type initOptions struct {
	customer     string
	projectType  string
	preset       string
	outputMode   string
	localDocsDir string
}

// writeProjectConfig writes llmwiki.yaml from opts. When force is false it
// refuses to overwrite an existing file (preserving the original `init`
// behaviour); the interactive wizard passes force=true for edit mode.
func writeProjectConfig(projectDir string, opts initOptions, force bool) error {
	path := filepath.Join(projectDir, "llmwiki.yaml")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("llmwiki.yaml already exists in %s", projectDir)
		}
	}

	type extractionYAML struct {
		Preset string `yaml:"preset,omitempty"`
	}
	type projectYAML struct {
		Customer     string         `yaml:"customer,omitempty"`
		Type         string         `yaml:"type,omitempty"`
		Extraction   extractionYAML `yaml:"extraction,omitempty"`
		OutputMode   string         `yaml:"output_mode,omitempty"`
		LocalDocsDir string         `yaml:"local_docs_dir,omitempty"`
	}
	data, err := yaml.Marshal(projectYAML{
		Customer:     opts.customer,
		Type:         opts.projectType,
		Extraction:   extractionYAML{Preset: opts.preset},
		OutputMode:   opts.outputMode,
		LocalDocsDir: opts.localDocsDir,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// installIntegrations installs the optional graymatter hook+MCP and/or the
// pre-commit freshness hook, printing progress to stdout/stderr. Shared by the
// flag-driven and wizard-driven init paths.
func installIntegrations(abs string, graymatter, preCommit bool) {
	if graymatter {
		if !graymatterInstalled() {
			fmt.Println("  graymatter not found in PATH — skipping (install from https://github.com/gdgvda/graymatter)")
		} else {
			if err := installGraymatterHook(abs); err != nil {
				fmt.Fprintf(os.Stderr, "warning: graymatter hook: %v\n", err)
			} else {
				fmt.Printf("✓ graymatter Stop hook → %s/.claude/\n", abs)
			}
			if err := installGraymatterMCP(abs); err != nil {
				fmt.Fprintf(os.Stderr, "warning: graymatter MCP: %v\n", err)
			} else {
				fmt.Printf("✓ graymatter MCP server → %s/.mcp.json\n", abs)
			}
		}
	}
	if preCommit {
		if err := installPreCommitHook(abs); err != nil {
			fmt.Fprintf(os.Stderr, "warning: pre-commit hook: %v\n", err)
		} else {
			fmt.Printf("✓ pre-commit hook → %s/.git/hooks/pre-commit\n", abs)
		}
	}
}

func graymatterInstalled() bool {
	_, err := exec.LookPath("graymatter")
	return err == nil
}

// graymatterDetected is graymatterInstalled by default; overridable in tests so
// the wizard's prompt sequence is deterministic regardless of the host PATH.
var graymatterDetected = graymatterInstalled

// installChoices records which optional integrations the user opted into.
type installChoices struct {
	graymatter bool
	preCommit  bool
}

func presetOptions() []wizard.Option {
	return []wizard.Option{
		{Value: "default", Label: "default — full software project sections"},
		{Value: "minimal", Label: "minimal — domain/architecture/features/tags"},
		{Value: "software", Label: "software — architecture, patterns, testing, runbook"},
		{Value: "feature", Label: "feature — features + roadmap focus"},
		{Value: "full", Label: "full — every section"},
		{Value: "notes", Label: "notes — prose, for notes/meeting docs"},
		{Value: "research", Label: "research — prose + references/glossary"},
	}
}

// runInitWizard collects project config + install choices via prompts. existing
// supplies edit-mode defaults (zero value for a fresh project). Returns the
// collected options, install choices, and whether the user confirmed the save.
func runInitWizard(p *wizard.Prompter, existing config.ProjectConfig) (initOptions, installChoices, bool) {
	opts := initOptions{
		customer:     existing.Customer,
		projectType:  orDefault(existing.Type, "client"),
		preset:       existing.Extraction.Preset,
		outputMode:   orDefault(existing.OutputMode, "central"),
		localDocsDir: orDefault(existing.LocalDocsDir, "docs/llmwiki"),
	}

	opts.projectType = p.Choice("Project type?", []wizard.Option{
		{Value: "client", Label: "client"},
		{Value: "personal", Label: "personal"},
		{Value: "oss", Label: "open source"},
	}, opts.projectType)

	if opts.projectType == "client" {
		opts.customer = p.TextValidated("Customer", opts.customer, func(s string) error {
			return validation.NameComponentOptional("customer", s)
		})
	}

	opts.preset = p.Choice("Extraction preset?", presetOptions(), orDefault(opts.preset, "default"))

	opts.outputMode = p.Choice("Output mode?", []wizard.Option{
		{Value: "central", Label: "central (~/llmwiki/wiki only, default)"},
		{Value: "local", Label: "local (project docs dir only)"},
		{Value: "both", Label: "both"},
	}, opts.outputMode)
	if opts.outputMode == "local" || opts.outputMode == "both" {
		opts.localDocsDir = p.Text("Local docs dir", opts.localDocsDir)
	}

	var inst installChoices
	if graymatterDetected() {
		inst.graymatter = p.Confirm("Install graymatter hook + MCP?", true)
	} else {
		p.Note("graymatter not found in PATH — skipping hook/MCP (install from https://github.com/gdgvda/graymatter)")
	}
	inst.preCommit = p.Confirm("Install pre-commit freshness hook?", false)

	p.Note("Optional metadata (links/team/cost/per-project llm) is not prompted — add it manually in llmwiki.yaml; it inherits from client config.")

	p.Note("")
	p.Note("Summary:")
	p.Note("  type:        %s", opts.projectType)
	if opts.projectType == "client" {
		p.Note("  customer:    %s", opts.customer)
	}
	p.Note("  preset:      %s", orDefault(opts.preset, "default"))
	p.Note("  output_mode: %s", opts.outputMode)
	save := p.Confirm("Write llmwiki.yaml?", true)
	return opts, inst, save
}

func installGraymatterHook(projectDir string) error {
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		return fmt.Errorf("create .claude: %w", err)
	}

	scriptPath := filepath.Join(claudeDir, "graymatter_stop.sh")
	if err := os.WriteFile(scriptPath, []byte(graymatterStopScript), 0o755); err != nil { // #nosec G306 -- shell hook must be executable
		return fmt.Errorf("write script: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	settings, err := loadJSONMap(settingsPath)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	graymatterDir := filepath.Join(projectDir, ".graymatter")
	jsonSetPath(settings, graymatterDir, "env", "GRAYMATTER_HOOK_DIR")
	jsonAddStopHook(settings, scriptPath)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(data, '\n'), 0o600)
}

func installGraymatterMCP(projectDir string) error {
	mcpPath := filepath.Join(projectDir, ".mcp.json")
	mcp, err := loadJSONMap(mcpPath)
	if err != nil {
		return fmt.Errorf("load .mcp.json: %w", err)
	}

	servers, _ := mcp["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		mcp["mcpServers"] = servers
	}

	if _, exists := servers["graymatter"]; exists {
		return nil // already configured
	}

	graymatterDir := filepath.Join(projectDir, ".graymatter")
	servers["graymatter"] = map[string]any{
		"command": "graymatter",
		"args":    []any{"mcp", "serve", "--dir", graymatterDir},
	}

	data, err := json.MarshalIndent(mcp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mcpPath, append(data, '\n'), 0o600)
}

func loadJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// jsonSetPath sets m["k1"]["k2"]...= value, creating intermediate maps as needed.
func jsonSetPath(m map[string]any, value any, keys ...string) {
	for _, k := range keys[:len(keys)-1] {
		if _, ok := m[k]; !ok {
			m[k] = map[string]any{}
		}
		if sub, ok := m[k].(map[string]any); ok {
			m = sub
		} else {
			return
		}
	}
	m[keys[len(keys)-1]] = value
}

// jsonAddStopHook appends the graymatter Stop hook entry if not already present.
func jsonAddStopHook(settings map[string]any, scriptPath string) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}

	stopRaw, _ := hooks["Stop"].([]any)

	// Check if our script is already wired
	for _, entry := range stopRaw {
		entryMap, _ := entry.(map[string]any)
		if entryMap == nil {
			continue
		}
		subHooks, _ := entryMap["hooks"].([]any)
		for _, h := range subHooks {
			hMap, _ := h.(map[string]any)
			if hMap != nil && hMap["command"] == scriptPath {
				return // already present
			}
		}
	}

	hookEntry := map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": scriptPath,
				"async":   true,
			},
		},
	}
	hooks["Stop"] = append(stopRaw, hookEntry)
}
