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

func codexTestSetup(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestCodexHook_Install_WritesWrapperAndAppendsNotify(t *testing.T) {
	home := codexTestSetup(t)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "codex"})
	require.NoError(t, root.Execute())

	wrapper := filepath.Join(home, ".llmwiki", "hooks", "codex-absorb.js")
	data, err := os.ReadFile(wrapper)
	require.NoError(t, err)
	assert.Contains(t, string(data), "llmwiki", "wrapper must call llmwiki")
	assert.Contains(t, string(data), "last-assistant-message", "wrapper must parse codex payload")

	cfg := filepath.Join(home, ".codex", "config.toml")
	toml, err := os.ReadFile(cfg)
	require.NoError(t, err)
	got := string(toml)
	assert.Contains(t, got, "# llmwiki:begin codex-notify")
	assert.Contains(t, got, "# llmwiki:end codex-notify")
	assert.Contains(t, got, "notify = [\"node\",")
	assert.Contains(t, got, "codex-absorb.js")
}

func TestCodexHook_Install_PreservesExistingConfig(t *testing.T) {
	home := codexTestSetup(t)
	cfg := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(cfg), 0o755))

	existing := "# user config\n[profile]\nmodel = \"gpt-5\"\n"
	require.NoError(t, os.WriteFile(cfg, []byte(existing), 0o644))

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "codex"})
	require.NoError(t, root.Execute())

	data, _ := os.ReadFile(cfg)
	got := string(data)
	assert.Contains(t, got, "[profile]", "must preserve user's [profile] table")
	assert.Contains(t, got, `model = "gpt-5"`, "must preserve user keys verbatim")
	assert.Contains(t, got, "# llmwiki:begin codex-notify", "must append our marker block")
}

func TestCodexHook_Install_RefusesWhenUserHasTopLevelNotify(t *testing.T) {
	home := codexTestSetup(t)
	cfg := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(cfg), 0o755))

	// User already has notify set at top level — TOML forbids duplicates,
	// and we must refuse rather than silently clobber their config.
	userConfig := "notify = [\"my-custom-notifier\"]\n"
	require.NoError(t, os.WriteFile(cfg, []byte(userConfig), 0o644))

	root := buildTestRoot()
	root.SilenceUsage = true
	root.SetArgs([]string{"hook", "install", "codex"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notify", "error must explain the notify collision")
	assert.Contains(t, err.Error(), "refusing to install", "error must be unambiguous")

	// Original config must remain untouched.
	data, _ := os.ReadFile(cfg)
	assert.Equal(t, userConfig, string(data),
		"user config must not be modified when install refuses")
}

func TestCodexHook_Install_Idempotent(t *testing.T) {
	home := codexTestSetup(t)
	cfg := filepath.Join(home, ".codex", "config.toml")

	for i := 0; i < 3; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", "codex"})
		require.NoError(t, root.Execute(), "install %d must succeed", i+1)
	}

	data, _ := os.ReadFile(cfg)
	assert.Equal(t, 1, strings.Count(string(data), "# llmwiki:begin codex-notify"),
		"marker block should appear exactly once regardless of install count")
}

func TestCodexHook_Uninstall_StripsBlockAndRemovesWrapper(t *testing.T) {
	home := codexTestSetup(t)

	// Install first, then uninstall.
	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "codex"})
	require.NoError(t, root.Execute())

	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "codex"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(filepath.Join(home, ".llmwiki", "hooks", "codex-absorb.js"))
	assert.True(t, errors.Is(err, fs.ErrNotExist), "wrapper must be removed")

	data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	require.NoError(t, err)
	got := string(data)
	assert.NotContains(t, got, "llmwiki:begin codex-notify")
	assert.NotContains(t, got, "codex-absorb.js")
}

func TestCodexHook_Uninstall_PreservesUserConfig(t *testing.T) {
	home := codexTestSetup(t)
	cfg := filepath.Join(home, ".codex", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(cfg), 0o755))
	userKeys := "[profile]\nmodel = \"gpt-5\"\n"
	require.NoError(t, os.WriteFile(cfg, []byte(userKeys), 0o644))

	// Install adds our block.
	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "codex"})
	require.NoError(t, root.Execute())

	// Uninstall strips just our block.
	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "codex"})
	require.NoError(t, root.Execute())

	data, _ := os.ReadFile(cfg)
	got := string(data)
	assert.Contains(t, got, "[profile]", "user's TOML table must survive uninstall")
	assert.Contains(t, got, `model = "gpt-5"`, "user keys must survive uninstall")
	assert.NotContains(t, got, "llmwiki:begin codex-notify", "our markers must be gone")
}
