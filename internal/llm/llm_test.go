package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestNewLLM_ClaudeAPIBackend_MissingKey(t *testing.T) {
	_, err := llm.NewLLM(llm.Config{Backend: "claude-api", AnthropicAPIKey: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
}

func TestNewLLM_ClaudeAPIBackend_WithKey(t *testing.T) {
	l, err := llm.NewLLM(llm.Config{Backend: "claude-api", AnthropicAPIKey: "sk-test"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}

func TestOllamaLLM_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
			Stream bool   `json:"stream"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "llama3.2", req.Model)
		assert.False(t, req.Stream)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": "## Domain\nOllama response."})
	}))
	defer server.Close()

	l := llm.NewOllamaLLM(server.URL, "llama3.2")
	result, err := l.Generate(context.Background(), "describe this project")
	require.NoError(t, err)
	assert.Equal(t, "## Domain\nOllama response.", result)
}
