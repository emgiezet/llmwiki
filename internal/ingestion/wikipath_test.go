package ingestion_test

import (
	"strings"
	"testing"

	"github.com/mgz/llmwiki/internal/validation"
)

// TestNameComponent_BlocksTraversal is a belt-and-suspenders test that
// confirms the validator would catch traversal payloads that would
// previously have escaped the wiki root when joined with filepath.Join.
func TestNameComponent_BlocksTraversal(t *testing.T) {
	payloads := []string{
		"../../../tmp",
		"..",
		"/etc/passwd",
		"foo/bar",
		"foo\\bar",
		".ssh",
	}
	for _, p := range payloads {
		if err := validation.NameComponent("test", p); err == nil {
			t.Errorf("payload %q should have been rejected", p)
		} else if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "empty") && !strings.Contains(err.Error(), `"."`) && !strings.Contains(err.Error(), `".."`) {
			t.Errorf("error for %q should be descriptive: %v", p, err)
		}
	}
}
