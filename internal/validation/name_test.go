package validation

import "testing"

func TestNameComponent(t *testing.T) {
	valid := []string{"acme", "my-project", "svc_1", "v1.2.3", "a", "A1"}
	invalid := []struct {
		name, want string
	}{
		{"", "empty"},
		{".", `"."`},
		{"..", `".."`},
		{"../etc", "invalid"},
		{"foo/bar", "invalid"},
		{"foo\\bar", "invalid"},
		{".hidden", "invalid"},
		{"with space", "invalid"},
		{"has\x00null", "invalid"},
		{"foo;bar", "invalid"},
		{"-leading-dash", "invalid"},
	}
	for _, s := range valid {
		if err := NameComponent("x", s); err != nil {
			t.Errorf("NameComponent(%q) unexpected error: %v", s, err)
		}
	}
	for _, tc := range invalid {
		if err := NameComponent("x", tc.name); err == nil {
			t.Errorf("NameComponent(%q) expected error containing %q, got nil", tc.name, tc.want)
		}
	}
}

func TestNameComponentOptional(t *testing.T) {
	if err := NameComponentOptional("x", ""); err != nil {
		t.Errorf("empty should be allowed: %v", err)
	}
	if err := NameComponentOptional("x", "../etc"); err == nil {
		t.Error("traversal should be rejected")
	}
}
