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

func TestOpencodeHook_Install_WritesPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "opencode"})
	require.NoError(t, root.Execute())

	plugin := filepath.Join(home, ".config", "opencode", "plugins", "llmwiki.ts")
	data, err := os.ReadFile(plugin)
	require.NoError(t, err)
	got := string(data)
	assert.Contains(t, got, "session.idle", "plugin must subscribe to session.idle")
	assert.Contains(t, got, "llmwiki absorb", "plugin must forward to llmwiki absorb")
	assert.Contains(t, got, "@opencode-ai/plugin", "plugin must import the opencode plugin type")
}

func TestOpencodeHook_Uninstall_RemovesPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "opencode"})
	require.NoError(t, root.Execute())

	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "opencode"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "llmwiki.ts"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestOpencodeHook_Install_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	for i := 0; i < 3; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", "opencode"})
		require.NoError(t, root.Execute(), "install %d must succeed", i+1)
	}

	_, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "llmwiki.ts"))
	require.NoError(t, err, "plugin must exist after idempotent installs")
}

func TestPiHook_Install_WritesExtension(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "pi"})
	require.NoError(t, root.Execute())

	ext := filepath.Join(home, ".pi", "agent", "extensions", "llmwiki.ts")
	data, err := os.ReadFile(ext)
	require.NoError(t, err)
	got := string(data)
	assert.Contains(t, got, "agent_end", "extension must subscribe to pi's agent_end event")
	assert.Contains(t, got, "llmwiki", "extension must reference llmwiki absorb")
	assert.Contains(t, got, "pi.on", "extension must use pi's on() API")
}

func TestPiHook_Uninstall_RemovesExtension(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "pi"})
	require.NoError(t, root.Execute())

	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "pi"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(filepath.Join(home, ".pi", "agent", "extensions", "llmwiki.ts"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestHookStatus_AfterMultipleInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	for _, tool := range []string{"claude-code", "opencode", "pi"} {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", tool})
		require.NoError(t, root.Execute(), "install %s", tool)
	}

	var buf strings.Builder
	root := buildTestRoot()
	root.SetOut(&buf)
	root.SetArgs([]string{"hook", "status"})
	require.NoError(t, root.Execute())

	out := buf.String()
	for _, tool := range []string{"claude-code", "opencode", "pi"} {
		assert.Contains(t, out, tool+" ", "status must mention %s", tool)
	}
	// Codex and gemini-cli weren't installed — they should show "no".
	assert.Contains(t, out, "codex", "status must list codex even when not installed")
	assert.Contains(t, out, "gemini-cli", "status must list gemini-cli even when not installed")
}
