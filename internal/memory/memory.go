package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/angelnicolasc/graymatter"
	"github.com/emgiezet/llmwiki/internal/config"
)

// Store wraps graymatter.Memory with llmwiki-specific agent naming and nil-safe no-op behavior.
type Store struct {
	mem *graymatter.Memory
}

// New creates a Store. If dataDir is empty, memory is disabled (no-op).
func New(dataDir string) *Store {
	if dataDir == "" {
		return &Store{}
	}
	mem := graymatter.New(dataDir)
	return &Store{mem: mem}
}

// NewFromConfig builds a Store from llmwiki merged config.
// Returns a no-op Store if memory is disabled.
func NewFromConfig(cfg config.Merged) (*Store, error) {
	if !cfg.MemoryEnabled {
		return &Store{}, nil
	}
	gmCfg := graymatter.DefaultConfig()
	gmCfg.DataDir = cfg.MemoryDir
	gmCfg.EmbeddingMode = graymatter.EmbeddingAuto
	gmCfg.AsyncConsolidate = true
	gmCfg.DecayHalfLife = 30 * 24 * time.Hour
	if cfg.AnthropicAPIKey != "" {
		gmCfg.AnthropicAPIKey = cfg.AnthropicAPIKey
	}
	if cfg.OllamaHost != "" {
		gmCfg.OllamaURL = cfg.OllamaHost
	}

	mem, err := graymatter.NewWithConfig(gmCfg)
	if err != nil {
		return nil, fmt.Errorf("init graymatter: %w", err)
	}
	return &Store{mem: mem}, nil
}

// Enabled returns true if the memory backend is active.
func (s *Store) Enabled() bool {
	return s != nil && s.mem != nil && s.mem.Healthy()
}

// Close releases the underlying graymatter handle. We cap the wait at 3s
// because graymatter's internal Close() blocks on async consolidation
// goroutines that may make LLM calls; in hook-triggered paths we can't
// afford to hold the bbolt lock indefinitely. bbolt is crash-safe, so
// any interrupted writes recover on next open.
func (s *Store) Close() error {
	if s == nil || s.mem == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- s.mem.Close()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		return nil
	}
}

func projectAgent(name string) string  { return "llmwiki/project/" + name }
func customerAgent(name string) string { return "llmwiki/customer/" + name }

// RecallForProject retrieves relevant facts about a project for prompt enrichment.
func (s *Store) RecallForProject(ctx context.Context, projectName, customer string) (string, error) {
	if !s.Enabled() {
		return "", nil
	}

	query := fmt.Sprintf("architecture integrations tech stack for %s", projectName)
	projectFacts, err := s.mem.Recall(ctx, projectAgent(projectName), query)
	if err != nil {
		return "", err
	}

	var customerFacts []string
	if customer != "" {
		q := fmt.Sprintf("common patterns shared infrastructure for %s", customer)
		customerFacts, _ = s.mem.Recall(ctx, customerAgent(customer), q)
	}

	all := append(projectFacts, customerFacts...)
	if len(all) == 0 {
		return "", nil
	}
	return "PREVIOUS KNOWLEDGE (from earlier ingestions and cross-project analysis):\n" +
		strings.Join(all, "\n") + "\n", nil
}

// RecallForContext retrieves facts for CLAUDE.md context injection.
func (s *Store) RecallForContext(ctx context.Context, projectName, customer string) (string, error) {
	if !s.Enabled() {
		return "", nil
	}

	query := fmt.Sprintf("key facts architecture integrations for %s", projectName)
	facts, err := s.mem.Recall(ctx, projectAgent(projectName), query)
	if err != nil {
		return "", err
	}

	var customerFacts []string
	if customer != "" {
		q := fmt.Sprintf("cross-project patterns for %s", customer)
		customerFacts, _ = s.mem.Recall(ctx, customerAgent(customer), q)
	}

	all := append(facts, customerFacts...)
	if len(all) == 0 {
		return "", nil
	}
	return "\n## Related Knowledge\n" + strings.Join(all, "\n") + "\n", nil
}

// RecallForQuery retrieves semantically relevant facts for the query command.
func (s *Store) RecallForQuery(ctx context.Context, question string) ([]string, error) {
	if !s.Enabled() {
		return nil, nil
	}
	return s.mem.RecallShared(ctx, question)
}

// RememberIngestion stores facts from a completed project ingestion.
func (s *Store) RememberIngestion(ctx context.Context, projectName, customer, wikiBody string, tags []string) error {
	if !s.Enabled() {
		return nil
	}

	agent := projectAgent(projectName)

	// Auto-extract atomic facts from the wiki body.
	_ = s.mem.RememberExtracted(ctx, agent, wikiBody)

	// Store structured metadata.
	meta := fmt.Sprintf("Project %s (customer: %s) uses: %s",
		projectName, customer, strings.Join(tags, ", "))
	_ = s.mem.Remember(ctx, agent, meta)

	// Also store in customer agent for cross-project recall.
	if customer != "" {
		summary := truncate(wikiBody, 500)
		_ = s.mem.Remember(ctx, customerAgent(customer),
			fmt.Sprintf("Project %s: %s", projectName, summary))
	}

	// Store in shared namespace for cross-agent query recall.
	_ = s.mem.RememberShared(ctx, fmt.Sprintf("Project %s (%s): %s",
		projectName, customer, strings.Join(tags, ", ")))

	return nil
}

// RememberServiceIngestion stores facts from a completed service ingestion.
func (s *Store) RememberServiceIngestion(ctx context.Context, projectName, serviceName, customer, wikiBody string, tags []string) error {
	if !s.Enabled() {
		return nil
	}

	agent := projectAgent(projectName)

	_ = s.mem.RememberExtracted(ctx, agent, wikiBody)

	meta := fmt.Sprintf("Service %s in project %s uses: %s",
		serviceName, projectName, strings.Join(tags, ", "))
	_ = s.mem.Remember(ctx, agent, meta)

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
