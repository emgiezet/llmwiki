package llm

// NewCodexCLILLM returns an LLM backend that shells out to OpenAI's Codex
// CLI in non-interactive mode: `codex exec "<prompt>"`. The `exec` subcommand
// (alias `e`) is verified against openai/codex codex-rs/cli/src/main.rs.
// Requires `codex` to be authenticated. Empty binaryPath defaults to "codex".
func NewCodexCLILLM(binaryPath string) LLM {
	if binaryPath == "" {
		binaryPath = "codex"
	}
	return &cliBackend{
		name:   "codex",
		binary: binaryPath,
		argsFn: func(prompt string) []string {
			return []string{"exec", prompt}
		},
	}
}
