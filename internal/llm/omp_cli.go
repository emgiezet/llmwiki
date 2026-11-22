package llm

// NewOmpCLILLM returns an LLM backend that shells out to omp (oh-my-pi) in
// print mode. The prompt goes on stdin — omp reads non-TTY stdin to EOF as
// prompt text — which keeps large scan summaries clear of the 128 KB
// MAX_ARG_STRLEN limit that the argv-based CLI backends are subject to.
//
// The flags pin a deterministic, side-effect-free text run: no session file,
// no title-generation call, no tools, and none of the user's extensions,
// skills, rules or context files, which would otherwise be loaded from the
// directory being ingested and billed into every call.
// Empty binaryPath defaults to "omp"; empty model leaves omp's own default.
func NewOmpCLILLM(binaryPath, model string) LLM {
	if binaryPath == "" {
		binaryPath = "omp"
	}
	return &cliBackend{
		name:     "omp",
		binary:   binaryPath,
		useStdin: true,
		// The prompt argument is ignored on purpose: useStdin pipes it, and
		// repeating it in argv would send it twice.
		argsFn: func(string) []string {
			args := []string{
				"-p", "--mode", "text",
				"--no-session", "--no-title", "--no-tools",
				"--no-extensions", "--no-context-files", "--no-skills", "--no-rules",
			}
			if model != "" {
				args = append(args, "--model", model)
			}
			return args
		},
	}
}
