package llm

import "context"

type ClaudeCodeLLM struct{}

func NewClaudeCodeLLM() LLM { return &ClaudeCodeLLM{} }

func (l *ClaudeCodeLLM) Generate(ctx context.Context, prompt string) (string, error) {
	panic("not yet implemented")
}
