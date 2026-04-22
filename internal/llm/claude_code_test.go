package llm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailString_ShortString_ReturnedUnchanged(t *testing.T) {
	s := "short message"
	got := tailString(s, 512)
	assert.Equal(t, s, got)
}

func TestTailString_LongString_Truncated(t *testing.T) {
	// Build a 2000-character string so it exceeds the 512-byte limit.
	s := strings.Repeat("x", 2000)
	got := tailString(s, 512)

	// Result must fit within prefix + 512 bytes.
	assert.LessOrEqual(t, len(got), len("...(truncated)...")+512)
	// Must contain the truncation marker.
	assert.Contains(t, got, "...(truncated)...")
	// Must end with the last 512 bytes of the original.
	assert.True(t, strings.HasSuffix(got, s[len(s)-512:]))
}

func TestTailString_ExactlyAtLimit_NotTruncated(t *testing.T) {
	s := strings.Repeat("y", 512)
	got := tailString(s, 512)
	assert.Equal(t, s, got)
	assert.NotContains(t, got, "...(truncated)...")
}

// TestClaudeCodeLLM_BinaryPathPlumbed verifies D11: a custom binary path set
// in Config is stored on the constructed ClaudeCodeLLM rather than silently
// ignored.
func TestClaudeCodeLLM_BinaryPathPlumbed(t *testing.T) {
	const customPath = "/usr/local/bin/my-claude"
	l, err := NewLLM(Config{Backend: "claude-code", ClaudeBinaryPath: customPath})
	require.NoError(t, err)
	cc, ok := l.(*ClaudeCodeLLM)
	require.True(t, ok, "expected *ClaudeCodeLLM")
	assert.Equal(t, customPath, cc.binaryPath)
}

func TestClaudeCodeLLM_BinaryPathDefaultsToClaudeBinary(t *testing.T) {
	l, err := NewLLM(Config{Backend: "claude-code"})
	require.NoError(t, err)
	cc, ok := l.(*ClaudeCodeLLM)
	require.True(t, ok, "expected *ClaudeCodeLLM")
	assert.Equal(t, "claude", cc.binaryPath)
}
