package safeio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mgz/llmwiki/internal/safeio"
)

func TestReadRegularFile_ReadsNormalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := safeio.ReadRegularFile(path)
	if err != nil {
		t.Fatalf("expected no error reading regular file, got: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected %q, got %q", "hello", string(data))
	}
}

func TestReadRegularFile_RefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	_, err := safeio.ReadRegularFile(link)
	if err == nil {
		t.Error("expected error reading symlink, got nil")
	}
}

func TestWriteRegularFile_RefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	err := safeio.WriteRegularFile(link, []byte("injected"), 0644)
	if err == nil {
		t.Error("expected error writing to symlink, got nil")
	}
}

func TestWriteRegularFile_WritesNormalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := safeio.WriteRegularFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("expected no error writing to new file, got: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("expected %q, got %q", "data", string(data))
	}
}
