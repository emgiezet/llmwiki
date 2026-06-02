// Package extractor pulls plain text out of non-UTF8 document files (PDF,
// DOCX, ODT, EPUB, …) by shelling out to user-configurable external tools.
//
// The scanner reads only UTF-8 text directly; binary document formats need a
// converter (pandoc, pdftotext, …). Which tool handles which extension is
// configured in llmwiki's config (per machine / OS), so the same binary works
// on macOS, Linux, and — with different command values — Windows.
//
// The subprocess plumbing (5-minute default timeout, stderr redaction, stdout
// trimming) mirrors internal/llm/cli_backend.go so behaviour stays consistent
// with the agentic-coder CLI backends.
package extractor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// inputPlaceholder is replaced with the document's path inside a command
// template, e.g. "pdftotext {{input}} -".
const inputPlaceholder = "{{input}}"

// ErrToolNotFound is returned (wrapped) when the configured extractor binary
// is not on PATH. Callers (the scanner) treat this as "skip this file", not a
// fatal error — a machine without pandoc installed simply can't read .docx.
var ErrToolNotFound = errors.New("extractor tool not found in PATH")

// Extractor pulls plain text out of a document file.
type Extractor interface {
	// CanExtract reports whether a command template is configured for the
	// file's extension.
	CanExtract(name string) bool
	// Extract runs the configured command and returns the file's text.
	Extract(ctx context.Context, path string) (string, error)
}

// CommandExtractor runs one configured external command per file extension.
type CommandExtractor struct {
	// commands maps a lower-cased, dot-prefixed extension (".pdf") to a
	// command template containing the {{input}} placeholder.
	commands map[string]string
}

// New builds a CommandExtractor from an extension→command-template map. Keys
// are normalised to lower-case with a leading dot, so "pdf", ".PDF", and
// ".pdf" are equivalent. A nil or empty map yields an extractor that handles
// nothing (CanExtract always returns false).
func New(commands map[string]string) *CommandExtractor {
	norm := make(map[string]string, len(commands))
	for ext, tmpl := range commands {
		ext = normalizeExt(ext)
		if ext == "" || strings.TrimSpace(tmpl) == "" {
			continue
		}
		norm[ext] = tmpl
	}
	return &CommandExtractor{commands: norm}
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// CanExtract reports whether a command template is configured for name's
// extension.
func (e *CommandExtractor) CanExtract(name string) bool {
	if e == nil {
		return false
	}
	_, ok := e.commands[strings.ToLower(filepath.Ext(name))]
	return ok
}

// Extract runs the configured command for path's extension and returns trimmed
// stdout. The command must write the extracted text to stdout. A missing
// binary yields ErrToolNotFound (wrapped); a non-zero exit yields an error
// carrying the last 512 bytes of stderr.
func (e *CommandExtractor) Extract(ctx context.Context, path string) (string, error) {
	tmpl, ok := e.commands[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", fmt.Errorf("no extractor configured for %q", filepath.Ext(path))
	}

	argv := buildArgv(tmpl, path)
	if len(argv) == 0 {
		return "", fmt.Errorf("empty extractor command for %q", filepath.Ext(path))
	}

	// Resolve the binary up front so a missing tool is a clean, skippable
	// sentinel rather than an opaque exec error.
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, argv[0])
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, bin, argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("extract %s failed: %w\nstderr (last 512 bytes): %s",
			filepath.Base(path), err, tail(stderr.String(), 512))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// buildArgv splits a command template on whitespace and substitutes the
// {{input}} placeholder with path. The template is NOT run through a shell, so
// pipes/redirects are not interpreted and paths with spaces are safe (the
// placeholder is substituted into a single argv element). If no token contains
// the placeholder, path is appended as the final argument.
func buildArgv(tmpl, path string) []string {
	fields := strings.Fields(tmpl)
	argv := make([]string, 0, len(fields)+1)
	substituted := false
	for _, f := range fields {
		if strings.Contains(f, inputPlaceholder) {
			f = strings.ReplaceAll(f, inputPlaceholder, path)
			substituted = true
		}
		argv = append(argv, f)
	}
	if !substituted && len(argv) > 0 {
		argv = append(argv, path)
	}
	return argv
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
