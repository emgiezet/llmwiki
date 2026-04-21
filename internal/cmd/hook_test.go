package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFakeSettings(t *testing.T, dir string, content map[string]any) string {
	t.Helper()
	claudeDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	path := filepath.Join(claudeDir, "settings.json")
	data, _ := json.MarshalIndent(content, "", "  ")
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestHookInstall_WritesScriptAndModifiesSettings(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := writeFakeSettings(t, homeDir, map[string]any{
		"model": "sonnet",
	})
	scriptDir := filepath.Join(homeDir, ".llmwiki", "hooks")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install",
		"--settings", settingsPath,
		"--script-dir", scriptDir,
	})
	require.NoError(t, root.Execute())

	// Script must exist and contain key phrases.
	scriptPath := filepath.Join(scriptDir, "stop-hook.py")
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "llmwiki absorb")
	assert.Contains(t, string(data), "MIN_RESPONSE_CHARS")

	// settings.json must contain Stop hook entry.
	raw, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))
	// Existing keys must survive.
	assert.Equal(t, "sonnet", settings["model"])
	hooks := settings["hooks"].(map[string]any)
	stopHooks := hooks["Stop"].([]any)
	require.Len(t, stopHooks, 1)
	entry := stopHooks[0].(map[string]any)
	assert.Equal(t, "command", entry["type"])
	assert.Contains(t, entry["command"].(string), "stop-hook.py")
	assert.Equal(t, float64(30), entry["timeout"])
}

func TestHookInstall_Idempotent(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := writeFakeSettings(t, homeDir, map[string]any{})
	scriptDir := filepath.Join(homeDir, ".llmwiki", "hooks")

	for i := 0; i < 2; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install",
			"--settings", settingsPath,
			"--script-dir", scriptDir,
		})
		require.NoError(t, root.Execute())
	}

	raw, _ := os.ReadFile(settingsPath)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))
	stopHooks := settings["hooks"].(map[string]any)["Stop"].([]any)
	assert.Len(t, stopHooks, 1, "double install must not duplicate entry")
}

func TestHookUninstall_RemovesEntry(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := writeFakeSettings(t, homeDir, map[string]any{
		"model": "sonnet",
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{"type": "command", "command": "python3 /home/.llmwiki/hooks/stop-hook.py", "timeout": 30},
			},
		},
	})

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "--settings", settingsPath})
	require.NoError(t, root.Execute())

	raw, _ := os.ReadFile(settingsPath)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(raw, &settings))
	assert.Equal(t, "sonnet", settings["model"])
	if hooks, ok := settings["hooks"].(map[string]any); ok {
		if stop, ok := hooks["Stop"].([]any); ok {
			assert.Empty(t, stop)
		}
	}
}

func TestHookStatus_Installed(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := writeFakeSettings(t, homeDir, map[string]any{
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{"type": "command", "command": "python3 ~/.llmwiki/hooks/stop-hook.py"},
			},
		},
	})

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status", "--settings", settingsPath})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "installed")
}

func TestHookStatus_NotInstalled(t *testing.T) {
	homeDir := t.TempDir()
	settingsPath := writeFakeSettings(t, homeDir, map[string]any{})

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status", "--settings", settingsPath})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "not installed")
}
