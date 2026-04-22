package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestOllamaLLM_ContextCancellation verifies that the Ollama client respects
// context cancellation/deadline and does not hang indefinitely on a slow server.
func TestOllamaLLM_ContextCancellation(t *testing.T) {
	// Server that hangs forever (simulates a stuck Ollama instance)
	hanging := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-hanging // block until test ends
	}))
	defer server.Close()
	defer close(hanging)

	l := llm.NewOllamaLLM(server.URL, "llama3.2")

	// Use a short deadline to keep the test fast
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := l.Generate(ctx, "describe this project")
	elapsed := time.Since(start)

	require.Error(t, err, "expected error due to context deadline")
	assert.Less(t, elapsed, 2*time.Second, "should have returned well before the 2-minute client timeout")
}

// TestNewLLM_OllamaRejectsRemoteHost verifies D4: remote Ollama hosts are
// rejected by default and accepted when AllowRemoteOllama is set.
func TestNewLLM_OllamaRejectsRemoteHost(t *testing.T) {
	_, err := llm.NewLLM(llm.Config{
		Backend:    "ollama",
		OllamaHost: "http://169.254.169.254/latest/meta-data/",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loopback")

	// With AllowRemoteOllama the same host is accepted
	l, err := llm.NewLLM(llm.Config{
		Backend:           "ollama",
		OllamaHost:        "http://169.254.169.254/latest/meta-data/",
		AllowRemoteOllama: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, l)
}
