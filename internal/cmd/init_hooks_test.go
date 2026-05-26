package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPreCommitHook_createsScript(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := installPreCommitHook(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("hook file not found: %v", err)
	}

	content := string(data)
	if !contains(content, "llmwiki check") {
		t.Errorf("hook does not contain 'llmwiki check'; got:\n%s", content)
	}
	if !contains(content, "--exit-code") {
		t.Errorf("hook does not contain '--exit-code'; got:\n%s", content)
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("hook file is not executable; mode=%v", info.Mode())
	}
}

func TestInstallPreCommitHook_doesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	original := "#!/bin/sh\nexit 0\n"
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte(original), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := installPreCommitHook(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if string(data) != original {
		t.Errorf("hook was overwritten; got:\n%s", string(data))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
