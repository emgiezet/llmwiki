package wiki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testBody = `## Domain
Some domain text.
More domain text.

## Architecture
Architecture details here.

## Services
Service list here.

## Ignored
This section is not wanted.
`

func TestExtractSection(t *testing.T) {
	t.Run("found section", func(t *testing.T) {
		result := ExtractSection(testBody, "## Domain")
		assert.Equal(t, "Some domain text.\nMore domain text.", result)
	})

	t.Run("found section Architecture", func(t *testing.T) {
		result := ExtractSection(testBody, "## Architecture")
		assert.Equal(t, "Architecture details here.", result)
	})

	t.Run("missing section returns empty string", func(t *testing.T) {
		result := ExtractSection(testBody, "## NonExistent")
		assert.Equal(t, "", result)
	})

	t.Run("heading itself not included", func(t *testing.T) {
		result := ExtractSection(testBody, "## Services")
		assert.NotContains(t, result, "## Services")
		assert.Contains(t, result, "Service list here.")
	})

	t.Run("empty body returns empty string", func(t *testing.T) {
		result := ExtractSection("", "## Domain")
		assert.Equal(t, "", result)
	})
}

func TestExtractSections(t *testing.T) {
	t.Run("multiple sections included with headings", func(t *testing.T) {
		result := ExtractSections(testBody, []string{"## Domain", "## Architecture"})
		assert.Contains(t, result, "## Domain")
		assert.Contains(t, result, "Some domain text.")
		assert.Contains(t, result, "## Architecture")
		assert.Contains(t, result, "Architecture details here.")
		assert.NotContains(t, result, "## Ignored")
		assert.NotContains(t, result, "## Services")
	})

	t.Run("no matching sections returns empty string", func(t *testing.T) {
		result := ExtractSections(testBody, []string{"## Nope", "## Missing"})
		assert.Equal(t, "", result)
	})

	t.Run("single section", func(t *testing.T) {
		result := ExtractSections(testBody, []string{"## Services"})
		assert.Contains(t, result, "## Services")
		assert.Contains(t, result, "Service list here.")
	})

	t.Run("empty sections list returns empty string", func(t *testing.T) {
		result := ExtractSections(testBody, []string{})
		assert.Equal(t, "", result)
	})
}

func TestTruncateSection(t *testing.T) {
	t.Run("short text not truncated", func(t *testing.T) {
		result := TruncateSection("hello", 100)
		assert.Equal(t, "hello", result)
	})

	t.Run("exact length not truncated", func(t *testing.T) {
		result := TruncateSection("hello", 5)
		assert.Equal(t, "hello", result)
	})

	t.Run("long text truncated with marker", func(t *testing.T) {
		text := "abcdefghij"
		result := TruncateSection(text, 5)
		assert.Equal(t, "abcde\n[truncated]", result)
	})

	t.Run("empty text not truncated", func(t *testing.T) {
		result := TruncateSection("", 10)
		assert.Equal(t, "", result)
	})
}
