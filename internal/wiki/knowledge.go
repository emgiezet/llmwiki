package wiki

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emgiezet/llmwiki/internal/safeio"
	"github.com/emgiezet/llmwiki/internal/validation"
)

// KnowledgeDir is the subdirectory of the wiki root holding the knowledge
// layers: <wiki_root>/knowledge/<layer>/*.md.
//
// Unlike the project layer, this tree is read by walking the filesystem rather
// than from _index.md. That keeps hand-written files and git submodules working
// with no bookkeeping — nothing has to write into a tracked index file for a
// layer to become visible.
const KnowledgeDir = "knowledge"

// KnowledgeEntry is one markdown file in a knowledge layer. Layer is the
// directory name, which is the single source of truth for which layer an
// entry belongs to — callers surface it as the attribution for the content.
type KnowledgeEntry struct {
	Layer string // directory name under KnowledgeDir
	Topic string // file basename without the .md extension
	Path  string // path relative to Store.Root
	Title string // first "# " heading, falling back to Topic
	Body  string // file content with any YAML front matter stripped
}

// SearchKnowledge returns the entries of the given layers. An empty query
// returns everything, so this doubles as the list operation.
//
// Results come back in the caller's layer order — that order *is* the lookup
// priority, so pass the most specific layer first (config.Merged.Knowledge is
// already ordered this way). Within a layer, entries are sorted by path.
//
// A layer directory that doesn't exist is skipped: layers are optional, and an
// install with no knowledge/ tree at all returns an empty result, not an error.
func (s *Store) SearchKnowledge(layers []string, query string) ([]KnowledgeEntry, error) {
	q := strings.ToLower(query)
	var out []KnowledgeEntry

	for _, layer := range layers {
		if err := validation.NameComponent("knowledge layer", layer); err != nil {
			return nil, err
		}
		entries, err := s.readLayer(layer)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if q == "" || matchesKnowledge(e, q) {
				out = append(out, e)
			}
		}
	}
	return out, nil
}

// GetKnowledge returns a single entry by layer and topic. Topic is matched
// against the file basename, including files in nested subdirectories of the
// layer (so knowledge/global/adr/001-postgres.md is topic "001-postgres").
func (s *Store) GetKnowledge(layer, topic string) (KnowledgeEntry, error) {
	if err := validation.NameComponent("knowledge layer", layer); err != nil {
		return KnowledgeEntry{}, err
	}
	if err := validation.NameComponent("topic", topic); err != nil {
		return KnowledgeEntry{}, err
	}
	entries, err := s.readLayer(layer)
	if err != nil {
		return KnowledgeEntry{}, err
	}
	for _, e := range entries {
		if strings.EqualFold(e.Topic, topic) {
			return e, nil
		}
	}
	return KnowledgeEntry{}, fmt.Errorf("no knowledge entry %q in layer %q", topic, layer)
}

// ListKnowledgeLayers returns the layer names present on disk, sorted. Used to
// answer "all layers" when a caller doesn't name one, and to let agents
// discover which layers exist. Dot-directories (.git in a submodule checkout)
// and loose files directly under knowledge/ are not layers.
func (s *Store) ListKnowledgeLayers() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, KnowledgeDir))
	if err != nil {
		// No knowledge tree yet is the normal state for a fresh install.
		return nil, nil //nolint:nilerr // absent knowledge dir is not an error
	}
	var layers []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		layers = append(layers, e.Name())
	}
	sort.Strings(layers)
	return layers, nil
}

// readLayer walks one layer directory and parses every markdown file in it.
// Entries come back sorted by relative path (WalkDir walks lexically).
func (s *Store) readLayer(layer string) ([]KnowledgeEntry, error) {
	layerDir := filepath.Join(s.Root, KnowledgeDir, layer)
	if _, err := os.Stat(layerDir); err != nil {
		// A missing or unreadable layer is a no-op, not a failure.
		return nil, nil //nolint:nilerr // absent layers are normal
	}

	var out []KnowledgeEntry
	err := filepath.WalkDir(layerDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Submodule checkouts leave a .git directory inside the layer.
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		// safeio refuses symlinks and other non-regular files; skip rather
		// than abort so one bad entry can't hide a whole layer.
		data, readErr := safeio.ReadRegularFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(s.Root, path)
		if relErr != nil {
			return nil
		}
		topic := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		// Front matter is optional: splitFrontMatter returns the whole file as
		// the body when there is none, so hand-written files work unchanged.
		_, body := splitFrontMatter(data)
		out = append(out, KnowledgeEntry{
			Layer: layer,
			Topic: topic,
			Path:  rel,
			Title: knowledgeTitle(string(body), topic),
			Body:  strings.TrimSpace(string(body)),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read knowledge layer %q: %w", layer, err)
	}
	return out, nil
}

// knowledgeTitle returns the first "# " heading of body, or fallback when the
// file has no H1 (common for ingest-generated entries, which start at "## ").
func knowledgeTitle(body, fallback string) string {
	for line := range strings.SplitSeq(body, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(line), "# "); ok {
			if t = strings.TrimSpace(t); t != "" {
				return t
			}
		}
	}
	return fallback
}

// matchesKnowledge reports whether the entry matches an already-lowercased
// query as a substring of its topic, title, or body.
func matchesKnowledge(e KnowledgeEntry, lowerQuery string) bool {
	return strings.Contains(strings.ToLower(e.Topic), lowerQuery) ||
		strings.Contains(strings.ToLower(e.Title), lowerQuery) ||
		strings.Contains(strings.ToLower(e.Body), lowerQuery)
}
