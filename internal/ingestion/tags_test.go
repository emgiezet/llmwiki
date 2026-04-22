package ingestion_test

import (
	"testing"

	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/stretchr/testify/assert"
)

func TestParseTagsFromBody_CommaSeparated(t *testing.T) {
	body := "## Domain\nStuff.\n\n## Tags\ngo, grpc, kubernetes, event-driven\n"
	tags, clean := ingestion.ParseTagsFromBody(body)
	assert.Equal(t, []string{"go", "grpc", "kubernetes", "event-driven"}, tags)
	assert.Contains(t, clean, "## Domain")
	assert.NotContains(t, clean, "## Tags")
}

func TestParseTagsFromBody_WithBullets(t *testing.T) {
	body := "## Notes\nNone.\n\n## Tags\n- go\n- grpc\n- rabbitmq\n"
	tags, clean := ingestion.ParseTagsFromBody(body)
	assert.Equal(t, []string{"go", "grpc", "rabbitmq"}, tags)
	assert.NotContains(t, clean, "## Tags")
}

func TestParseTagsFromBody_NoSection(t *testing.T) {
	body := "## Domain\nStuff.\n\n## Notes\nNone.\n"
	tags, clean := ingestion.ParseTagsFromBody(body)
	assert.Nil(t, tags)
	assert.Equal(t, body, clean)
}

func TestParseTagsFromBody_EmptySection(t *testing.T) {
	body := "## Domain\nStuff.\n\n## Tags\n\n"
	tags, _ := ingestion.ParseTagsFromBody(body)
	assert.Empty(t, tags)
}

func TestParseTagsFromBody_TagsInMiddle(t *testing.T) {
	body := "## Domain\nStuff.\n\n## Tags\ngo, rest\n\n## Notes\nGotchas.\n"
	tags, clean := ingestion.ParseTagsFromBody(body)
	assert.Equal(t, []string{"go", "rest"}, tags)
	assert.Contains(t, clean, "## Domain")
	assert.Contains(t, clean, "## Notes")
	assert.NotContains(t, clean, "## Tags")
}
