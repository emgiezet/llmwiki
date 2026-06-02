// Package wizard provides dependency-free interactive prompt helpers for the
// llmwiki CLI. All I/O goes through an injected reader/writer so callers can
// be tested without a terminal.
package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Option is one selectable entry in a Choice prompt. Value is what gets stored
// in config; Label is the human-readable text shown to the user.
type Option struct {
	Value string
	Label string
}

// Prompter reads answers from in and writes prompts to out. Once the input is
// exhausted (EOF), every subsequent prompt returns its default rather than
// blocking or looping.
type Prompter struct {
	in  *bufio.Scanner
	out io.Writer
	eof bool
}

// New returns a Prompter reading from in and writing to out.
func New(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewScanner(in), out: out}
}

// readLine returns the next trimmed input line. After EOF it sets p.eof and
// returns "" for all further calls.
func (p *Prompter) readLine() string {
	if p.eof {
		return ""
	}
	if p.in.Scan() {
		return strings.TrimSpace(p.in.Text())
	}
	p.eof = true
	return ""
}

// Note prints an informational line (no input read).
func (p *Prompter) Note(format string, args ...any) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

// Text asks a free-text question. Empty input (or EOF) returns def.
func (p *Prompter) Text(label, def string) string {
	if def != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	line := p.readLine()
	if line == "" {
		return def
	}
	return line
}
