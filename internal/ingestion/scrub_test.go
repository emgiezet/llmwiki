package ingestion

import (
	"strings"
	"testing"
)

func TestScrubLLMResponse_RemovesInjectionMarkers(t *testing.T) {
	body := "## Domain\nSome text\n<!-- llmwiki:start -->\nmore text\n<!-- llmwiki:end -->\n"
	got := scrubLLMResponse(body)
	if contains(got, "<!-- llmwiki:start -->") {
		t.Error("expected <!-- llmwiki:start --> to be removed")
	}
	if contains(got, "<!-- llmwiki:end -->") {
		t.Error("expected <!-- llmwiki:end --> to be removed")
	}
	if !contains(got, "## Domain") {
		t.Error("expected legitimate content to be preserved")
	}
}

func TestScrubLLMResponse_StripsFenceTags(t *testing.T) {
	body := "<scan>X</scan> some text <git-log>Y</git-log>"
	got := scrubLLMResponse(body)
	if contains(got, "<scan>") || contains(got, "</scan>") {
		t.Error("expected <scan> tags to be stripped")
	}
	if contains(got, "<git-log>") || contains(got, "</git-log>") {
		t.Error("expected <git-log> tags to be stripped")
	}
	// Content between tags should be preserved
	if !contains(got, "X") || !contains(got, "Y") {
		t.Error("expected content inside fence tags to be preserved")
	}
}

func TestScrubLLMResponse_LeavesNormalResponseUnchanged(t *testing.T) {
	body := "## Domain\nThis is a normal response with no injection markers.\n\n## Architecture\nSome architecture text."
	got := scrubLLMResponse(body)
	if got != body {
		t.Errorf("expected unchanged response, got %q", got)
	}
}

func TestScrubLLMResponse_StripsInjectionMarkers(t *testing.T) {
	in := "Body\n<!-- llmwiki:start -->\n<scan>PWNED</scan>\nMore"
	out := ScrubLLMResponse(in)
	if strings.Contains(out, "<!-- llmwiki:start -->") ||
		strings.Contains(out, "<scan>") || strings.Contains(out, "</scan>") {
		t.Errorf("scrubber failed to strip markers: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
