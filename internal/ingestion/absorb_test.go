package ingestion_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsorbSession_NilMemory_NoOp(t *testing.T) {
	projectDir := t.TempDir()
	err := ingestion.AbsorbSession(context.Background(), projectDir, "myproject", "", "some note", nil)
	assert.NoError(t, err)
}

func TestAbsorbSession_NotGitRepo_ContinuesWithNote(t *testing.T) {
	projectDir := t.TempDir() // not a git repo
	memDir := t.TempDir()
	mem := memory.New(memDir)
	defer mem.Close()
	// Should not fail even though git commands will fail — falls back to note only.
	err := ingestion.AbsorbSession(context.Background(), projectDir, "notgit", "", "manual note about the session", mem)
	assert.NoError(t, err)
}

func TestAbsorbSession_DefaultsProjectNameToDirBasename(t *testing.T) {
	projectDir := t.TempDir()
	memDir := t.TempDir()
	mem := memory.New(memDir)
	defer mem.Close()
	// Empty projectName — should infer from dir basename without panicking.
	err := ingestion.AbsorbSession(context.Background(), projectDir, "", "", "some note", mem)
	assert.NoError(t, err)
}

func TestAbsorbSession_GitRepo_StoresFacts(t *testing.T) {
	projectDir := t.TempDir()
	memDir := t.TempDir()

	initGitRepo(t, projectDir)

	mem := memory.New(memDir)
	err := ingestion.AbsorbSession(context.Background(), projectDir, "myproject", "acme", "integrated Stripe payments", mem)
	require.NoError(t, err)
	mem.Close() // flush async writes

	mem2 := memory.New(memDir)
	defer mem2.Close()
	facts, err := mem2.RecallForProject(context.Background(), "myproject", "acme")
	require.NoError(t, err)
	assert.NotEmpty(t, facts)
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644))
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "feat: initial commit"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
}
