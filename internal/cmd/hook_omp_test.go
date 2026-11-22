package cmd_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ompTestHome isolates the omp hook: HOME for the default agent dir, and an
// empty PI_CODING_AGENT_DIR so a developer's env can't redirect the write.
func ompTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_CODING_AGENT_DIR", "")
	return home
}

func TestOmpHook_Install_WritesExtension(t *testing.T) {
	home := ompTestHome(t)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "omp"})
	require.NoError(t, root.Execute())

	data, err := os.ReadFile(filepath.Join(home, ".omp", "agent", "extensions", "llmwiki.ts"))
	require.NoError(t, err)
	got := string(data)
	assert.Contains(t, got, "agent_end", "extension must flush on omp's agent_end event")
	assert.Contains(t, got, "ExtensionAPI", "extension must use omp's typed extension API")
	assert.Contains(t, got, "export default", "omp requires a default-exported factory")
	assert.Contains(t, got, "absorb", "extension must forward to llmwiki absorb")
}

func TestOmpHook_Uninstall_RemovesExtension(t *testing.T) {
	home := ompTestHome(t)

	root := buildTestRoot()
	root.SetArgs([]string{"hook", "install", "omp"})
	require.NoError(t, root.Execute())

	root = buildTestRoot()
	root.SetArgs([]string{"hook", "uninstall", "omp"})
	require.NoError(t, root.Execute())

	_, err := os.Stat(filepath.Join(home, ".omp", "agent", "extensions", "llmwiki.ts"))
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}
