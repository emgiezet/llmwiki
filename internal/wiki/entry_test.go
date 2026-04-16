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
