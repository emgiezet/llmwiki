package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/tracker"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

// CheckResult is the top-level JSON output for `llmwiki check`.
type CheckResult struct {
	ProjectDir string        `json:"project_dir"`
	Entries    []EntryStatus `json:"entries"`
	AnyStale   bool          `json:"any_stale"`
}

// EntryStatus describes the freshness of a single wiki file.
type EntryStatus struct {
	File        string `json:"file"`
	Area        string `json:"area,omitempty"`
	IsStale     bool   `json:"is_stale"`
	StoredHash  string `json:"stored_hash,omitempty"`
	CurrentHash string `json:"current_hash,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Note        string `json:"note,omitempty"`
}

// entryStatusInternal is an unexported wrapper that carries areaFiles for the
// --files filter before the final EntryStatus is produced.
type entryStatusInternal struct {
	EntryStatus
	areaFiles []string
}

// NewCheckCmd creates the `llmwiki check` command.
func NewCheckCmd() *cobra.Command {
	var jsonOutput bool
	var exitCode bool
	var filesFilter string
	var wikiRootFlag string

	cmd := &cobra.Command{
		Use:   "check [project-path]",
		Short: "Check whether wiki entries for a project are stale",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Resolve project directory.
			projectDir := "."
			if len(args) > 0 {
				projectDir = args[0]
			}
			absProjectDir, err := filepath.Abs(projectDir)
			if err != nil {
				return fmt.Errorf("resolve project dir: %w", err)
			}

			// 2. Load config.
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			project, err := config.LoadProjectConfig(absProjectDir)
			if err != nil {
				return fmt.Errorf("load project config: %w", err)
			}
			cfg := config.Merge(global, config.ClientConfig{}, project)

			// 3. --wiki-root flag overrides config.
			if wikiRootFlag != "" {
				cfg.WikiRoot = wikiRootFlag
			}

			// 4. Build git runner (ErrNoGit is handled gracefully below).
			runner, gitErr := tracker.NewGitRunner()
			noGit := errors.Is(gitErr, tracker.ErrNoGit)
			if gitErr != nil && !noGit {
				return fmt.Errorf("init git runner: %w", gitErr)
			}

			// 5. Parse --files filter into a set.
			filterSet := map[string]bool{}
			if filesFilter != "" {
				for _, f := range strings.Split(filesFilter, ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						filterSet[f] = true
					}
				}
			}

			// 6. Walk wiki root and collect entries.
			var internals []entryStatusInternal

			walkErr := filepath.WalkDir(cfg.WikiRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// Skip directories/files we cannot read.
					return nil
				}
				if d.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".md" {
					return nil
				}
				if filepath.Base(path) == "_index.md" {
					return nil
				}

				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil // skip unreadable files
				}

				// Relative file path for display.
				relPath, relErr := filepath.Rel(cfg.WikiRoot, path)
				if relErr != nil {
					relPath = path
				}

				// Try to parse as a ProjectEntry first, then ServiceEntry.
				// We include the entry when Meta.Path matches absProjectDir or
				// when Meta.Path is empty (unqualified entries).
				pEntry, pErr := wiki.ParseProjectEntry(data)
				if pErr == nil {
					match := pathMatches(pEntry.Meta.Path, absProjectDir)
					if !match {
						return nil
					}
					is := buildInternalFromProject(relPath, pEntry, noGit, filterSet, runner, absProjectDir)
					internals = append(internals, is)
					return nil
				}

				sEntry, sErr := wiki.ParseServiceEntry(data)
				if sErr == nil {
					match := pathMatches(sEntry.Meta.Path, absProjectDir)
					if !match {
						return nil
					}
					is := buildInternalFromService(relPath, sEntry, noGit, filterSet, runner, absProjectDir)
					internals = append(internals, is)
					return nil
				}

				return nil
			})
			if walkErr != nil {
				return fmt.Errorf("walk wiki root %q: %w", cfg.WikiRoot, walkErr)
			}

			// 7. Convert to public EntryStatus, check staleness.
			result := CheckResult{
				ProjectDir: absProjectDir,
				Entries:    make([]EntryStatus, 0, len(internals)),
			}
			for _, is := range internals {
				result.Entries = append(result.Entries, is.EntryStatus)
				if is.IsStale {
					result.AnyStale = true
				}
			}

			// 8. Output.
			if jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return fmt.Errorf("encode JSON: %w", err)
				}
			} else {
				printHumanReport(cmd, result)
			}

			// 9. --exit-code: signal staleness via process exit.
			if exitCode && result.AnyStale {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as indented JSON")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Exit with code 1 if any entry is stale")
	cmd.Flags().StringVar(&filesFilter, "files", "", "Comma-separated file list; only check entries whose tracked files overlap")
	cmd.Flags().StringVar(&wikiRootFlag, "wiki-root", "", "Override wiki root directory")

	return cmd
}

// buildInternalFromProject converts a ProjectEntry into an entryStatusInternal,
// applying git freshness check or appropriate note.
func buildInternalFromProject(relPath string, e wiki.ProjectEntry, noGit bool, filterSet map[string]bool, runner tracker.GitRunner, projectRoot string) entryStatusInternal {
	tracking := e.Meta.LLMWikiTracking
	is := entryStatusInternal{
		EntryStatus: EntryStatus{
			File:       relPath,
			Area:       tracking.Area,
			StoredHash: tracking.Hash,
			UpdatedAt:  tracking.UpdatedAt,
		},
		areaFiles: tracking.Files,
	}

	if tracking.Hash == "" {
		is.Note = "not tracked"
		return is
	}

	if noGit {
		is.Note = "git not available"
		return is
	}

	// --files filter: skip if no overlap.
	if len(filterSet) > 0 && !hasOverlap(tracking.Files, filterSet) {
		is.Note = "skipped (no file overlap)"
		return is
	}

	area := tracker.Area{
		Name:  tracking.Area,
		Files: tracking.Files,
	}
	result, err := tracker.CheckFreshness(runner, projectRoot, area, tracking.Hash)
	if err != nil {
		is.Note = fmt.Sprintf("error: %v", err)
		return is
	}
	is.IsStale = result.IsStale
	is.CurrentHash = result.CurrentHash
	return is
}

// buildInternalFromService converts a ServiceEntry into an entryStatusInternal.
func buildInternalFromService(relPath string, e wiki.ServiceEntry, noGit bool, filterSet map[string]bool, runner tracker.GitRunner, projectRoot string) entryStatusInternal {
	tracking := e.Meta.LLMWikiTracking
	is := entryStatusInternal{
		EntryStatus: EntryStatus{
			File:      relPath,
			Area:      tracking.Area,
			StoredHash: tracking.Hash,
			UpdatedAt: tracking.UpdatedAt,
		},
		areaFiles: tracking.Files,
	}

	if tracking.Hash == "" {
		is.Note = "not tracked"
		return is
	}

	if noGit {
		is.Note = "git not available"
		return is
	}

	if len(filterSet) > 0 && !hasOverlap(tracking.Files, filterSet) {
		is.Note = "skipped (no file overlap)"
		return is
	}

	area := tracker.Area{
		Name:  tracking.Area,
		Files: tracking.Files,
	}
	result, err := tracker.CheckFreshness(runner, projectRoot, area, tracking.Hash)
	if err != nil {
		is.Note = fmt.Sprintf("error: %v", err)
		return is
	}
	is.IsStale = result.IsStale
	is.CurrentHash = result.CurrentHash
	return is
}

// pathMatches returns true when metaPath matches the resolved absolute project
// directory, or when metaPath is empty or "." (meaning the entry is
// unqualified and applies to any project).
func pathMatches(metaPath, absProjectDir string) bool {
	if metaPath == "" || metaPath == "." {
		return true
	}
	if metaPath == absProjectDir {
		return true
	}
	// Resolve relative meta path against absProjectDir.
	if !filepath.IsAbs(metaPath) {
		resolved, err := filepath.Abs(metaPath)
		if err == nil && resolved == absProjectDir {
			return true
		}
	}
	return false
}

// hasOverlap returns true when at least one file in files appears in filterSet.
func hasOverlap(files []string, filterSet map[string]bool) bool {
	for _, f := range files {
		if filterSet[f] {
			return true
		}
	}
	return false
}

// printHumanReport writes human-readable output to the command's writer.
func printHumanReport(cmd *cobra.Command, result CheckResult) {
	out := cmd.OutOrStdout()
	if len(result.Entries) == 0 {
		fmt.Fprintln(out, "no wiki entries found for", result.ProjectDir)
		return
	}
	for _, e := range result.Entries {
		if e.Note != "" {
			fmt.Fprintf(out, "  %s (%s)\n", e.File, e.Note)
			continue
		}
		if e.IsStale {
			line := fmt.Sprintf("✗ %s\tSTALE", e.File)
			if e.Area != "" {
				line += fmt.Sprintf("\tarea: %s", e.Area)
			}
			fmt.Fprintln(out, line)
		} else {
			line := fmt.Sprintf("✓ %s\tfresh", e.File)
			if e.Area != "" {
				line += fmt.Sprintf("\tarea: %s", e.Area)
			}
			if e.UpdatedAt != "" {
				line += fmt.Sprintf("\tupdated: %s", e.UpdatedAt)
			}
			fmt.Fprintln(out, line)
		}
	}
}
