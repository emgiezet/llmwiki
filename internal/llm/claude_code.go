package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ClaudeCodeLLM struct {
	binaryPath string
}

// NewClaudeCodeLLM returns a ClaudeCodeLLM that shells out to the 'claude' binary.
// binaryPath overrides the PATH lookup; empty string defaults to "claude".
func NewClaudeCodeLLM(binaryPath string) LLM {
	if binaryPath == "" {
		binaryPath = "claude"
	}
	return &ClaudeCodeLLM{binaryPath: binaryPath}
}

// Generate shells out to `claude -p <prompt>`.
// Requires claude CLI to be installed and authenticated.
func (l *ClaudeCodeLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, l.binaryPath, "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrTail := tailString(stderr.String(), 512)
		return "", fmt.Errorf("claude -p failed: %w\nstderr (last 512 bytes): %s", err, stderrTail)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// tailString returns the last n bytes of s, preceded by an ellipsis if
// truncation happened. Used in error messages to avoid leaking large prompt
// content from subprocess stderr.
func tailString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "...(truncated)..." + s[len(s)-n:]
}
