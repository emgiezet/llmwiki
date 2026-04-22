package wiki

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProjectMeta is YAML front matter for a project _index.md.
type ProjectMeta struct {
	Name         string    `yaml:"name"`
	Customer     string    `yaml:"customer"`
	Type         string    `yaml:"type"`
	Status       string    `yaml:"status"`
	Path         string    `yaml:"path"`
	LLM          string    `yaml:"llm"`
	OllamaModel  string    `yaml:"ollama_model,omitempty"`
	Tags         []string  `yaml:"tags,omitempty"`
	LastIngested time.Time `yaml:"last_ingested"`
}

// ServiceMeta is YAML front matter for a per-service file.
type ServiceMeta struct {
	Service      string    `yaml:"service"`
	Project      string    `yaml:"project"`
	Customer     string    `yaml:"customer"`
	Language     string    `yaml:"language,omitempty"`
	Path         string    `yaml:"path,omitempty"`
	Exposes      []string  `yaml:"exposes,omitempty"`
	DependsOn    []string  `yaml:"depends_on,omitempty"`
	Tags         []string  `yaml:"tags,omitempty"`
	LastIngested time.Time `yaml:"last_ingested"`
}

// ProjectEntry holds parsed project wiki file.
type ProjectEntry struct {
	Meta ProjectMeta
	Body string
}

// ServiceEntry holds parsed service wiki file.
type ServiceEntry struct {
	Meta ServiceMeta
	Body string
}

// ClientMeta is YAML front matter for a per-client _index.md.
type ClientMeta struct {
	Customer      string    `yaml:"customer"`
	Projects      []string  `yaml:"projects"`
	Tags          []string  `yaml:"tags,omitempty"`
	LastGenerated time.Time `yaml:"last_generated"`
}

// MultiProjectMeta is YAML front matter for a multi-service project _index.md.
type MultiProjectMeta struct {
	Name          string    `yaml:"name"`
	Customer      string    `yaml:"customer"`
	Type          string    `yaml:"type"`
	Status        string    `yaml:"status"`
	Services      []string  `yaml:"services"`
	Tags          []string  `yaml:"tags,omitempty"`
	LastGenerated time.Time `yaml:"last_generated"`
}

// ClientEntry holds parsed client index wiki file.
type ClientEntry struct {
	Meta ClientMeta
	Body string
}

// MultiProjectEntry holds parsed multi-service project index.
type MultiProjectEntry struct {
	Meta MultiProjectMeta
	Body string
}

func ParseClientEntry(data []byte) (ClientEntry, error) {
	fm, body := splitFrontMatter(data)
	var meta ClientMeta
	if fm != nil {
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return ClientEntry{}, fmt.Errorf("parse client front matter: %w", err)
		}
	}
	return ClientEntry{Meta: meta, Body: string(body)}, nil
}

func WriteClientEntry(path string, meta ClientMeta, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec G301 -- wiki dirs are world-readable by design
		return err
	}
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n%s", fm, body)
	return os.WriteFile(path, []byte(content), 0644) // #nosec G306 -- wiki files are world-readable by design
}

func ParseMultiProjectEntry(data []byte) (MultiProjectEntry, error) {
	fm, body := splitFrontMatter(data)
	var meta MultiProjectMeta
	if fm != nil {
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return MultiProjectEntry{}, fmt.Errorf("parse multi-project front matter: %w", err)
		}
	}
	return MultiProjectEntry{Meta: meta, Body: string(body)}, nil
}

func WriteMultiProjectEntry(path string, meta MultiProjectMeta, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec G301 -- wiki dirs are world-readable by design
		return err
	}
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n%s", fm, body)
	return os.WriteFile(path, []byte(content), 0644) // #nosec G306 -- wiki files are world-readable by design
}

var separator = []byte("---\n")

func splitFrontMatter(data []byte) ([]byte, []byte) {
	if !bytes.HasPrefix(data, separator) {
		return nil, data
	}
	rest := data[len(separator):]
	end := bytes.Index(rest, separator)
	if end == -1 {
		return nil, data
	}
	return rest[:end], rest[end+len(separator):]
}

func ParseProjectEntry(data []byte) (ProjectEntry, error) {
	fm, body := splitFrontMatter(data)
	var meta ProjectMeta
	if fm != nil {
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return ProjectEntry{}, fmt.Errorf("parse front matter: %w", err)
		}
	}
	return ProjectEntry{Meta: meta, Body: string(body)}, nil
}

func ParseServiceEntry(data []byte) (ServiceEntry, error) {
	fm, body := splitFrontMatter(data)
	var meta ServiceMeta
	if fm != nil {
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			return ServiceEntry{}, fmt.Errorf("parse service front matter: %w", err)
		}
	}
	return ServiceEntry{Meta: meta, Body: string(body)}, nil
}

func WriteProjectEntry(path string, meta ProjectMeta, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec G301 -- wiki dirs are world-readable by design
		return err
	}
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n%s", fm, body)
	return os.WriteFile(path, []byte(content), 0644) // #nosec G306 -- wiki files are world-readable by design
}

func WriteServiceEntry(path string, meta ServiceMeta, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec G301 -- wiki dirs are world-readable by design
		return err
	}
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n%s", fm, body)
	return os.WriteFile(path, []byte(content), 0644) // #nosec G306 -- wiki files are world-readable by design
}
