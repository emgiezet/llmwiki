package cmd

import "github.com/spf13/cobra"

func NewHookCmd() *cobra.Command {
	return &cobra.Command{Use: "hook", Short: "Manage Claude Code hooks", Args: cobra.NoArgs}
}
