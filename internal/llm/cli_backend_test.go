package llm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emgiezet/llmwiki/internal/llm"
)

// TestCLIBackend_ContextCancellation verifies that a CLI backend respects
// context deadlines. Uses /bin/sh as a stand-in "slow binary" that sleeps
// longer than the context allows, via the shared cliBackend path that all
// agentic-coder CLIs use.
func TestCLIBackend_ContextCancellation(t *testing.T) {
	// We go through NewGeminiCLILLM with a custom binary path pointing at
	// /bin/sh. The argsFn for gemini is `-p <prompt> --output-format text`;
	// sh will interpret -p as... nothing useful. That's fine — sh fails
	// fast with an error, and the test is only checking the ctx-deadline
	// doesn't hang the caller. To actually test slow commands, we pipe a
	// sleep via argv.
	t.Skip("slow-binary test requires platform-specific sleep fixture; covered by TestClaudeCodeLLM_ContextCancellation in llm_test.go for the shared cliBackend pattern")
}

// TestCLIBackend_ErrorRedaction confirms that a failing subprocess surfaces
// a stderr tail in the error message without leaking the prompt itself.
func TestCLIBackend_ErrorRedaction(t *testing.T) {
	// Run a real command (/bin/false) with a gemini-shaped backend so we
	// can assert the error message format. We route through the real
	// cliBackend by constructing NewGeminiCLILLM("/bin/false") — the
	// argsFn returns `-p <prompt> --output-format text` which /bin/false
	// ignores; /bin/false exits 1 with no stderr.
	l := llm.NewGeminiCLILLM("/bin/false")
	_, err := l.Generate(context.Background(), "some prompt containing secrets")
	if err == nil {
		t.Fatal("expected error from /bin/false, got nil")
	}
	if !strings.Contains(err.Error(), "gemini-cli failed") {
		t.Errorf("error should identify backend: %v", err)
	}
	if strings.Contains(err.Error(), "secrets") {
		t.Errorf("error message should not leak prompt content: %v", err)
	}
}

// TestNewLLM_Routes_NewBackends exercises the switch in llm.NewLLM for
// every new backend and checks that each constructor returns a non-nil LLM
// when given a binary path.
func TestNewLLM_Routes_NewBackends(t *testing.T) {
	cases := []struct {
		backend string
		cfgExtra func(*llm.Config)
	}{
		{backend: "gemini-cli", cfgExtra: func(c *llm.Config) { c.GeminiBinaryPath = "/bin/true" }},
		{backend: "codex", cfgExtra: func(c *llm.Config) { c.CodexBinaryPath = "/bin/true" }},
		{backend: "opencode", cfgExtra: func(c *llm.Config) { c.OpencodeBinaryPath = "/bin/true" }},
		{backend: "pi", cfgExtra: func(c *llm.Config) { c.PiBinaryPath = "/bin/true" }},
	}
	for _, tc := range cases {
		t.Run(tc.backend, func(t *testing.T) {
			cfg := llm.Config{Backend: tc.backend}
			tc.cfgExtra(&cfg)
			l, err := llm.NewLLM(cfg)
			if err != nil {
				t.Fatalf("NewLLM(%q) error: %v", tc.backend, err)
			}
			if l == nil {
				t.Fatalf("NewLLM(%q) returned nil LLM", tc.backend)
			}
		})
	}
}

// TestNewLLM_UnknownBackendLists_NewBackends makes sure the error message
// includes all seven valid backend names so users typoing a name see the
// full list.
func TestNewLLM_UnknownBackendLists_NewBackends(t *testing.T) {
	_, err := llm.NewLLM(llm.Config{Backend: "typoed"})
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
	for _, expected := range []string{"claude-code", "claude-api", "ollama", "gemini-cli", "codex", "opencode", "pi"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error message should list %q among valid backends, got: %v", expected, err)
		}
	}
}

// TestCLIBackend_RealBinarySmoke runs /bin/echo through a cliBackend and
// confirms stdout is trimmed and returned. Guards the happy path.
func TestCLIBackend_RealBinarySmoke(t *testing.T) {
	// We build a gemini-cli backend pointed at /bin/echo. The argsFn
	// returns `-p <prompt> --output-format text` — echo will print those
	// args verbatim, which is fine for asserting stdout capture works.
	l := llm.NewGeminiCLILLM("/bin/echo")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := l.Generate(ctx, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// echo will print all its args; we only care that the prompt made it
	// through the argv path, not the exact echo formatting.
	if !strings.Contains(out, "hello") {
		t.Errorf("stdout should contain the prompt word 'hello', got: %q", out)
	}
	if strings.HasSuffix(out, "\n") {
		t.Errorf("stdout should be TrimSpace'd, still has trailing newline: %q", out)
	}
}
