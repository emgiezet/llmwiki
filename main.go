package main

import (
	"fmt"
	"os"

	"github.com/mgz/llmwiki/internal/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "llmwiki",
		Short: "LLM-maintained knowledge base for your projects",
	}
	root.SilenceErrors = true
	root.AddCommand(
		cmd.NewIngestCmd(),
		cmd.NewQueryCmd(),
		cmd.NewContextCmd(),
		cmd.NewListCmd(),
		cmd.NewLinkCmd(),
		cmd.NewIndexCmd(),
		cmd.NewRememberCmd(),
		cmd.NewRecallCmd(),
		cmd.NewDocsCmd(),
		cmd.NewAbsorbCmd(),
		cmd.NewMaterializeCmd(),
	)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
