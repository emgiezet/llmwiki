package wiki

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IndexEntry is one row in _index.md.
type IndexEntry struct {
	Name     string `yaml:"name"`
	Customer string `yaml:"customer,omitempty"`
	Type     string `yaml:"type"`
	Status   string `yaml:"status"`
	WikiPath string `yaml:"wiki_path"`
}

type indexFile struct {
	Projects []IndexEntry `yaml:"projects"`
}

// WriteIndex writes all entries to path as YAML front matter + markdown table.
func WriteIndex(path string, entries []IndexEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec G301 -- wiki dirs are world-readable by design
		return err
	}
	data := indexFile{Projects: entries}
	fm, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "---\n%s---\n\n# Project Index\n\n", fm)
	fmt.Fprintf(&buf, "| Project | Customer | Type | Status | Wiki |\n")
	fmt.Fprintf(&buf, "|---------|----------|------|--------|------|\n")
	for _, e := range entries {
		fmt.Fprintf(&buf, "| %s | %s | %s | %s | [link](%s) |\n",
			e.Name, e.Customer, e.Type, e.Status, e.WikiPath)
	}
	return os.WriteFile(path, buf.Bytes(), 0644) // #nosec G306 -- wiki index is world-readable by design
}

// ReadIndex reads entries from _index.md front matter.
func ReadIndex(path string) ([]IndexEntry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fm, _ := splitFrontMatter(data)
	if fm == nil {
		return nil, nil
	}
	var idx indexFile
	if err := yaml.Unmarshal(fm, &idx); err != nil {
		return nil, err
	}
	return idx.Projects, nil
}

// UpsertIndex adds or updates an entry in the index file.
func UpsertIndex(path string, entry IndexEntry) error {
	entries, err := ReadIndex(path)
	if err != nil {
		return err
	}
	updated := false
	for i, e := range entries {
		if strings.EqualFold(e.Name, entry.Name) {
			entries[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		entries = append(entries, entry)
	}
	return WriteIndex(path, entries)
}
