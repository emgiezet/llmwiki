package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
import subprocess
import sys

MIN_RESPONSE_CHARS = 300
MAX_NOTE_CHARS = 2000
ANALYTICAL_TOOLS = {"Read", "Grep", "Glob", "Bash"}
RECENT_WINDOW = 20


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
    except Exception:
        pass
    sys.exit(0)


if __name__ == "__main__":
    main()
`

const hookEntryMarker = "stop-hook.py"

func NewHookCmd() *cobra.Command {
	hook := &cobra.Command{
		Use:   "hook",
		Short: "Manage Claude Code Stop hooks for automatic session absorption",
	}
	hook.AddCommand(newHookInstallCmd(), newHookUninstallCmd(), newHookStatusCmd())
	return hook
}

func newHookInstallCmd() *cobra.Command {
	var settingsPath, scriptDir string
	var showSnippet bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Write stop-hook.py and register it in Claude Code settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if settingsPath == "" {
				settingsPath = defaultSettingsPath()
			}
			if scriptDir == "" {
				scriptDir = defaultScriptDir()
			}

			if err := writeHookScript(scriptDir); err != nil {
				return fmt.Errorf("write hook script: %w", err)
			}

			scriptPath := filepath.Join(scriptDir, "stop-hook.py")
			command := "python3 " + scriptPath
			if err := injectHookEntry(settingsPath, command); err != nil {
				return fmt.Errorf("update settings.json: %w", err)
			}

			if _, err := exec.LookPath("llmwiki"); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: 'llmwiki' not found in PATH — add it before using the hook")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Hook installed.\n  Script:   %s\n  Settings: %s\n", scriptPath, settingsPath)

			if showSnippet {
				fmt.Fprintln(cmd.OutOrStdout(), claudeMDSnippet())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&settingsPath, "settings", "", "Path to Claude Code settings.json (default: ~/.claude/settings.json)")
	cmd.Flags().StringVar(&scriptDir, "script-dir", "", "Directory to write stop-hook.py (default: ~/.llmwiki/hooks)")
	cmd.Flags().BoolVar(&showSnippet, "show-snippet", false, "Print recommended CLAUDE.md addition")
	return cmd
}

func newHookUninstallCmd() *cobra.Command {
	var settingsPath string

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the llmwiki Stop hook entry from Claude Code settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if settingsPath == "" {
				settingsPath = defaultSettingsPath()
			}
			if err := removeHookEntry(settingsPath); err != nil {
				return fmt.Errorf("update settings.json: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Hook uninstalled from settings.json (script not deleted).")
			return nil
		},
	}

	cmd.Flags().StringVar(&settingsPath, "settings", "", "Path to Claude Code settings.json (default: ~/.claude/settings.json)")
	return cmd
}

func newHookStatusCmd() *cobra.Command {
	var settingsPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether the llmwiki Stop hook is installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			if settingsPath == "" {
				settingsPath = defaultSettingsPath()
			}
			installed, err := hookIsInstalled(settingsPath)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "not installed (settings.json not found or unreadable)")
				return nil
			}
			if installed {
				fmt.Fprintf(cmd.OutOrStdout(), "installed: %s\n", settingsPath)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "not installed")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&settingsPath, "settings", "", "Path to Claude Code settings.json (default: ~/.claude/settings.json)")
	return cmd
}

func defaultSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func defaultScriptDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".llmwiki", "hooks")
}

func writeHookScript(scriptDir string) error {
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(scriptDir, "stop-hook.py"), []byte(stopHookScript), 0755)
}

func loadSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

func saveSettings(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func injectHookEntry(settingsPath, command string) error {
	m, err := loadSettings(settingsPath)
	if err != nil {
		return err
	}

	hooks, _ := m["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	stopHooks, _ := hooks["Stop"].([]any)

	for _, entry := range stopHooks {
		if e, ok := entry.(map[string]any); ok {
			if cmd, ok := e["command"].(string); ok && strings.Contains(cmd, hookEntryMarker) {
				return nil // already installed
			}
		}
	}

	hooks["Stop"] = append(stopHooks, map[string]any{
		"type":    "command",
		"command": command,
		"timeout": 30,
	})
	m["hooks"] = hooks
	return saveSettings(settingsPath, m)
}

func removeHookEntry(settingsPath string) error {
	m, err := loadSettings(settingsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	hooks, _ := m["hooks"].(map[string]any)
	if hooks == nil {
		return nil
	}
	stopHooks, _ := hooks["Stop"].([]any)
	filtered := make([]any, 0, len(stopHooks))
	for _, entry := range stopHooks {
		if e, ok := entry.(map[string]any); ok {
			if cmd, ok := e["command"].(string); ok && strings.Contains(cmd, hookEntryMarker) {
				continue
			}
		}
		filtered = append(filtered, entry)
	}

	hooks["Stop"] = filtered
	m["hooks"] = hooks
	return saveSettings(settingsPath, m)
}

func hookIsInstalled(settingsPath string) (bool, error) {
	m, err := loadSettings(settingsPath)
	if err != nil {
		return false, err
	}
	hooks, _ := m["hooks"].(map[string]any)
	stopHooks, _ := hooks["Stop"].([]any)
	for _, entry := range stopHooks {
		if e, ok := entry.(map[string]any); ok {
			if cmd, ok := e["command"].(string); ok && strings.Contains(cmd, hookEntryMarker) {
				return true, nil
			}
		}
	}
	return false, nil
}

func claudeMDSnippet() string {
	return `
Add to your project CLAUDE.md to capture explicit insights:

  When you discover how a non-obvious system, component, or pattern works,
  run: llmwiki remember --project <project> "<concise single-sentence insight>"
`
}
