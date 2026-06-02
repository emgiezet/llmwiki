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

func TestScrubLLMResponse_DeduplicatesL2Sections(t *testing.T) {
	body := "## Domain\nFirst domain content.\n\n## Purpose\nPurpose content.\n\n## Domain\nDuplicate domain — should be dropped.\n\n## API Surface\nAPI content.\n\n## API Surface\nDuplicate API — should be dropped.\n"
	got := scrubLLMResponse(body)

	if !strings.Contains(got, "First domain content.") {
		t.Error("expected first Domain section to be preserved")
	}
	if !strings.Contains(got, "Purpose content.") {
		t.Error("expected Purpose section to be preserved")
	}
	if !strings.Contains(got, "API content.") {
		t.Error("expected first API Surface section to be preserved")
	}
	if strings.Contains(got, "Duplicate domain") {
		t.Error("expected duplicate Domain section to be removed")
	}
	if strings.Contains(got, "Duplicate API") {
		t.Error("expected duplicate API Surface section to be removed")
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
