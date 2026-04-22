package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const stopHookScript = `#!/usr/bin/env python3
"""
llmwiki Claude Code Stop Hook
Reads Stop hook JSON from stdin, extracts the last analytical assistant
response, and pipes it to 'llmwiki absorb <cwd> --note-stdin'.
Always exits 0 — never blocks Claude.
"""
import json
import os
import os.path
import subprocess
import sys

MIN_RESPONSE_CHARS = 300
MAX_NOTE_CHARS = 2000
ANALYTICAL_TOOLS = {"Read", "Grep", "Glob", "Bash"}
RECENT_WINDOW = 20

ALLOWED_TRANSCRIPT_PREFIX = os.path.realpath(os.path.expanduser("~/.claude/projects"))


def _is_safe_transcript_path(path):
    try:
        real = os.path.realpath(path)
    except OSError:
        return False
    return real == ALLOWED_TRANSCRIPT_PREFIX or real.startswith(ALLOWED_TRANSCRIPT_PREFIX + os.sep)


def extract_last_response(transcript_path):
    try:
        with open(transcript_path, "r", encoding="utf-8") as f:
            lines = f.readlines()
    except OSError:
        return None, set()

    last_text = None
    recent_tools = set()
    window = lines[-RECENT_WINDOW:] if len(lines) > RECENT_WINDOW else lines

    for raw in window:
        raw = raw.strip()
        if not raw:
            continue
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            continue

        entry_type = msg.get("type", "")
        inner = msg.get("message")
        if not isinstance(inner, dict):
            inner = msg

        content = inner.get("content", "")
        is_assistant = entry_type == "assistant" or inner.get("role") == "assistant"

        if isinstance(content, list):
            for block in content:
                if not isinstance(block, dict):
                    continue
                block_type = block.get("type", "")
                if block_type == "text" and is_assistant:
                    last_text = block.get("text", last_text)
                elif block_type == "tool_use":
                    tname = block.get("name", "")
                    if tname in ANALYTICAL_TOOLS:
                        recent_tools.add(tname)
        elif isinstance(content, str) and is_assistant:
            last_text = content

    return last_text, recent_tools


def main():
    try:
        event = json.loads(sys.stdin.read())
        cwd = event.get("cwd", "")
        transcript_path = event.get("transcript_path", "")

        if not cwd or not transcript_path:
            sys.exit(0)

        if not _is_safe_transcript_path(transcript_path):
            sys.exit(0)

        last_text, recent_tools = extract_last_response(transcript_path)

        if not last_text or len(last_text) < MIN_RESPONSE_CHARS:
            sys.exit(0)

        if not recent_tools.intersection(ANALYTICAL_TOOLS):
            sys.exit(0)

        note = last_text[:MAX_NOTE_CHARS]
        subprocess.run(
            ["llmwiki", "absorb", cwd, "--note-stdin"],
            input=note,
            text=True,
            timeout=30,
            capture_output=True,
        )
    except Exception as exc:
        try:
            log_path = os.path.join(os.path.expanduser("~"), ".llmwiki", "hook.log")
            os.makedirs(os.path.dirname(log_path), exist_ok=True)
            with open(log_path, "a", encoding="utf-8") as logf:
                logf.write(f"{os.path.basename(__file__)}: {type(exc).__name__}: {exc}\n")
        except Exception:
            pass
    sys.exit(0)


if __name__ == "__main__":
    main()
`

const pluginManifest = `{
  "name": "llmwiki",
  "description": "Captures analytical Claude Code sessions into graymatter memory via llmwiki absorb",
  "author": {
    "name": "Max Małecki"
  },
  "version": "1.0.0"
}
`

const hooksConfig = `{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "python3 ${CLAUDE_PLUGIN_ROOT}/hooks/stop-hook.py",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
`

func NewHookCmd() *cobra.Command {
	hook := &cobra.Command{
		Use:   "hook",
		Short: "Manage Claude Code Stop hooks for automatic session absorption",
	}
	hook.AddCommand(newHookInstallCmd(), newHookUninstallCmd(), newHookStatusCmd())
	return hook
}

func defaultPluginDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plugins", "llmwiki")
}

func writePlugin(pluginDir string) error {
	manifestDir := filepath.Join(pluginDir, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(pluginManifest), 0644); err != nil {
		return err
	}

	hooksDir := filepath.Join(pluginDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "hooks.json"), []byte(hooksConfig), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(hooksDir, "stop-hook.py"), []byte(stopHookScript), 0755)
}

func newHookInstallCmd() *cobra.Command {
	var pluginDir string
	var showSnippet bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install llmwiki as a Claude Code plugin (auto-discovered, no settings.json edit needed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pluginDir == "" {
				pluginDir = defaultPluginDir()
			}

			if err := writePlugin(pluginDir); err != nil {
				return fmt.Errorf("write plugin: %w", err)
			}

			if _, err := exec.LookPath("llmwiki"); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: 'llmwiki' not found in PATH — add it before using the hook")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Plugin installed at %s\nRestart Claude Code to activate.\n", pluginDir)

			if showSnippet {
				fmt.Fprintln(cmd.OutOrStdout(), claudeMDSnippet())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "", "Plugin installation directory (default: ~/.claude/plugins/llmwiki)")
	cmd.Flags().BoolVar(&showSnippet, "show-snippet", false, "Print recommended CLAUDE.md addition")
	return cmd
}

func newHookUninstallCmd() *cobra.Command {
	var pluginDir string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the llmwiki Claude Code plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pluginDir == "" {
				pluginDir = defaultPluginDir()
			}
			if err := os.RemoveAll(pluginDir); err != nil {
				return fmt.Errorf("remove plugin: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Plugin uninstalled.")
			return nil
		},
	}

	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "", "Plugin directory to remove (default: ~/.claude/plugins/llmwiki)")
	return cmd
}

func newHookStatusCmd() *cobra.Command {
	var pluginDir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether the llmwiki Claude Code plugin is installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pluginDir == "" {
				pluginDir = defaultPluginDir()
			}
			manifestPath := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
			_, err := os.Stat(manifestPath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					fmt.Fprintln(cmd.OutOrStdout(), "not installed")
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "not installed (could not read plugin dir: %v)\n", err)
				}
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed: %s\n", pluginDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&pluginDir, "plugin-dir", "", "Plugin directory to check (default: ~/.claude/plugins/llmwiki)")
	return cmd
}

func claudeMDSnippet() string {
	return `
Add to your project CLAUDE.md to capture explicit insights:

  When you discover how a non-obvious system, component, or pattern works,
  run: llmwiki remember --project <project> "<concise single-sentence insight>"
`
}
