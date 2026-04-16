package llm

import "context"

type ClaudeAPILLM struct{ apiKey string }

func NewClaudeAPILLM(apiKey string) LLM { return &ClaudeAPILLM{apiKey: apiKey} }

func (l *ClaudeAPILLM) Generate(ctx context.Context, prompt string) (string, error) {
	panic("not yet implemented")
}
