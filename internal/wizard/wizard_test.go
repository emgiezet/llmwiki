package wizard

import (
	"bytes"
	"errors"
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

func TestConfirm_Yes(t *testing.T) {
	p, _ := newTestPrompter("y\n")
	if !p.Confirm("OK?", false) {
		t.Error("expected true for 'y'")
	}
}

func TestConfirm_No(t *testing.T) {
	p, _ := newTestPrompter("n\n")
	if p.Confirm("OK?", true) {
		t.Error("expected false for 'n'")
	}
}

func TestConfirm_DefaultOnEmpty(t *testing.T) {
	p, out := newTestPrompter("\n")
	if !p.Confirm("OK?", true) {
		t.Error("expected default true on empty")
	}
	if !strings.Contains(out.String(), "[Y/n]") {
		t.Errorf("default hint wrong; got %q", out.String())
	}
}

func TestConfirm_InvalidThenValid(t *testing.T) {
	p, out := newTestPrompter("maybe\nyes\n")
	if !p.Confirm("OK?", false) {
		t.Error("expected true after re-prompt")
	}
	if !strings.Contains(out.String(), "! please answer y or n") {
		t.Errorf("missing re-prompt message; got %q", out.String())
	}
}

func TestTextValidated_RepromptsOnError(t *testing.T) {
	p, out := newTestPrompter("bad name\ngoodname\n")
	validate := func(s string) error {
		if strings.Contains(s, " ") {
			return errors.New("no spaces allowed")
		}
		return nil
	}
	if got := p.TextValidated("Name", "", validate); got != "goodname" {
		t.Errorf("got %q, want %q", got, "goodname")
	}
	if !strings.Contains(out.String(), "! no spaces allowed") {
		t.Errorf("missing validation error; got %q", out.String())
	}
}

func TestTextValidated_StopsOnEOF(t *testing.T) {
	p, _ := newTestPrompter("bad\n")
	validate := func(s string) error { return errors.New("always invalid") }
	got := p.TextValidated("Name", "", validate)
	if got != "bad" {
		t.Errorf("got %q, want last input %q on EOF", got, "bad")
	}
}

func TestTextValidated_DefaultOnImmediateEOF(t *testing.T) {
	p, _ := newTestPrompter("") // no input at all
	got := p.TextValidated("Customer", "acme", func(string) error { return nil })
	if got != "acme" {
		t.Errorf("got %q, want default %q on immediate EOF", got, "acme")
	}
}
