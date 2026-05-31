package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGitRunner implements GitRunner for tests.
// lsTreeResults maps "projectRoot|file" → ls-tree output line.
type fakeGitRunner struct {
	lsTreeResults map[string]string
	commits       []CommitFiles
}

func (f *fakeGitRunner) LogFiles(_ string) ([]CommitFiles, error) { return f.commits, nil }
func (f *fakeGitRunner) LSTree(projectRoot, file string) (string, error) {
	return f.lsTreeResults[projectRoot+"|"+file], nil
}

func TestAreaName_commonPrefix(t *testing.T) {
	files := []string{"internal/auth/handler.go", "internal/auth/middleware.go"}
	result := areaName(files)
	assert.Equal(t, "internal/auth", result)
}

func TestAreaName_singleFile(t *testing.T) {
	files := []string{"main.go"}
	result := areaName(files)
	assert.Equal(t, ".", result)
}

func TestAreaName_noCommonDir(t *testing.T) {
	files := []string{"alpha/a.go", "beta/b.go"}
	result := areaName(files)
	assert.Equal(t, ".", result)
}

func TestComputeHash_deterministicAndStable(t *testing.T) {
	runner := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob abc123 a.go",
			"root|b.go": "100644 blob def456 b.go",
		},
	}

	files1 := []string{"a.go", "b.go"}
	files2 := []string{"b.go", "a.go"} // different order

	hash1, err := ComputeHash(runner, "root", files1)
	require.NoError(t, err)

	hash2, err := ComputeHash(runner, "root", files2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "hash must be the same regardless of input order")
	assert.Len(t, hash1, 16, "hash must be 16 characters")
}

func TestComputeHash_changesWhenContentChanges(t *testing.T) {
	runner1 := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob abc123 a.go",
		},
	}
	runner2 := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob 999999 a.go", // different blob hash
		},
	}

	hash1, err := ComputeHash(runner1, "root", []string{"a.go"})
	require.NoError(t, err)

	hash2, err := ComputeHash(runner2, "root", []string{"a.go"})
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "different ls-tree output must produce different hash")
}
