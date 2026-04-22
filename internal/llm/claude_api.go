package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type ClaudeAPILLM struct {
	client    anthropic.Client
	maxTokens int64
}

// claudeAPIDefaultMaxTokens is the fallback when callers pass 0.
const claudeAPIDefaultMaxTokens = 8192

func NewClaudeAPILLM(apiKey string, maxTokens int) LLM {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	mt := int64(maxTokens)
	if mt <= 0 {
		mt = claudeAPIDefaultMaxTokens
	}
	return &ClaudeAPILLM{client: client, maxTokens: mt}
}

func (l *ClaudeAPILLM) Generate(ctx context.Context, prompt string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}
	msg, err := l.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_6,
		MaxTokens: l.maxTokens,
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
