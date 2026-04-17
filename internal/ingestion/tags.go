package ingestion

import (
	"strings"
)

// ParseTagsFromBody extracts a ## Tags section from LLM-generated body,
// returns parsed tags and the body with the section removed.
func ParseTagsFromBody(body string) ([]string, string) {
	const header = "## Tags"
	idx := strings.Index(body, header)
	if idx == -1 {
		return nil, body
	}

	// Find the end of the Tags section (next ## heading or end of string)
	rest := body[idx+len(header):]
	endIdx := strings.Index(rest, "\n## ")
	var tagText string
	var cleanBody string
	if endIdx == -1 {
		tagText = rest
		cleanBody = strings.TrimRight(body[:idx], "\n")
	} else {
		tagText = rest[:endIdx]
		cleanBody = strings.TrimRight(body[:idx], "\n") + rest[endIdx:]
	}

	return parseTags(tagText), cleanBody
}

func parseTags(text string) []string {
	// Handle comma-separated, newline-separated, or bullet-list formats
	text = strings.ReplaceAll(text, "\n", ",")
	text = strings.ReplaceAll(text, "- ", "")
	text = strings.ReplaceAll(text, "* ", "")

	var tags []string
	for _, raw := range strings.Split(text, ",") {
		tag := strings.TrimSpace(strings.ToLower(raw))
		tag = strings.Trim(tag, "`\"'")
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
