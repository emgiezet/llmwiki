package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckFreshness_fresh(t *testing.T) {
	runner := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob abc\ta.go",
		},
	}

	// Compute expected hash
	expectedHash, err := ComputeHash(runner, "root", []string{"a.go"})
	require.NoError(t, err)

	area := Area{
		Name:  ".",
		Files: []string{"a.go"},
	}

	result, err := CheckFreshness(runner, "root", area, expectedHash)
	require.NoError(t, err)

	assert.Equal(t, ".", result.AreaName)
	assert.Equal(t, []string{"a.go"}, result.Files)
	assert.Equal(t, expectedHash, result.StoredHash)
	assert.Equal(t, expectedHash, result.CurrentHash)
	assert.False(t, result.IsStale)
}

func TestCheckFreshness_stale(t *testing.T) {
	runner := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob newblob\ta.go",
		},
	}

	area := Area{
		Name:  ".",
		Files: []string{"a.go"},
	}

	result, err := CheckFreshness(runner, "root", area, "oldhash12345678")
	require.NoError(t, err)

	assert.Equal(t, ".", result.AreaName)
	assert.Equal(t, []string{"a.go"}, result.Files)
	assert.Equal(t, "oldhash12345678", result.StoredHash)
	assert.NotEqual(t, "oldhash12345678", result.CurrentHash)
	assert.True(t, result.IsStale)
}

func TestCheckFreshness_emptyStoredHash(t *testing.T) {
	runner := &fakeGitRunner{
		lsTreeResults: map[string]string{
			"root|a.go": "100644 blob abc\ta.go",
		},
	}

	area := Area{
		Name:  ".",
		Files: []string{"a.go"},
	}

	result, err := CheckFreshness(runner, "root", area, "")
	require.NoError(t, err)

	assert.True(t, result.IsStale)
}
