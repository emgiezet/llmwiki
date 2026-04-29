package llm

// NewGeminiCLILLM returns an LLM backend that shells out to Google's Gemini
// CLI in non-interactive mode: `gemini -p "<prompt>" --output-format text`.
// Requires `gemini` (or the provided binaryPath) to be on PATH and
// authenticated. Empty binaryPath defaults to the basename "gemini".
func NewGeminiCLILLM(binaryPath string) LLM {
	if binaryPath == "" {
		binaryPath = "gemini"
	}
	return &cliBackend{
		name:   "gemini-cli",
		binary: binaryPath,
		argsFn: func(prompt string) []string {
			return []string{"-p", prompt, "--output-format", "text"}
		},
	}
}
