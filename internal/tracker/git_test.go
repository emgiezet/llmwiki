package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLogOutput_empty(t *testing.T) {
	result := parseLogOutput("")
	assert.Empty(t, result)
}

func TestParseLogOutput_singleCommit(t *testing.T) {
	input := "COMMIT:abc123\nfoo/bar.go\nbaz/qux.go\n"
	result := parseLogOutput(input)
	assert.Len(t, result, 1)
	assert.Equal(t, "abc123", result[0].Hash)
	assert.Equal(t, []string{"foo/bar.go", "baz/qux.go"}, result[0].Files)
}

func TestParseLogOutput_multipleCommits(t *testing.T) {
	input := "COMMIT:aaa\nfoo.go\n\nCOMMIT:bbb\nbar.go\nbaz.go\n"
	result := parseLogOutput(input)
	assert.Len(t, result, 2)
	assert.Equal(t, "aaa", result[0].Hash)
	assert.Equal(t, []string{"foo.go"}, result[0].Files)
	assert.Equal(t, "bbb", result[1].Hash)
	assert.Equal(t, []string{"bar.go", "baz.go"}, result[1].Files)
}
