package wiki

import "strings"

// LegacyIndexFileName is the old universal index filename (pre-v1.1.1).
// Readers that need to recognize both the new and the old name should
// compare against this constant and use strings.HasSuffix(name, "_index.md")
// which matches both shapes ("_index.md" and "{prefix}_index.md").
const LegacyIndexFileName = "_index.md"

// IndexFileName returns the correct _index.md filename given the path
// context. parts are joined in order with "_" and suffixed with "_index.md".
// Zero parts (top-level index) returns the bare "_index.md" for backward
// compatibility with existing readers and external tooling that greps for
// that exact name.
//
// Examples:
//
//	IndexFileName()                    → "_index.md"
//	IndexFileName("acme")              → "acme_index.md"
//	IndexFileName("acme", "billing")   → "acme_billing_index.md"
//	IndexFileName("foo")               → "foo_index.md"   (personal/oss project)
//
// Empty-string arguments are filtered out so callers don't have to guard
// against optional fields (e.g. personal projects where customer=="" and
// client-level indexes where project=="").
func IndexFileName(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) == 0 {
		return LegacyIndexFileName
	}
	return strings.Join(nonEmpty, "_") + "_index.md"
}

// IsIndexFileName reports whether name looks like an index file under the
// new or legacy convention. Used by tree walkers that need to skip indexes
// while tolerating mixed-era wiki layouts.
func IsIndexFileName(name string) bool {
	return strings.HasSuffix(name, "_index.md")
}
