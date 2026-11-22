package llm_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emgiezet/llmwiki/internal/llm"
)

// fakeOmp writes an executable shell script standing in for the omp binary.
func fakeOmp(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "omp")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func generate(t *testing.T, l llm.LLM, prompt string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := l.Generate(ctx, prompt)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return out
}

const echoArgvThenStdin = "#!/bin/sh\necho \"ARGV:$*\"\ncat\n"

func argvLine(t *testing.T, out string) string {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "ARGV:") {
			return line
		}
	}
	t.Fatalf("no ARGV line in output: %q", out)
	return ""
}

func TestOmpCLILLM_PromptOnStdinNotArgv(t *testing.T) {
	const prompt = "SENTINEL_PROMPT_TEXT"
	out := generate(t, llm.NewOmpCLILLM(fakeOmp(t, echoArgvThenStdin), ""), prompt)

	if !strings.Contains(out, prompt) {
		t.Errorf("prompt should arrive on stdin and be echoed back, got: %q", out)
	}
	argv := argvLine(t, out)
	if strings.Contains(argv, prompt) {
		t.Errorf("prompt must not be passed in argv, got: %q", argv)
	}
	for _, flag := range []string{"-p", "--no-session"} {
		if !strings.Contains(argv, flag) {
			t.Errorf("argv should contain %q, got: %q", flag, argv)
		}
	}
}

// An argv-based implementation fails this with E2BIG: Linux caps a single
// argv string at 128 KB (MAX_ARG_STRLEN).
func TestOmpCLILLM_LargePromptSurvivesArgvLimit(t *testing.T) {
	prompt := strings.Repeat("x", 200_000)
	out := generate(t, llm.NewOmpCLILLM(fakeOmp(t, "#!/bin/sh\ncat\n"), ""), prompt)
	if len(out) != len(prompt) {
		t.Errorf("output length = %d, want %d", len(out), len(prompt))
	}
}

func TestOmpCLILLM_ModelFlag(t *testing.T) {
	bin := fakeOmp(t, echoArgvThenStdin)

	t.Run("empty model omits flag", func(t *testing.T) {
		argv := argvLine(t, generate(t, llm.NewOmpCLILLM(bin, ""), "hi"))
		if strings.Contains(argv, "--model") {
			t.Errorf("empty model must not pass --model, got: %q", argv)
		}
	})
	t.Run("model is passed through", func(t *testing.T) {
		argv := argvLine(t, generate(t, llm.NewOmpCLILLM(bin, "gpt-5.2"), "hi"))
		if !strings.Contains(argv, "--model gpt-5.2") {
			t.Errorf("argv should contain --model gpt-5.2, got: %q", argv)
		}
	})
}
