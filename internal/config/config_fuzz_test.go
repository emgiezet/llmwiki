package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadGlobalConfig(f *testing.F) {
	f.Add([]byte(`wiki_root: ~/wiki
llm: ollama
memory_enabled: true`))
	f.Add([]byte(`llm: [nope, not a string]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadGlobalConfig(path)
	})
}

func FuzzLoadProjectConfig(f *testing.F) {
	f.Add([]byte(`type: client
customer: acme`))
	f.Add([]byte(`customer: "../../../etc"`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "llmwiki.yaml")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadProjectConfig(dir)
	})
}
