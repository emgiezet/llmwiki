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

func TestChoice_SelectByNumber(t *testing.T) {
	p, _ := newTestPrompter("2\n")
	opts := []Option{{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"}}
	if got := p.Choice("Pick", opts, "a"); got != "b" {
		t.Errorf("got %q, want %q", got, "b")
	}
}

func TestChoice_DefaultOnEmpty(t *testing.T) {
	p, out := newTestPrompter("\n")
	opts := []Option{{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"}}
	if got := p.Choice("Pick", opts, "b"); got != "b" {
		t.Errorf("got %q, want default %q", got, "b")
	}
	if !strings.Contains(out.String(), "Choice [2]:") {
		t.Errorf("default index not shown; got %q", out.String())
	}
}

func TestChoice_InvalidThenValid(t *testing.T) {
	p, out := newTestPrompter("9\nx\n1\n")
	opts := []Option{{Value: "a", Label: "Apple"}, {Value: "b", Label: "Banana"}}
	if got := p.Choice("Pick", opts, "a"); got != "a" {
		t.Errorf("got %q, want %q", got, "a")
	}
	if strings.Count(out.String(), "! invalid choice") != 2 {
		t.Errorf("expected two error lines; got %q", out.String())
	}
}

func TestChoice_DefaultOnEOF(t *testing.T) {
	p, _ := newTestPrompter("")
	opts := []Option{{Value: "a", Label: "Apple"}}
	if got := p.Choice("Pick", opts, "a"); got != "a" {
		t.Errorf("got %q, want default %q on EOF", got, "a")
	}
}
