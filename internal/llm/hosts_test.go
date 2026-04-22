package llm

import "testing"

func TestValidateOllamaHost(t *testing.T) {
	allow := []string{
		"http://localhost",
		"http://localhost:11434",
		"http://127.0.0.1:11434",
		"https://127.0.0.1",
		"http://[::1]:11434",
	}
	deny := []string{
		"",
		"http://169.254.169.254/latest/meta-data/",
		"http://example.com",
		"file:///etc/passwd",
		"ftp://localhost",
	}
	for _, h := range allow {
		if err := ValidateOllamaHost(h, false); err != nil {
			t.Errorf("expected %q allowed, got %v", h, err)
		}
	}
	for _, h := range deny {
		if err := ValidateOllamaHost(h, false); err == nil {
			t.Errorf("expected %q denied", h)
		}
	}
	// allowRemote opens the door
	if err := ValidateOllamaHost("http://example.com", true); err != nil {
		t.Errorf("allowRemote should bypass allowlist: %v", err)
	}
	// But syntax errors still rejected even with allowRemote
	if err := ValidateOllamaHost("not a url at all", true); err == nil {
		// Actually "not a url at all" parses as a relative ref with empty scheme, which fails the scheme check
		t.Error("malformed URL should be rejected")
	}
}
