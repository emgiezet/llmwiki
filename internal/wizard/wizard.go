// Package wizard provides dependency-free interactive prompt helpers for the
// llmwiki CLI. All I/O goes through an injected reader/writer so callers can
// be tested without a terminal.
package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
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

// Choice prints a numbered list of opts and returns the chosen option's Value.
// The option whose Value == def is marked and selected on empty input. Invalid
// numbers re-prompt; on EOF the default is returned to avoid an infinite loop.
func (p *Prompter) Choice(label string, opts []Option, def string) string {
	for {
		fmt.Fprintln(p.out, label)
		defIdx := 0
		for i, o := range opts {
			marker := " "
			if o.Value == def {
				marker = "*"
				defIdx = i + 1
			}
			fmt.Fprintf(p.out, "  %s %d) %s\n", marker, i+1, o.Label)
		}
		if defIdx > 0 {
			fmt.Fprintf(p.out, "Choice [%d]: ", defIdx)
		} else {
			fmt.Fprint(p.out, "Choice: ")
		}

		line := p.readLine()
		if line == "" {
			return def
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(opts) {
			fmt.Fprintf(p.out, "  ! invalid choice %q\n", line)
			if p.eof {
				return def
			}
			continue
		}
		return opts[n-1].Value
	}
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
