package wizard

import (
	"bytes"
	"strings"
	"testing"
)

func newTestPrompter(input string) (*Prompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return New(strings.NewReader(input), out), out
}

func TestText_ReturnsInput(t *testing.T) {
	p, out := newTestPrompter("hello\n")
	got := p.Text("Name", "def")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if !strings.Contains(out.String(), "Name [def]:") {
		t.Errorf("prompt not rendered; got %q", out.String())
	}
}

func TestText_DefaultOnEmpty(t *testing.T) {
	p, _ := newTestPrompter("\n")
	if got := p.Text("Name", "def"); got != "def" {
		t.Errorf("got %q, want default %q", got, "def")
	}
}

func TestText_DefaultOnEOF(t *testing.T) {
	p, _ := newTestPrompter("")
	if got := p.Text("Name", "def"); got != "def" {
		t.Errorf("got %q, want default %q", got, "def")
	}
}

func TestText_NoDefaultLabel(t *testing.T) {
	p, out := newTestPrompter("x\n")
	p.Text("Name", "")
	if strings.Contains(out.String(), "[]") {
		t.Errorf("empty default should not render brackets; got %q", out.String())
	}
}

func TestNote_PrintsLine(t *testing.T) {
	p, out := newTestPrompter("")
	p.Note("hello %s", "world")
	if out.String() != "hello world\n" {
		t.Errorf("got %q", out.String())
	}
}
