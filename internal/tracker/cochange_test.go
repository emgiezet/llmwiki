package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeCommits creates n CommitFiles each containing the given files.
func makeCommits(n int, files []string) []CommitFiles {
	commits := make([]CommitFiles, n)
	for i := range commits {
		commits[i] = CommitFiles{
			Hash:  "deadbeef",
			Files: files,
		}
	}
	return commits
}

func TestClusterer_belowMinCommits_usesHeuristic(t *testing.T) {
	runner := &fakeGitRunner{
		commits: makeCommits(5, []string{"internal/auth/a.go", "internal/billing/b.go"}),
	}
	c := NewClusterer(runner)
	areas, err := c.Cluster("/repo", []string{"internal/auth/a.go", "internal/billing/b.go"})
	require.NoError(t, err)
	for _, a := range areas {
		assert.Equal(t, "scanner-heuristic", a.ClusterMethod)
	}
}

func TestClusterer_highCoChange_mergesFiles(t *testing.T) {
	files := []string{"pkg/a.go", "pkg/b.go"}
	runner := &fakeGitRunner{
		commits: makeCommits(25, files),
	}
	c := NewClusterer(runner)
	areas, err := c.Cluster("/repo", files)
	require.NoError(t, err)
	assert.Len(t, areas, 1)
	assert.Equal(t, "git-cochange", areas[0].ClusterMethod)
	assert.Contains(t, areas[0].Files, "pkg/a.go")
	assert.Contains(t, areas[0].Files, "pkg/b.go")
}

func TestClusterer_lowCoChange_keepsFilesSeparate(t *testing.T) {
	// 25 commits alternating: even have ["pkg/a.go"], odd have ["pkg/b.go"]
	commits := make([]CommitFiles, 25)
	for i := range commits {
		if i%2 == 0 {
			commits[i] = CommitFiles{Hash: "even", Files: []string{"pkg/a.go"}}
		} else {
			commits[i] = CommitFiles{Hash: "odd", Files: []string{"pkg/b.go"}}
		}
	}
	runner := &fakeGitRunner{commits: commits}
	c := NewClusterer(runner)
	areas, err := c.Cluster("/repo", []string{"pkg/a.go", "pkg/b.go"})
	require.NoError(t, err)
	assert.Len(t, areas, 2)
}

func TestFallbackAreas_groupsByTopDir(t *testing.T) {
	files := []string{"internal/auth/a.go", "internal/auth/b.go", "cmd/main.go"}
	areas := fallbackAreas(files)
	names := make([]string, len(areas))
	for i, a := range areas {
		names[i] = a.Name
	}
	assert.Contains(t, names, "internal")
	assert.Contains(t, names, "cmd")
	assert.Len(t, areas, 2)
}
