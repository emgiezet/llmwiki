package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

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

exit 0
`

func NewInitCmd() *cobra.Command {
	var customer, projectType string
	var noGraymatter bool

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

			if err := writeProjectConfig(abs, customer, projectType); err != nil {
				return err
			}
			fmt.Printf("✓ %s\n", filepath.Join(abs, "llmwiki.yaml"))

			if !noGraymatter {
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

			return nil
		},
	}

	c.Flags().StringVar(&customer, "customer", "", "customer name (e.g. acme)")
	c.Flags().StringVar(&projectType, "type", "client", "project type: client|personal|oss")
	c.Flags().BoolVar(&noGraymatter, "no-graymatter", false, "skip graymatter hook installation")
	return c
}

func writeProjectConfig(projectDir, customer, projectType string) error {
	path := filepath.Join(projectDir, "llmwiki.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("llmwiki.yaml already exists in %s", projectDir)
	}

	type projectYAML struct {
		Customer string `yaml:"customer,omitempty"`
		Type     string `yaml:"type,omitempty"`
	}
	data, err := yaml.Marshal(projectYAML{Customer: customer, Type: projectType})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func graymatterInstalled() bool {
	_, err := exec.LookPath("graymatter")
	return err == nil
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
