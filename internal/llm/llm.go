package llm

import (
	"context"
	"fmt"
)

// LLM is the interface all backends implement.
type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// Config holds backend selection and credentials.
type Config struct {
	Backend           string // "claude-code" | "claude-api" | "ollama"
	OllamaHost        string
	OllamaModel       string
	AllowRemoteOllama bool
	AnthropicAPIKey   string
	// ClaudeBinaryPath overrides the PATH lookup for the 'claude' binary.
	// Empty (default) = look up 'claude' via PATH.
	ClaudeBinaryPath string
	// MaxTokens caps the backend's output length. 0 means backend default.
	// Claude API falls back to 8192 when zero; Ollama and claude-code pass the
	// limit through only when non-zero.
	MaxTokens int
}

// NewLLM returns the appropriate LLM backend.
func NewLLM(cfg Config) (LLM, error) {
	switch cfg.Backend {
	case "claude-code", "":
		return NewClaudeCodeLLM(cfg.ClaudeBinaryPath), nil
	case "claude-api":
		if cfg.AnthropicAPIKey == "" {
			return nil, fmt.Errorf("claude-api requires ANTHROPIC_API_KEY")
		}
		return NewClaudeAPILLM(cfg.AnthropicAPIKey, cfg.MaxTokens), nil
	case "ollama":
		host := cfg.OllamaHost
		if host == "" {
			host = "http://localhost:11434"
		}
		if err := ValidateOllamaHost(host, cfg.AllowRemoteOllama); err != nil {
			return nil, err
		}
		model := cfg.OllamaModel
		if model == "" {
			model = "llama3.2"
		}
		return NewOllamaLLM(host, model, cfg.MaxTokens), nil
	default:
		return nil, fmt.Errorf("unknown LLM backend: %q (valid: claude-code, claude-api, ollama)", cfg.Backend)
	}
}

// FakeLLM returns a fixed response — used in tests.
type FakeLLM struct{ response string }

func NewFakeLLM(response string) LLM { return &FakeLLM{response: response} }

func (f *FakeLLM) Generate(_ context.Context, _ string) (string, error) {
	return f.response, nil
}
