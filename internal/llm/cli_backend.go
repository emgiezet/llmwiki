package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// cliBackend is a shared implementation for LLM backends that invoke an
// external CLI subprocess (gemini-cli, codex, opencode, pi).
//
// Differences between tools are encoded in argsFn (how to translate a prompt
// into argv) and useStdin (whether to pipe the prompt to stdin instead of or
// in addition to argv). The rest — timeout enforcement, stderr redaction,
// output trimming — is identical to the existing claude-code backend.
type cliBackend struct {
	// name is the user-facing backend name, embedded in error messages.
	name string
	// binary is the resolved path to the CLI executable.
	binary string
	// argsFn maps a prompt to the argv tail (i.e. everything after the binary).
	argsFn func(prompt string) []string
	// useStdin, when true, pipes the prompt to stdin of the subprocess. Some
	// CLIs (like pi) accept prompts via both argv and stdin; we default to
	// argv unless useStdin is set. Prompts long enough to blow past argv
	// limits would need this toggle.
	useStdin bool
}

// Generate shells out to the configured CLI, passing the prompt according to
// argsFn / useStdin, and returns trimmed stdout. On non-zero exit, the error
// includes the last 512 bytes of stderr — enough to diagnose without leaking
// large secrets or prompts.
func (b *cliBackend) Generate(ctx context.Context, prompt string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	args := b.argsFn(prompt)
	cmd := exec.CommandContext(ctx, b.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if b.useStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %w\nstderr (last 512 bytes): %s",
			b.name, err, tailString(stderr.String(), 512))
	}
	return strings.TrimSpace(stdout.String()), nil
}
