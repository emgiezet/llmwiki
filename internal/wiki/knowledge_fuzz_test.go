package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzKnowledgeLookup checks that no layer/topic input escapes the wiki root.
// Both values are joined onto <root>/knowledge/, so a traversal value must be
// rejected rather than resolved.
func FuzzKnowledgeLookup(f *testing.F) {
	f.Add("global", "auth")
	f.Add("..", "auth")
	f.Add("../../etc", "passwd")
	f.Add("global", "../../../../etc/passwd")
	f.Add("global/../..", "auth")
	f.Add("", "")
	f.Add(".git", "config")
	f.Add("global\x00", "auth")

	root := f.TempDir()
	// A file outside the knowledge tree that must never be reachable.
	if err := os.WriteFile(filepath.Join(root, "outside.md"), []byte("# Secret\n"), 0644); err != nil {
		f.Fatal(err)
	}
	layerDir := filepath.Join(root, KnowledgeDir, "global")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		f.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layerDir, "auth.md"), []byte("# Auth\n"), 0644); err != nil {
		f.Fatal(err)
	}
	store := NewStore(root)
	knowledgeRoot := filepath.Join(root, KnowledgeDir)

	f.Fuzz(func(t *testing.T, layer, topic string) {
		// Must not panic, and must not reach outside <root>/knowledge/.
		if entry, err := store.GetKnowledge(layer, topic); err == nil {
			abs := filepath.Join(root, entry.Path)
			if !strings.HasPrefix(filepath.Clean(abs), knowledgeRoot+string(filepath.Separator)) {
				t.Fatalf("GetKnowledge(%q, %q) escaped the knowledge root: %s", layer, topic, abs)
			}
		}

		entries, err := store.SearchKnowledge([]string{layer}, topic)
		if err != nil {
			return
		}
		for _, e := range entries {
			abs := filepath.Join(root, e.Path)
			if !strings.HasPrefix(filepath.Clean(abs), knowledgeRoot+string(filepath.Separator)) {
				t.Fatalf("SearchKnowledge(%q) escaped the knowledge root: %s", layer, abs)
			}
		}
	})
}
