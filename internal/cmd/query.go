package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/mgz/llmwiki/internal/safeio"
	"github.com/spf13/cobra"
)

func NewQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query <question>",
		Short: "Ask a question across all wiki entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := args[0]
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}

			wikiContent, err := loadAllWikiContent(global.WikiRoot)
			if err != nil {
				return err
			}
			if wikiContent == "" {
				fmt.Println("No wiki entries found. Run: llmwiki ingest <path>")
				return nil
			}

			if global.AnthropicAPIKey == "" {
				global.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}

			// Initialize memory store if enabled.
			cfg := config.Merge(global, config.ProjectConfig{})
			var memoryContext string
			if cfg.MemoryEnabled {
				mem, memErr := memory.NewFromConfig(cfg)
				if memErr == nil {
					defer mem.Close()
					if facts, recallErr := mem.RecallForQuery(cmd.Context(), question); recallErr == nil && len(facts) > 0 {
						memoryContext = "\nRELEVANT FACTS FROM MEMORY:\n" + strings.Join(facts, "\n") + "\n"
					}
				}
			}

			l, err := llm.NewLLM(llm.Config{
				Backend:           global.LLM,
				AnthropicAPIKey:   global.AnthropicAPIKey,
				OllamaHost:        global.OllamaHost,
				AllowRemoteOllama: global.AllowRemoteOllama,
			})
			if err != nil {
				return err
			}

			prompt := fmt.Sprintf(`You are answering a question about a collection of software projects.
Use only the wiki content below to answer. Be concise.

WIKI CONTENT:
%s
%s
QUESTION: %s

Answer:`, wikiContent, memoryContext, question)

			answer, err := l.Generate(cmd.Context(), prompt)
			if err != nil {
				return err
			}
			fmt.Println(answer)
			return nil
		},
	}
}

func loadAllWikiContent(wikiRoot string) (string, error) {
	var parts []string
	err := filepath.WalkDir(wikiRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") || strings.HasSuffix(path, "_index.md") {
			return nil
		}
		data, readErr := safeio.ReadRegularFile(path)
		if readErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(wikiRoot, path)
		content := string(data)
		if len(content) > 3000 {
			content = content[:3000] + "\n[truncated]"
		}
		parts = append(parts, fmt.Sprintf("=== %s ===\n%s", rel, content))
		return nil
	})
	return strings.Join(parts, "\n\n"), err
}
