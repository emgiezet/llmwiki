package llm_test

import (
	"context"
	"testing"

	"github.com/mgz/llmwiki/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeLLM_Generate(t *testing.T) {
	fake := llm.NewFakeLLM("## Domain\nTest output.")
	result, err := fake.Generate(context.Background(), "describe this project")
	require.NoError(t, err)
	assert.Equal(t, "## Domain\nTest output.", result)
}

func TestNewLLM_UnknownBackend(t *testing.T) {
	_, err := llm.NewLLM(llm.Config{Backend: "unknown"})
	assert.Error(t, err)
}

func TestNewLLM_OllamaBackend(t *testing.T) {
	l, err := llm.NewLLM(llm.Config{Backend: "ollama", OllamaHost: "http://localhost:11434", OllamaModel: "llama3.2"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}

func TestNewLLM_ClaudeCodeBackend(t *testing.T) {
	l, err := llm.NewLLM(llm.Config{Backend: "claude-code"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}
