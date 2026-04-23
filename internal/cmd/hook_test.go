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

// hookTestHome points the hook installer at a temp directory by overriding
// $HOME — the installers resolve `~/.claude/plugins/llmwiki` via
// os.UserHomeDir() which consults HOME on unix.
func hookTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestHookInstall_ClaudeCode_WritesPluginStructure(t *testing.T) {
	home := hookTestHome(t)
	pluginDir := filepath.Join(home, ".claude", "plugins", "llmwiki")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "claude-code"})
	require.NoError(t, root.Execute())

	manifestData, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	require.NoError(t, err)
	assert.Contains(t, string(manifestData), `"name"`)
	assert.Contains(t, string(manifestData), `"1.1.0"`, "plugin.json should carry the 1.1.0 version")

	hooksData, err := os.ReadFile(filepath.Join(pluginDir, "hooks", "hooks.json"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksData), "Stop")
	assert.Contains(t, string(hooksData), "${CLAUDE_PLUGIN_ROOT}")
	assert.Contains(t, string(hooksData), "node ", "hooks.json must invoke node, not python")
	assert.NotContains(t, string(hooksData), "python", "hooks.json must not reference python anywhere")

	scriptData, err := os.ReadFile(filepath.Join(pluginDir, "hooks", "stop-hook.js"))
	require.NoError(t, err)
	assert.Contains(t, string(scriptData), "llmwiki")
	assert.Contains(t, string(scriptData), "MIN_RESPONSE_CHARS")
	assert.Contains(t, string(scriptData), "#!/usr/bin/env node")

	// And the .py hook should never be written any more.
	_, err = os.Stat(filepath.Join(pluginDir, "hooks", "stop-hook.py"))
	assert.True(t, errors.Is(err, fs.ErrNotExist), "stop-hook.py must not be written (migrated to .js)")
}

func TestHookInstall_ClaudeCode_MigratesPythonToNode(t *testing.T) {
	home := hookTestHome(t)
	pluginDir := filepath.Join(home, ".claude", "plugins", "llmwiki")
	hooksDir := filepath.Join(pluginDir, "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0o755))

	// Simulate a pre-1.1.0 install leaving a stale stop-hook.py behind.
	legacy := filepath.Join(hooksDir, "stop-hook.py")
	require.NoError(t, os.WriteFile(legacy, []byte("#!/usr/bin/env python3\n# legacy hook\n"), 0o755))

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "claude-code"})
	require.NoError(t, root.Execute())

	// Migration: .py gone, .js present, hooks.json uses node.
	_, err := os.Stat(legacy)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "legacy stop-hook.py must be removed during migration")

	_, err = os.Stat(filepath.Join(hooksDir, "stop-hook.js"))
	require.NoError(t, err, "stop-hook.js must exist after migration")

	hooksData, err := os.ReadFile(filepath.Join(hooksDir, "hooks.json"))
	require.NoError(t, err)
	assert.Contains(t, string(hooksData), "node ")
	assert.NotContains(t, string(hooksData), "python")
}

func TestHookInstall_ClaudeCode_Idempotent(t *testing.T) {
	hookTestHome(t)

	for i := 0; i < 2; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", "claude-code"})
		require.NoError(t, root.Execute(), "install call %d must succeed", i+1)
	}
}

func TestHookUninstall_ClaudeCode_RemovesPluginDir(t *testing.T) {
	home := hookTestHome(t)
	pluginDir := filepath.Join(home, ".claude", "plugins", "llmwiki")
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	require.NoError(t, os.MkdirAll(manifestDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{"name":"llmwiki"}`), 0o644))

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "claude-code"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(pluginDir)
	assert.True(t, errors.Is(err, fs.ErrNotExist), "plugin dir must be removed after uninstall")
}

func TestHookStatus_ListsAllToolsWithState(t *testing.T) {
	home := hookTestHome(t)
	// Install claude-code so it's reported as installed; other tools are stubs
	// (not-yet-implemented Install), so status should report them not-installed.
	pluginDir := filepath.Join(home, ".claude", "plugins", "llmwiki")
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	require.NoError(t, os.MkdirAll(manifestDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{"name":"llmwiki"}`), 0o644))

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status"})
	require.NoError(t, root.Execute())

	out := buf.String()
	// Every registered tool should appear in the table.
	for _, name := range []string{"claude-code", "codex", "opencode", "pi", "gemini-cli"} {
		assert.Contains(t, out, name, "status output must mention %q", name)
	}
	assert.Contains(t, out, "yes", "at least one tool should be reported installed (claude-code)")
}

func TestHookInstall_UnknownToolErrorsWithValidList(t *testing.T) {
	hookTestHome(t)

	root := buildTestRoot()
	root.SilenceUsage = true
	root.SetArgs([]string{"hook", "install", "typo"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
	// The error must list the five valid tool names so users can recover.
	for _, valid := range []string{"claude-code", "codex", "opencode", "pi", "gemini-cli"} {
		assert.Contains(t, err.Error(), valid, "error must list %q as a valid tool", valid)
	}
}
