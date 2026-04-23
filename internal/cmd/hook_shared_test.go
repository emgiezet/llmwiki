package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	cases := []struct {
		in   string
		want shellKind
	}{
		{"/bin/bash", shellBash},
		{"/usr/bin/bash", shellBash},
		{"/bin/zsh", shellZsh},
		{"/usr/local/bin/zsh", shellZsh},
		{"/usr/bin/fish", shellFish},
		{"/bin/sh", shellOther},
		{"", shellOther},
		{"/bin/ksh", shellOther},
	}
	for _, c := range cases {
		if got := detectShell(c.in); got != c.want {
			t.Errorf("detectShell(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestRCFilePath(t *testing.T) {
	home := "/home/alice"
	cases := []struct {
		kind shellKind
		want string
	}{
		{shellBash, filepath.Join(home, ".bashrc")},
		{shellZsh, filepath.Join(home, ".zshrc")},
		{shellFish, filepath.Join(home, ".config", "fish", "config.fish")},
		{shellOther, filepath.Join(home, ".profile")},
	}
	for _, c := range cases {
		if got := rcFilePath(c.kind, home); got != c.want {
			t.Errorf("rcFilePath(%v) = %q, want %q", c.kind, got, c.want)
		}
	}
}

func TestUpsertMarkerBlock_Insert(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rc")
	if err := os.WriteFile(target, []byte("existing line 1\nexisting line 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := upsertMarkerBlock(target, markerBlock{
		Begin: "# llmwiki:begin foo",
		End:   "# llmwiki:end foo",
		Body:  "source /path/to/hook.sh",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !changed {
		t.Error("first insert should report changed=true")
	}
	data, _ := os.ReadFile(target)
	got := string(data)
	if !strings.Contains(got, "existing line 1") || !strings.Contains(got, "existing line 2") {
		t.Errorf("upsert should preserve existing content, got:\n%s", got)
	}
	if !strings.Contains(got, "# llmwiki:begin foo\nsource /path/to/hook.sh\n# llmwiki:end foo\n") {
		t.Errorf("upsert should emit marker block, got:\n%s", got)
	}
}

func TestUpsertMarkerBlock_Idempotent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rc")

	block := markerBlock{
		Begin: "# llmwiki:begin foo",
		End:   "# llmwiki:end foo",
		Body:  "source /path/to/hook.sh",
	}

	// First call: inserts.
	changed1, err := upsertMarkerBlock(target, block)
	if err != nil || !changed1 {
		t.Fatalf("first upsert: err=%v changed=%v", err, changed1)
	}

	// Second call with identical content: should not rewrite.
	changed2, err := upsertMarkerBlock(target, block)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if changed2 {
		t.Error("second identical upsert should report changed=false")
	}
}

func TestUpsertMarkerBlock_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rc")
	if err := os.WriteFile(target, []byte(
		"line A\n# llmwiki:begin foo\nold body\n# llmwiki:end foo\nline B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := upsertMarkerBlock(target, markerBlock{
		Begin: "# llmwiki:begin foo",
		End:   "# llmwiki:end foo",
		Body:  "new body",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	data, _ := os.ReadFile(target)
	got := string(data)
	if strings.Contains(got, "old body") {
		t.Errorf("replace should drop old body, got:\n%s", got)
	}
	if !strings.Contains(got, "new body") {
		t.Errorf("replace should include new body, got:\n%s", got)
	}
	if !strings.Contains(got, "line A") || !strings.Contains(got, "line B") {
		t.Errorf("replace must preserve surrounding content, got:\n%s", got)
	}
}

func TestRemoveMarkerBlock_StripsRegion(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "rc")
	if err := os.WriteFile(target, []byte(
		"line A\n# llmwiki:begin foo\nbody\n# llmwiki:end foo\nline B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := removeMarkerBlock(target, "# llmwiki:begin foo", "# llmwiki:end foo")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed {
		t.Error("removeMarkerBlock should report true when region was present")
	}

	data, _ := os.ReadFile(target)
	got := string(data)
	if strings.Contains(got, "llmwiki:") || strings.Contains(got, "body") {
		t.Errorf("remove must strip entire marker region, got:\n%s", got)
	}
	if !strings.Contains(got, "line A") || !strings.Contains(got, "line B") {
		t.Errorf("remove must preserve surrounding content, got:\n%s", got)
	}
}

func TestRemoveMarkerBlock_MissingFile(t *testing.T) {
	removed, err := removeMarkerBlock(filepath.Join(t.TempDir(), "does-not-exist"), "BEGIN", "END")
	if err != nil {
		t.Errorf("missing file should not error: %v", err)
	}
	if removed {
		t.Error("missing file should not report removed=true")
	}
}
