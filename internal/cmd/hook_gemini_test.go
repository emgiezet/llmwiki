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

// geminiTestSetup points both the wrapper dir and the rc-file detection at a
// temp HOME so tests never touch the user's real files.
func geminiTestSetup(t *testing.T, shell string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", shell)
	return home
}

func TestGeminiCLIHook_Install_WritesWrappers(t *testing.T) {
	home := geminiTestSetup(t, "/bin/bash")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	posix := filepath.Join(home, ".llmwiki", "shell", "gemini-wrapper.sh")
	fish := filepath.Join(home, ".llmwiki", "shell", "gemini-wrapper.fish")

	data, err := os.ReadFile(posix)
	require.NoError(t, err)
	assert.Contains(t, string(data), "gemini()", "posix wrapper must define gemini function")
	assert.Contains(t, string(data), "llmwiki absorb", "wrapper must call llmwiki absorb")

	_, err = os.Stat(fish)
	require.NoError(t, err, "fish wrapper must also be written (both shells installed)")
}

func TestGeminiCLIHook_Install_AppendsSourceBlockToBashrc(t *testing.T) {
	home := geminiTestSetup(t, "/bin/bash")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	bashrc := filepath.Join(home, ".bashrc")
	data, err := os.ReadFile(bashrc)
	require.NoError(t, err, ".bashrc must be created / appended to")
	got := string(data)
	assert.Contains(t, got, "# llmwiki:begin gemini-wrapper")
	assert.Contains(t, got, "# llmwiki:end gemini-wrapper")
	assert.Contains(t, got, "gemini-wrapper.sh", "source line must reference the posix wrapper")
}

func TestGeminiCLIHook_Install_AppendsSourceBlockToZshrc(t *testing.T) {
	home := geminiTestSetup(t, "/usr/bin/zsh")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	zshrc := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(zshrc)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# llmwiki:begin gemini-wrapper")
	assert.NotContains(t, string(data), "gemini-wrapper.fish", "zsh rc must reference the POSIX wrapper, not fish")
}

func TestGeminiCLIHook_Install_UsesFishConfigForFishShell(t *testing.T) {
	home := geminiTestSetup(t, "/usr/bin/fish")

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
	data, err := os.ReadFile(fishConfig)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# llmwiki:begin gemini-wrapper")
	assert.Contains(t, string(data), "gemini-wrapper.fish", "fish rc must reference the fish wrapper")
	assert.Contains(t, string(data), "source ", "fish rc should use `source` rather than `.`")
}

func TestGeminiCLIHook_Install_Idempotent(t *testing.T) {
	home := geminiTestSetup(t, "/bin/bash")

	for i := 0; i < 3; i++ {
		root := buildTestRoot()
		root.SetArgs([]string{"hook", "install", "gemini-cli"})
		require.NoError(t, root.Execute(), "install %d must succeed", i+1)
	}

	bashrc := filepath.Join(home, ".bashrc")
	data, err := os.ReadFile(bashrc)
	require.NoError(t, err)
	got := string(data)
	// Exactly one marker block should be present even after multiple installs.
	assert.Equal(t, 1, strings.Count(got, "# llmwiki:begin gemini-wrapper"),
		"marker block should appear exactly once, got:\n%s", got)
}

func TestGeminiCLIHook_Install_PreservesExistingRCContent(t *testing.T) {
	home := geminiTestSetup(t, "/bin/bash")

	bashrc := filepath.Join(home, ".bashrc")
	existing := "# user-managed aliases\nalias ll='ls -lah'\nexport FOO=bar\n"
	require.NoError(t, os.WriteFile(bashrc, []byte(existing), 0o644))

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	data, _ := os.ReadFile(bashrc)
	got := string(data)
	assert.Contains(t, got, "alias ll=", "install must preserve pre-existing aliases")
	assert.Contains(t, got, "export FOO=bar", "install must preserve pre-existing exports")
	assert.Contains(t, got, "# llmwiki:begin gemini-wrapper")
}

func TestGeminiCLIHook_Uninstall_StripsBlocksAndWrapperFiles(t *testing.T) {
	home := geminiTestSetup(t, "/bin/bash")

	// Install first.
	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "gemini-cli"})
	require.NoError(t, root.Execute())

	// Now uninstall.
	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "gemini-cli"})
	require.NoError(t, root.Execute())

	bashrc := filepath.Join(home, ".bashrc")
	data, err := os.ReadFile(bashrc)
	if err == nil {
		got := string(data)
		assert.NotContains(t, got, "llmwiki:begin gemini-wrapper")
		assert.NotContains(t, got, "llmwiki:end gemini-wrapper")
		assert.NotContains(t, got, "gemini-wrapper.sh")
	}

	_, err = os.Stat(filepath.Join(home, ".llmwiki", "shell", "gemini-wrapper.sh"))
	assert.True(t, errors.Is(err, fs.ErrNotExist), "posix wrapper must be removed")
	_, err = os.Stat(filepath.Join(home, ".llmwiki", "shell", "gemini-wrapper.fish"))
	assert.True(t, errors.Is(err, fs.ErrNotExist), "fish wrapper must be removed")
}

func TestGeminiCLIHook_Uninstall_MissingIsHarmless(t *testing.T) {
	geminiTestSetup(t, "/bin/bash")

	// No install first — uninstall should still exit 0.
	root := buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "gemini-cli"})
	require.NoError(t, root.Execute())
}
