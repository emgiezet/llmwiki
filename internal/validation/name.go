// Package validation centralizes strict validation for CLI/config values that
// flow into filesystem paths, preventing path traversal and related issues.
package validation

import (
	"fmt"
	"regexp"
)

// nameRe allows letters, digits, hyphen, underscore, and dot.
// Must not be empty, must not be "." or "..", must not start with a dot
// (to avoid hidden files), must not contain path separators or null bytes.
var nameRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.\-]*$`)

// NameComponent validates that s is a safe single path segment:
//   - non-empty
//   - contains only [A-Za-z0-9_.-]
//   - does not start with a dot
//   - is not "." or ".."
//   - does not contain "/", "\\", or null bytes
//
// Used for customer, project, service, and type values that get joined into
// filesystem paths under the wiki root. Returns a wrapped error naming the field.
func NameComponent(field, s string) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	if s == "." || s == ".." {
		return fmt.Errorf("%s must not be %q", field, s)
	}
	if !nameRe.MatchString(s) {
		return fmt.Errorf("%s %q contains invalid characters (allowed: A-Z a-z 0-9 . _ -, no leading dot)", field, s)
	}
	return nil
}

// NameComponentOptional behaves like NameComponent but allows empty strings
// (for optional fields where empty means "use default"). Non-empty values
// must still pass strict validation.
func NameComponentOptional(field, s string) error {
	if s == "" {
		return nil
	}
	return NameComponent(field, s)
}
