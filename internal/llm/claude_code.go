package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ClaudeCodeLLM struct{}

func NewClaudeCodeLLM() LLM { return &ClaudeCodeLLM{} }

// Generate shells out to `claude -p <prompt>`.
// Requires claude CLI to be installed and authenticated.
func (l *ClaudeCodeLLM) Generate(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude -p failed: %w\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}
