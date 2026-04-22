package ingestion_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const materializeFakeBody = `## Domain
Insurance quoting platform.

## Architecture
Monolith written in Go.

## Services
- api: main HTTP service

## Features
Quote management.

## Flows
user → api → db

## System Diagram
` + "```mermaid\nflowchart TD\n  api --> db\n```" + `

## Data Model Diagram
No database schema detected in accumulated facts.

## Integrations
PostgreSQL

## Tech Stack
Go, PostgreSQL

## Configuration
PORT=8080

## Notes
None.

## Tags
go, postgres, rest`

func TestMaterializeFromMemory_NoMemory_ReturnsError(t *testing.T) {
	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, Type: "client", Customer: "acme"}
	err := ingestion.MaterializeFromMemory(context.Background(), "myapp", cfg, llm.NewFakeLLM(""), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory")
}

func TestMaterializeFromMemory_ExistingWikiNoFacts_Updates(t *testing.T) {
	wikiRoot := t.TempDir()
	memDir := t.TempDir()

	// Write a pre-existing wiki entry.
	wikiPath := filepath.Join(wikiRoot, "clients", "acme", "myapp.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(wikiPath), 0755))
	meta := wiki.ProjectMeta{Name: "myapp", Customer: "acme", Type: "client", Status: "active", LLM: "claude-code"}
	require.NoError(t, wiki.WriteProjectEntry(wikiPath, meta, "\n## Domain\nOld description.\n"))

	mem := memory.New(memDir)
	defer mem.Close()

	fakeLLM := llm.NewFakeLLM(materializeFakeBody)
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "acme", Type: "client"}

	err := ingestion.MaterializeFromMemory(context.Background(), "myapp", cfg, fakeLLM, mem)
	require.NoError(t, err)

	data, err := os.ReadFile(wikiPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "## Domain")
	assert.Contains(t, string(data), "Insurance quoting platform")

	entry, err := wiki.ParseProjectEntry(data)
	require.NoError(t, err)
	assert.Contains(t, entry.Meta.Tags, "go")
}

func TestMaterializeFromMemory_NoFactsNoWiki_ReturnsError(t *testing.T) {
	wikiRoot := t.TempDir()
	memDir := t.TempDir()
	mem := memory.New(memDir)
	defer mem.Close()

	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "acme", Type: "client"}
	err := ingestion.MaterializeFromMemory(context.Background(), "unknownproject", cfg, llm.NewFakeLLM(""), mem)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no memory facts")
}

func TestMaterializeFromMemory_UpdatesIndex(t *testing.T) {
	wikiRoot := t.TempDir()
	memDir := t.TempDir()

	// Pre-existing wiki so materialize has something to update.
	wikiPath := filepath.Join(wikiRoot, "clients", "acme", "myapp.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(wikiPath), 0755))
	meta := wiki.ProjectMeta{Name: "myapp", Customer: "acme", Type: "client", Status: "active", LLM: "claude-code"}
	require.NoError(t, wiki.WriteProjectEntry(wikiPath, meta, "\n## Domain\nOld.\n"))

	mem := memory.New(memDir)
	defer mem.Close()

	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "acme", Type: "client"}
	err := ingestion.MaterializeFromMemory(context.Background(), "myapp", cfg, llm.NewFakeLLM(materializeFakeBody), mem)
	require.NoError(t, err)

	indexData, err := os.ReadFile(filepath.Join(wikiRoot, "_index.md"))
	require.NoError(t, err)
	assert.Contains(t, string(indexData), "myapp")
}
