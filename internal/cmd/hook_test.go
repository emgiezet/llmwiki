package cmd_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookInstall_WritesPluginStructure(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "llmwiki")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "--plugin-dir", pluginDir})
	require.NoError(t, root.Execute())

	manifestData, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	require.NoError(t, err)
	assert.Contains(t, string(manifestData), `"name"`)

	hooksData, err := os.ReadFile(filepath.Join(pluginDir, "hooks", "hooks.json"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksData), "Stop")
	assert.Contains(t, string(hooksData), "${CLAUDE_PLUGIN_ROOT}")

	scriptData, err := os.ReadFile(filepath.Join(pluginDir, "hooks", "stop-hook.py"))
	require.NoError(t, err)
	assert.Contains(t, string(scriptData), "llmwiki absorb")
	assert.Contains(t, string(scriptData), "MIN_RESPONSE_CHARS")
}

func TestHookInstall_Idempotent(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "llmwiki")

	for i := 0; i < 2; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", "--plugin-dir", pluginDir})
		require.NoError(t, root.Execute())
	}

	_, err := os.Stat(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	require.NoError(t, err, "plugin manifest must exist after double install")
}

func TestHookUninstall_RemovesPluginDir(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "llmwiki")
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	require.NoError(t, os.MkdirAll(manifestDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{"name":"llmwiki"}`), 0644))

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "--plugin-dir", pluginDir})
	require.NoError(t, root.Execute())

	_, err := os.Stat(pluginDir)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "plugin dir must be removed after uninstall")
}

func TestHookStatus_Installed(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "llmwiki")
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	require.NoError(t, os.MkdirAll(manifestDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{"name":"llmwiki"}`), 0644))

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status", "--plugin-dir", pluginDir})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "installed")
}

func TestHookStatus_NotInstalled(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "llmwiki-nonexistent")

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status", "--plugin-dir", pluginDir})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "not installed")
}
