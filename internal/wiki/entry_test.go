package wiki_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mgz/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProjectEntry_YAMLBomb(t *testing.T) {
	// Small, bounded alias reference — real parsers may still accept this
	// without blowing memory because yaml.v3 limits alias recursion.
	bomb := []byte(`---
a: &a ["x","x","x","x","x","x","x","x","x","x"]
b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a,*a]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b,*b]
name: stillvalid
---
body`)

	// Should return in reasonable time without panicking.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = wiki.ParseProjectEntry(bomb)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("parser hung on YAML alias bomb")
	}
}

func TestParseProjectEntry_FrontMatter(t *testing.T) {
	content := `---
name: myproject
customer: acme
type: client
status: active
path: /home/mgz/workspace/myproject
llm: claude-code
tags: [go, grpc]
last_ingested: 2026-04-16T10:00:00Z
---
# myproject

## Domain
Insurance platform.
`
	entry, err := wiki.ParseProjectEntry([]byte(content))
	require.NoError(t, err)
	assert.Equal(t, "myproject", entry.Meta.Name)
	assert.Equal(t, "acme", entry.Meta.Customer)
	assert.Equal(t, "client", entry.Meta.Type)
	assert.Equal(t, "claude-code", entry.Meta.LLM)
	assert.Contains(t, entry.Body, "## Domain")
}

func TestParseProjectEntry_NoFrontMatter(t *testing.T) {
	content := "# myproject\n\n## Domain\nStuff.\n"
	entry, err := wiki.ParseProjectEntry([]byte(content))
	require.NoError(t, err)
	assert.Equal(t, "", entry.Meta.Name)
	assert.Contains(t, entry.Body, "## Domain")
}

func TestWriteProjectEntry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myproject.md")

	meta := wiki.ProjectMeta{
		Name:     "myproject",
		Customer: "acme",
		Type:     "client",
		Status:   "active",
		LLM:      "claude-code",
		Tags:     []string{"go", "grpc"},
		LastIngested: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
	}
	body := "# myproject\n\n## Domain\nInsurance platform.\n"

	require.NoError(t, wiki.WriteProjectEntry(path, meta, body))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	entry, err := wiki.ParseProjectEntry(data)
	require.NoError(t, err)
	assert.Equal(t, "myproject", entry.Meta.Name)
	assert.Equal(t, "acme", entry.Meta.Customer)
	assert.Contains(t, entry.Body, "## Domain")
}

func TestParseServiceEntry_FrontMatter(t *testing.T) {
	content := `---
service: api-gateway
project: myproject
customer: acme
language: go
path: ./services/api-gateway
exposes: [REST :8080, gRPC :9090]
depends_on: [policy-service]
last_ingested: 2026-04-16T10:00:00Z
---
# api-gateway

## Purpose
Routes traffic.
`
	entry, err := wiki.ParseServiceEntry([]byte(content))
	require.NoError(t, err)
	assert.Equal(t, "api-gateway", entry.Meta.Service)
	assert.Equal(t, []string{"REST :8080", "gRPC :9090"}, entry.Meta.Exposes)
	assert.Contains(t, entry.Body, "## Purpose")
}

func TestWriteReadIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "_index.md")

	entries := []wiki.IndexEntry{
		{Name: "mmx3", Customer: "insly", Type: "client", Status: "active", WikiPath: "clients/insly/mmx3/_index.md"},
		{Name: "llmwiki", Customer: "", Type: "personal", Status: "active", WikiPath: "personal/llmwiki.md"},
	}

	require.NoError(t, wiki.WriteIndex(indexPath, entries))

	data, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "mmx3")
	assert.Contains(t, string(data), "insly")
	assert.Contains(t, string(data), "personal")

	readEntries, err := wiki.ReadIndex(indexPath)
	require.NoError(t, err)
	assert.Len(t, readEntries, 2)
	assert.Equal(t, "mmx3", readEntries[0].Name)
}

func TestUpsertIndex_AddsNew(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "_index.md")

	require.NoError(t, wiki.UpsertIndex(indexPath, wiki.IndexEntry{
		Name: "project-a", Type: "client", Customer: "acme", Status: "active", WikiPath: "clients/acme/project-a.md",
	}))
	require.NoError(t, wiki.UpsertIndex(indexPath, wiki.IndexEntry{
		Name: "project-b", Type: "personal", Status: "active", WikiPath: "personal/project-b.md",
	}))

	entries, err := wiki.ReadIndex(indexPath)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestUpsertIndex_UpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "_index.md")

	require.NoError(t, wiki.UpsertIndex(indexPath, wiki.IndexEntry{Name: "proj", Type: "client", Customer: "acme", Status: "active", WikiPath: "clients/acme/proj.md"}))
	require.NoError(t, wiki.UpsertIndex(indexPath, wiki.IndexEntry{Name: "proj", Type: "client", Customer: "acme", Status: "archived", WikiPath: "clients/acme/proj.md"}))

	entries, err := wiki.ReadIndex(indexPath)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "archived", entries[0].Status)
}
