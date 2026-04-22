package llm

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateOllamaHost enforces a safe-by-default allowlist on the Ollama
// base URL. Returns nil if host is loopback/localhost (any scheme), or if
// allowRemote is true. Prevents SSRF when ollama_host is attacker-controlled
// (e.g. via a compromised llmwiki.yaml in a malicious project).
func ValidateOllamaHost(host string, allowRemote bool) error {
	if host == "" {
		return fmt.Errorf("ollama_host must not be empty")
	}
	u, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("ollama_host %q is not a valid URL: %w", host, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("ollama_host %q must use http or https", host)
	}
	if allowRemote {
		return nil
	}
	hostname := u.Hostname()
	if isLoopback(hostname) {
		return nil
	}
	return fmt.Errorf("ollama_host %q is not a loopback address; set allow_remote_ollama: true in config to override", host)
}

func isLoopback(h string) bool {
	// Strip IPv6 brackets if url.Parse didn't
	h = strings.TrimPrefix(h, "[")
	h = strings.TrimSuffix(h, "]")
	switch h {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if strings.HasPrefix(h, "127.") {
		return true
	}
	return false
}
