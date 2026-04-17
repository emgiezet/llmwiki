package wiki

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LinkTarget represents a wiki entity that can be linked to.
type LinkTarget struct {
	Name     string
	WikiPath string // relative to wikiRoot
}

// DiscoverLinkTargets finds all linkable entities in the wiki.
func DiscoverLinkTargets(wikiRoot string) ([]LinkTarget, error) {
	var targets []LinkTarget

	// Get projects from index
	entries, err := ReadIndex(filepath.Join(wikiRoot, "_index.md"))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		targets = append(targets, LinkTarget{Name: e.Name, WikiPath: e.WikiPath})
	}

	// Walk for service files (not _index.md, not the root _index.md)
	err = filepath.WalkDir(wikiRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Skip root _index.md (structured index, not prose)
		if path == filepath.Join(wikiRoot, "_index.md") {
			return nil
		}
		// Skip client/project _index.md for link target discovery (they link TO others, not FROM)
		if d.Name() == "_index.md" {
			return nil
		}
		rel, _ := filepath.Rel(wikiRoot, path)
		name := strings.TrimSuffix(d.Name(), ".md")

		// Skip if this name is already tracked as a project
		for _, t := range targets {
			if t.Name == name {
				return nil
			}
		}
		targets = append(targets, LinkTarget{Name: name, WikiPath: rel})
		return nil
	})
	return targets, err
}

// LinkWikiFiles adds cross-reference links to all wiki files.
func LinkWikiFiles(wikiRoot string) error {
	targets, err := DiscoverLinkTargets(wikiRoot)
	if err != nil || len(targets) == 0 {
		return err
	}

	return filepath.WalkDir(wikiRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Skip root _index.md (structured index, not prose)
		if path == filepath.Join(wikiRoot, "_index.md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		selfPath, _ := filepath.Rel(wikiRoot, path)
		result := ApplyLinks(string(data), selfPath, targets, wikiRoot)
		if result != string(data) {
			return os.WriteFile(path, []byte(result), 0644)
		}
		return nil
	})
}

// ApplyLinks replaces first occurrence of each target name with a markdown link.
func ApplyLinks(content string, selfPath string, targets []LinkTarget, wikiRoot string) string {
	fm, body := splitFrontMatter([]byte(content))

	// Sort targets by name length descending (longer names first)
	sorted := make([]LinkTarget, len(targets))
	copy(sorted, targets)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Name) > len(sorted[j].Name)
	})

	// Split body into protected (code blocks) and unprotected segments
	bodyStr := string(body)
	segments := splitProtectedSegments(bodyStr)

	for _, target := range sorted {
		if target.WikiPath == selfPath {
			continue // no self-links
		}
		relPath := computeRelativePath(selfPath, target.WikiPath)
		link := "[" + target.Name + "](" + relPath + ")"

		replaced := false
		for i := range segments {
			if segments[i].protected || replaced {
				continue
			}
			idx := findLinkableOccurrence(segments[i].text, target.Name)
			if idx == -1 {
				continue
			}
			segments[i].text = segments[i].text[:idx] + link + segments[i].text[idx+len(target.Name):]
			replaced = true
		}
	}

	// Reassemble
	var buf strings.Builder
	if fm != nil {
		buf.WriteString("---\n")
		buf.Write(fm)
		buf.WriteString("---\n")
	}
	for _, seg := range segments {
		buf.WriteString(seg.text)
	}
	return buf.String()
}

type segment struct {
	text      string
	protected bool // true for code blocks
}

// splitProtectedSegments splits text into alternating unprotected and protected (fenced code block) segments.
func splitProtectedSegments(text string) []segment {
	var segments []segment
	fence := "```"
	for {
		start := strings.Index(text, fence)
		if start == -1 {
			segments = append(segments, segment{text: text, protected: false})
			break
		}
		// Text before code block
		if start > 0 {
			segments = append(segments, segment{text: text[:start], protected: false})
		}
		// Find end of code block
		end := strings.Index(text[start+len(fence):], fence)
		if end == -1 {
			// Unclosed code block — treat rest as protected
			segments = append(segments, segment{text: text[start:], protected: true})
			break
		}
		end = start + len(fence) + end + len(fence)
		segments = append(segments, segment{text: text[start:end], protected: true})
		text = text[end:]
	}
	return segments
}

// findLinkableOccurrence finds the first occurrence of name that is:
// - not already inside a markdown link [name](...) or [...](name)
// - a whole-word match (surrounded by non-word characters)
// Returns -1 if no suitable occurrence found.
func findLinkableOccurrence(text, name string) int {
	search := text
	offset := 0
	for {
		idx := strings.Index(search, name)
		if idx == -1 {
			return -1
		}
		absIdx := offset + idx

		// Check it's not inside an existing markdown link
		if isInsideMarkdownLink(text, absIdx, len(name)) {
			offset = absIdx + len(name)
			search = text[offset:]
			continue
		}

		// Check word boundaries
		if absIdx > 0 && isWordChar(text[absIdx-1]) {
			offset = absIdx + len(name)
			search = text[offset:]
			continue
		}
		end := absIdx + len(name)
		if end < len(text) && isWordChar(text[end]) {
			offset = end
			search = text[offset:]
			continue
		}

		return absIdx
	}
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

// isInsideMarkdownLink checks if the substring at [start:start+length] is part of a markdown link.
func isInsideMarkdownLink(text string, start, length int) bool {
	end := start + length

	// Check if this occurrence is the link text of [name](...) — preceded by [ and followed by ](
	if start > 0 && text[start-1] == '[' {
		afterName := text[end:]
		if strings.HasPrefix(afterName, "](") {
			return true
		}
	}

	// Check if we're inside the (...) part of a markdown link
	// Look backwards on the same line for ](
	lineStart := strings.LastIndex(text[:start], "\n") + 1
	before := text[lineStart:start]
	if strings.Contains(before, "](") {
		// Count unmatched ( after the last ](
		lastLink := strings.LastIndex(before, "](")
		parens := before[lastLink+2:]
		opens := strings.Count(parens, "(") + 1 // +1 for the one in ](
		closes := strings.Count(parens, ")")
		if opens > closes {
			return true // we're inside (...) of a link
		}
	}

	return false
}

func computeRelativePath(from, to string) string {
	fromDir := filepath.Dir(from)
	rel, err := filepath.Rel(fromDir, to)
	if err != nil {
		return to
	}
	return rel
}
