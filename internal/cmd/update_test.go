package cmd

import (
	"strings"
	"testing"
)

func TestDetectInstallSource(t *testing.T) {
	cases := []struct {
		name    string
		exePath string
		goBin   string
		want    installSource
	}{
		{
			name:    "gopath bin matches",
			exePath: "/home/alice/go/bin/llmwiki",
			goBin:   "/home/alice/go/bin",
			want:    sourceGoInstall,
		},
		{
			name:    "path contains /go/bin/ even without GOBIN set",
			exePath: "/home/alice/go/bin/llmwiki",
			goBin:   "",
			want:    sourceGoInstall,
		},
		{
			name:    "release install under ~/.local/bin",
			exePath: "/home/alice/.local/bin/llmwiki",
			goBin:   "/home/alice/go/bin",
			want:    sourceRelease,
		},
		{
			name:    "release install under /usr/local/bin",
			exePath: "/usr/local/bin/llmwiki",
			goBin:   "/home/alice/go/bin",
			want:    sourceRelease,
		},
		{
			name:    "empty path returns unknown",
			exePath: "",
			goBin:   "/home/alice/go/bin",
			want:    sourceUnknown,
		},
		{
			name:    "trailing separator on goBin still matches",
			exePath: "/home/alice/go/bin/llmwiki",
			goBin:   "/home/alice/go/bin/",
			want:    sourceGoInstall,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectInstallSource(tc.exePath, tc.goBin)
			if got != tc.want {
				t.Errorf("detectInstallSource(%q, %q) = %v, want %v",
					tc.exePath, tc.goBin, got, tc.want)
			}
		})
	}
}

func TestBuildUpdateInvocation_GoInstallLatest(t *testing.T) {
	name, args := buildUpdateInvocation(sourceGoInstall, "/home/alice/go/bin/llmwiki", "")
	if name != "go" {
		t.Errorf("command name = %q, want %q", name, "go")
	}
	if len(args) != 2 || args[0] != "install" {
		t.Fatalf("unexpected args: %v", args)
	}
	if args[1] != "github.com/emgiezet/llmwiki@latest" {
		t.Errorf("target = %q, want @latest", args[1])
	}
}

func TestBuildUpdateInvocation_GoInstallPinned(t *testing.T) {
	_, args := buildUpdateInvocation(sourceGoInstall, "/go/bin/llmwiki", "v0.5.0")
	if args[1] != "github.com/emgiezet/llmwiki@v0.5.0" {
		t.Errorf("target = %q, want @v0.5.0", args[1])
	}
}

func TestBuildUpdateInvocation_ReleasePath(t *testing.T) {
	name, args := buildUpdateInvocation(sourceRelease, "/home/alice/.local/bin/llmwiki", "")
	if name != "sh" {
		t.Errorf("command name = %q, want %q", name, "sh")
	}
	if len(args) != 2 || args[0] != "-c" {
		t.Fatalf("unexpected args: %v", args)
	}
	script := args[1]
	// The sh -c script must set INSTALL_DIR to the current binary's directory.
	if !strings.Contains(script, `INSTALL_DIR="/home/alice/.local/bin"`) {
		t.Errorf("script missing INSTALL_DIR for exe dir: %q", script)
	}
	// Must reference the installer URL we serve from the repo.
	if !strings.Contains(script, "raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh") {
		t.Errorf("script does not reference the installer URL: %q", script)
	}
	// Latest (no pin) sends an empty VERSION.
	if !strings.Contains(script, `VERSION=""`) {
		t.Errorf("script should send empty VERSION for latest: %q", script)
	}
}

func TestBuildUpdateInvocation_ReleaseWithPin(t *testing.T) {
	_, args := buildUpdateInvocation(sourceRelease, "/usr/local/bin/llmwiki", "v0.5.0")
	script := args[1]
	if !strings.Contains(script, `VERSION="v0.5.0"`) {
		t.Errorf("script should pin VERSION: %q", script)
	}
}
