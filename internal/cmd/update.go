package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// installSource describes how the currently-running binary was installed,
// which determines how we upgrade it.
type installSource int

const (
	sourceUnknown installSource = iota
	sourceGoInstall
	sourceRelease
)

// installerURL is the raw URL of the release installer shell script.
const installerURL = "https://raw.githubusercontent.com/emgiezet/llmwiki/main/install.sh"

// modulePath is the canonical module path used by `go install`.
const modulePath = "github.com/emgiezet/llmwiki"

func NewUpdateCmd() *cobra.Command {
	var pinVersion string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update llmwiki in place",
		Long: `Update the running llmwiki binary to the latest release (or a pinned
version with --version). Picks the right mechanism based on how this binary
was installed: 'go install' for Go-managed installs, otherwise re-runs the
release installer script.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("locate current binary: %w", err)
			}
			exe, _ = filepath.EvalSymlinks(exe)

			source := detectInstallSource(exe, goBinDir())
			shellCmd, shellArgs := buildUpdateInvocation(source, exe, pinVersion)

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "would run: %s %s\n",
					shellCmd, strings.Join(shellArgs, " "))
				return nil
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "updating llmwiki (%s)…\n",
				describeSource(source))
			return runWithStreams(cmd.OutOrStdout(), cmd.ErrOrStderr(), shellCmd, shellArgs...)
		},
	}

	cmd.Flags().StringVar(&pinVersion, "version", "",
		"Pin to a specific release (e.g. v0.5.0). Default: latest.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Print the command that would run, without executing it.")
	return cmd
}

// detectInstallSource decides whether the binary at exePath looks like a
// go-install result or a release-binary install. goBin is the directory that
// `go env GOPATH`/bin resolves to — empty disables that signal.
//
// Heuristics (in order):
//  1. If exePath is inside goBin → go-install.
//  2. If exePath contains "/go/bin/" → go-install (catches GOBIN overrides,
//     GOPATH not set, etc. without shelling out).
//  3. Otherwise → release.
//
// Exposed for unit testing.
func detectInstallSource(exePath, goBin string) installSource {
	if exePath == "" {
		return sourceUnknown
	}
	cleanExe := filepath.Clean(exePath)
	if goBin != "" {
		cleanGoBin := filepath.Clean(goBin)
		if strings.HasPrefix(cleanExe, cleanGoBin+string(filepath.Separator)) {
			return sourceGoInstall
		}
	}
	if strings.Contains(cleanExe, string(filepath.Separator)+"go"+
		string(filepath.Separator)+"bin"+string(filepath.Separator)) {
		return sourceGoInstall
	}
	return sourceRelease
}

// buildUpdateInvocation returns the command (name + args) that will upgrade
// the binary. Kept pure so tests can assert the shape without executing.
func buildUpdateInvocation(source installSource, exePath, pinVersion string) (string, []string) {
	switch source {
	case sourceGoInstall:
		target := modulePath + "@latest"
		if pinVersion != "" {
			target = modulePath + "@" + pinVersion
		}
		return "go", []string{"install", target}
	default: // sourceRelease / sourceUnknown
		installDir := filepath.Dir(exePath)
		// Pipe curl → sh inside a single `sh -c` so we can inject VERSION and
		// INSTALL_DIR as environment variables for the installer.
		env := fmt.Sprintf("VERSION=%q INSTALL_DIR=%q", pinVersion, installDir)
		script := fmt.Sprintf("%s curl -fsSL %s | sh", env, installerURL)
		return "sh", []string{"-c", script}
	}
}

// describeSource returns a human-readable label for log lines.
func describeSource(s installSource) string {
	switch s {
	case sourceGoInstall:
		return "go install"
	case sourceRelease:
		return "release installer"
	default:
		return "unknown source"
	}
}

// goBinDir returns $GOBIN if set, else $GOPATH/bin, else empty string.
// Kept as a thin wrapper so tests can drive detectInstallSource directly
// without needing the Go toolchain installed.
func goBinDir() string {
	if v := os.Getenv("GOBIN"); v != "" {
		return v
	}
	if v := os.Getenv("GOPATH"); v != "" {
		return filepath.Join(v, "bin")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "go", "bin")
	}
	return ""
}

// runWithStreams executes the given command, piping stdout/stderr to the
// provided writers. Returns the command's error verbatim.
func runWithStreams(stdout, stderr io.Writer, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout = stdout
	c.Stderr = stderr
	c.Stdin = os.Stdin
	return c.Run()
}
