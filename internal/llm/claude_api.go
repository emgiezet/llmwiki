package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type ClaudeAPILLM struct{ client anthropic.Client }

func NewClaudeAPILLM(apiKey string) LLM {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &ClaudeAPILLM{client: client}
}

func (l *ClaudeAPILLM) Generate(ctx context.Context, prompt string) (string, error) {
	msg, err := l.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("claude API returned empty response")
	}
	return msg.Content[0].Text, nil
}
