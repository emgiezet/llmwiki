package wiki_test

import (
	"testing"

	"github.com/emgiezet/llmwiki/internal/wiki"
)

func TestIndexFileName(t *testing.T) {
	cases := []struct {
		name  string
		parts []string
		want  string
	}{
		{"no parts yields legacy name for top-level index", nil, "_index.md"},
		{"empty strings are filtered out", []string{"", ""}, "_index.md"},
		{"single client name", []string{"acme"}, "acme_index.md"},
		{"client + project", []string{"acme", "billing"}, "acme_billing_index.md"},
		{"personal project with empty customer", []string{"", "foo"}, "foo_index.md"},
		{"three-part (system layer, forward-compat)", []string{"acme", "billing-system", "payment-service"}, "acme_billing-system_payment-service_index.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wiki.IndexFileName(tc.parts...)
			if got != tc.want {
				t.Errorf("IndexFileName(%v) = %q, want %q", tc.parts, got, tc.want)
			}
		})
	}
}

func TestIsIndexFileName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"_index.md", true},                      // legacy
		{"acme_index.md", true},                  // v1.1.1+
		{"acme_billing-api_index.md", true},      // project under client
		{"foo_index.md", true},                   // personal/oss
		{"billing-api.md", false},                // regular project file
		{"service.md", false},                    // regular service file
		{"_index.yaml", false},                   // wrong extension
		{"index.md", false},                      // no underscore prefix
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wiki.IsIndexFileName(tc.name); got != tc.want {
				t.Errorf("IsIndexFileName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
