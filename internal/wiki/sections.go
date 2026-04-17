package wiki

import "strings"

// ExtractSection returns the content of a single named section from a markdown body.
// Returns empty string if the section is not found. The heading itself is not included.
func ExtractSection(body, heading string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inSection := false
	for _, line := range lines {
		isHeader := strings.HasPrefix(line, "## ")
		if isHeader {
			if strings.HasPrefix(line, heading) {
				inSection = true
				continue // skip the heading line itself
			}
			inSection = false
		}
		if inSection {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// ExtractSections returns the content of multiple sections concatenated, including their headings.
func ExtractSections(body string, sections []string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inKept := false
	for _, line := range lines {
		isHeader := strings.HasPrefix(line, "## ")
		if isHeader {
			inKept = false
			for _, s := range sections {
				if strings.HasPrefix(line, s) {
					inKept = true
					break
				}
			}
		}
		if inKept {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// TruncateSection truncates text to maxChars, adding [truncated] if needed.
func TruncateSection(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "\n[truncated]"
}
