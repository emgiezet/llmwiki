package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emgiezet/llmwiki/internal/safeio"
)

// Store provides read-only, LLM-free queries over an extracted wiki tree
// rooted at Root. It is the data source backing the MCP server.
type Store struct {
	Root string
}

// ProjectMatch is a single project surfaced by a query: index metadata plus
// the tags and a short Domain summary read from the entry file.
type ProjectMatch struct {
	Name     string
	Customer string
	Type     string
	Status   string
	WikiPath string
	Tags     []string
	Summary  string
}

// summaryMaxChars bounds the Domain snippet returned by Search.
const summaryMaxChars = 600

// NewStore returns a Store rooted at the given wiki root directory.
func NewStore(wikiRoot string) *Store {
	return &Store{Root: wikiRoot}
}

// Search returns the projects in _index.md matching the optional filters.
// Both filters are optional: an empty client/project matches everything.
// client is matched case-insensitively and exactly against the customer;
// project is matched case-insensitively as a substring of the project name.
func (s *Store) Search(client, project string) ([]ProjectMatch, error) {
	entries, err := ReadIndex(filepath.Join(s.Root, "_index.md"))
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}

	var matches []ProjectMatch
	for _, e := range filterEntries(entries, client, project) {
		m := ProjectMatch{
			Name:     e.Name,
			Customer: e.Customer,
			Type:     e.Type,
			Status:   e.Status,
			WikiPath: e.WikiPath,
		}
		// Best-effort enrichment: tags and Domain summary from the entry file.
		if data, readErr := safeio.ReadRegularFile(filepath.Join(s.Root, e.WikiPath)); readErr == nil {
			entry, parseErr := ParseProjectEntry(data)
			if parseErr == nil {
				m.Tags = entry.Meta.Tags
				m.Summary = TruncateSection(ExtractSection(entry.Body, "## Domain"), summaryMaxChars)
			}
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// GetProject returns the full extracted body of a single project entry. When
// service is non-empty, the body of that service file is returned instead.
// The match is resolved with the same filters as Search; ambiguous matches
// (more than one project) return an error listing the candidates.
func (s *Store) GetProject(client, project, service string) (string, ProjectMatch, error) {
	matches, err := s.Search(client, project)
	if err != nil {
		return "", ProjectMatch{}, err
	}
	switch len(matches) {
	case 0:
		return "", ProjectMatch{}, fmt.Errorf("no project found for client=%q project=%q", client, project)
	case 1:
		// ok
	default:
		return "", ProjectMatch{}, fmt.Errorf("ambiguous project %q matches %d projects: %s (pass a client to disambiguate)",
			project, len(matches), strings.Join(candidateLabels(matches), ", "))
	}

	m := matches[0]
	multi := isMultiServicePath(m.WikiPath)

	if service != "" {
		if !multi {
			return "", ProjectMatch{}, fmt.Errorf("project %q is single-service; no service %q", m.Name, service)
		}
		svcPath := filepath.Join(s.Root, filepath.Dir(m.WikiPath), service+".md")
		data, readErr := safeio.ReadRegularFile(svcPath)
		if readErr != nil {
			return "", ProjectMatch{}, fmt.Errorf("read service %q: %w", service, readErr)
		}
		entry, parseErr := ParseServiceEntry(data)
		if parseErr != nil {
			return "", ProjectMatch{}, parseErr
		}
		return entry.Body, m, nil
	}

	data, readErr := safeio.ReadRegularFile(filepath.Join(s.Root, m.WikiPath))
	if readErr != nil {
		return "", ProjectMatch{}, fmt.Errorf("read project %q: %w", m.Name, readErr)
	}
	entry, parseErr := ParseProjectEntry(data)
	if parseErr != nil {
		return "", ProjectMatch{}, parseErr
	}
	return entry.Body, m, nil
}

// ListServices returns the service names of a multi-service project, given the
// wiki_path of its index file. Services are the *.md files in the project
// directory excluding the *_index.md file, sorted alphabetically.
func (s *Store) ListServices(wikiPath string) ([]string, error) {
	dir := filepath.Join(s.Root, filepath.Dir(wikiPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read project dir: %w", err)
	}
	var services []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, "_index.md") {
			continue
		}
		services = append(services, strings.TrimSuffix(name, ".md"))
	}
	sort.Strings(services)
	return services, nil
}

func filterEntries(entries []IndexEntry, client, project string) []IndexEntry {
	var out []IndexEntry
	for _, e := range entries {
		if client != "" && !strings.EqualFold(e.Customer, client) {
			continue
		}
		if project != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(project)) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func candidateLabels(matches []ProjectMatch) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		if m.Customer != "" {
			out[i] = m.Customer + "/" + m.Name
		} else {
			out[i] = m.Name
		}
	}
	return out
}

// isMultiServicePath reports whether a wiki_path points at a multi-service
// project index (one file per service in a directory) rather than a single file.
func isMultiServicePath(wikiPath string) bool {
	return strings.HasSuffix(filepath.Base(wikiPath), "_index.md")
}
