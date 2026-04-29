package llm

// NewPiCLILLM returns an LLM backend that shells out to pi-coding-agent
// (`@mariozechner/pi-coding-agent`) in print mode: `pi -p "<prompt>"`.
// Pi can also read stdin and merge it into the initial prompt ("In print
// mode, pi also reads piped stdin") but we keep the simple argv form here;
// long prompts are handled by argv up to system limits.
// Empty binaryPath defaults to "pi".
func NewPiCLILLM(binaryPath string) LLM {
	if binaryPath == "" {
		binaryPath = "pi"
	}
	return &cliBackend{
		name:   "pi",
		binary: binaryPath,
		argsFn: func(prompt string) []string {
			return []string{"-p", prompt}
		},
	}
}
