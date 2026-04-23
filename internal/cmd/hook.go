package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// toolHook is the interface every per-tool hook installer implements. Each
// tool's install/uninstall/status logic lives in its own file (hook_claude_code.go,
// hook_codex.go, etc.) so the dispatcher stays small.
type toolHook interface {
	// Name returns the short tool name used on the command line (e.g. "claude-code").
	Name() string
	// Install writes the hook (and any migration) so the tool begins forwarding
	// session endings to `llmwiki absorb`. Idempotent on repeated calls.
	Install() error
	// Uninstall reverses Install. Leaves user-authored config untouched.
	Uninstall() error
	// Status reports whether the hook is currently installed; path is the
	// tool-specific location (for display only, empty when not installed).
	Status() (installed bool, path string, err error)
}

// toolHooks is the registry of installers. Edit this when adding a new tool.
var toolHooks = map[string]toolHook{
	"claude-code": &claudeCodeHook{},
	"codex":       &codexHook{},
	"opencode":    &opencodeHook{},
	"pi":          &piHook{},
	"gemini-cli":  &geminiCLIHook{},
}

func toolNamesSorted() []string {
	names := make([]string, 0, len(toolHooks))
	for n := range toolHooks {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// resolveTool maps a positional arg to either one installer or the full set
// (when the user passes "all"). Unknown values produce a helpful error.
func resolveTool(name string) ([]toolHook, error) {
	if name == "all" {
		names := toolNamesSorted()
		out := make([]toolHook, 0, len(names))
		for _, n := range names {
			out = append(out, toolHooks[n])
		}
		return out, nil
	}
	if h, ok := toolHooks[name]; ok {
		return []toolHook{h}, nil
	}
	return nil, fmt.Errorf("unknown tool %q — valid: %s, all", name, strings.Join(toolNamesSorted(), ", "))
}

func NewHookCmd() *cobra.Command {
	hook := &cobra.Command{
		Use:   "hook",
		Short: "Manage session-capture hooks across agentic coding tools",
		Long: `Installs, uninstalls, and inspects the llmwiki hook for each supported
agentic coding tool (claude-code, codex, opencode, pi, gemini-cli). Each
hook captures end-of-session assistant responses and forwards them to
'llmwiki absorb' so knowledge accumulates in graymatter memory.`,
	}
	hook.AddCommand(newHookInstallCmd(), newHookUninstallCmd(), newHookStatusCmd())
	return hook
}

func newHookInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <tool>",
		Short: "Install the llmwiki hook for a specific tool (or 'all')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hooks, err := resolveTool(args[0])
			if err != nil {
				return err
			}
			for _, h := range hooks {
				if err := h.Install(); err != nil {
					return fmt.Errorf("install %s: %w", h.Name(), err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "installed: %s\n", h.Name())
			}
			return nil
		},
	}
}

func newHookUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <tool>",
		Short: "Remove the llmwiki hook for a specific tool (or 'all')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hooks, err := resolveTool(args[0])
			if err != nil {
				return err
			}
			for _, h := range hooks {
				if err := h.Uninstall(); err != nil {
					return fmt.Errorf("uninstall %s: %w", h.Name(), err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "uninstalled: %s\n", h.Name())
			}
			return nil
		},
	}
}

func newHookStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show which per-tool hooks are installed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "tool         installed  path")
			for _, name := range toolNamesSorted() {
				h := toolHooks[name]
				installed, path, err := h.Status()
				status := "no"
				if installed {
					status = "yes"
				}
				if err != nil {
					fmt.Fprintf(out, "%-12s %-10s (error: %v)\n", name, status, err)
					continue
				}
				fmt.Fprintf(out, "%-12s %-10s %s\n", name, status, path)
			}
			return nil
		},
	}
}
