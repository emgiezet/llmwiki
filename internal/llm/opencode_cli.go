package llm

// NewOpencodeLLM returns an LLM backend that shells out to SST's opencode
// CLI in non-interactive mode: `opencode run --format default "<prompt>"`.
// Using --format default keeps stdout plain text; --format json is reserved
// for callers that want structured events (not needed for wiki-body prompts).
// Empty binaryPath defaults to "opencode".
func NewOpencodeLLM(binaryPath string) LLM {
	if binaryPath == "" {
		binaryPath = "opencode"
	}
	return &cliBackend{
		name:   "opencode",
		binary: binaryPath,
		argsFn: func(prompt string) []string {
			return []string{"run", "--format", "default", prompt}
		},
	}
}
