package main

import (
	"context"
	"fmt"
	"os"

	"github.com/emgiezet/llmwiki/internal/cmd"
	"github.com/emgiezet/llmwiki/internal/update"
	"github.com/emgiezet/llmwiki/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	// Kick off the update check before parsing args so it has the full
	// command duration to resolve. We drain the channel non-blockingly at
	// the very end — anything in flight is abandoned silently.
	updateCtx, cancelUpdate := context.WithCancel(context.Background())
	defer cancelUpdate()
	noticeCh := update.NewChecker().CheckAsync(updateCtx, version.Version)

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
		cmd.NewDrainCmd(),
		cmd.NewMaterializeCmd(),
		cmd.NewHookCmd(),
		cmd.NewVersionCmd(),
		cmd.NewUpdateCmd(),
		cmd.NewClientCmd(),
	)

	err := root.Execute()

	// Non-blocking drain of the update notice. If the HTTP call hasn't
	// finished yet, we simply skip — next invocation will pick it up.
	select {
	case notice := <-noticeCh:
		if notice != "" {
			fmt.Fprintln(os.Stderr, notice)
		}
	default:
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
