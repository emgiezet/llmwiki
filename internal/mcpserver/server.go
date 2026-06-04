// Package mcpserver exposes the extracted llmwiki knowledge base over the
// Model Context Protocol, so agents can search and fetch project information
// without invoking an LLM. It is a thin layer over wiki.Store.
package mcpserver

import (
	"context"

	"github.com/emgiezet/llmwiki/internal/version"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchInput is the argument schema for the search_projects tool. Both
// filters are optional: omit them to list every project.
type SearchInput struct {
	Client  string `json:"client,omitempty" jsonschema:"filter by client/customer name (exact, case-insensitive)"`
	Project string `json:"project,omitempty" jsonschema:"filter by project name (case-insensitive substring match)"`
}

// ProjectInfo is one project in a search result.
type ProjectInfo struct {
	Name     string   `json:"name"`
	Customer string   `json:"customer,omitempty"`
	Type     string   `json:"type"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	WikiPath string   `json:"wiki_path"`
}

// SearchOutput is the result schema for the search_projects tool.
type SearchOutput struct {
	Projects []ProjectInfo `json:"projects"`
	Count    int           `json:"count"`
}

// GetInput is the argument schema for the get_project tool.
type GetInput struct {
	Project string `json:"project" jsonschema:"project name (case-insensitive substring match; must resolve to a single project)"`
	Client  string `json:"client,omitempty" jsonschema:"client/customer name to disambiguate when several projects share a name"`
	Service string `json:"service,omitempty" jsonschema:"service name within a multi-service project"`
}

// GetOutput is the result schema for the get_project tool.
type GetOutput struct {
	Name     string `json:"name"`
	Customer string `json:"customer,omitempty"`
	Service  string `json:"service,omitempty"`
	Content  string `json:"content"`
}

// Handlers holds the tool handlers, backed by a wiki.Store.
type Handlers struct {
	store *wiki.Store
}

// NewHandlers returns Handlers backed by the given store.
func NewHandlers(store *wiki.Store) *Handlers {
	return &Handlers{store: store}
}

// Search implements the search_projects tool.
func (h *Handlers) Search(_ context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	matches, err := h.store.Search(in.Client, in.Project)
	if err != nil {
		return nil, SearchOutput{}, err
	}
	projects := make([]ProjectInfo, len(matches))
	for i, m := range matches {
		projects[i] = ProjectInfo{
			Name:     m.Name,
			Customer: m.Customer,
			Type:     m.Type,
			Status:   m.Status,
			Tags:     m.Tags,
			Summary:  m.Summary,
			WikiPath: m.WikiPath,
		}
	}
	return nil, SearchOutput{Projects: projects, Count: len(projects)}, nil
}

// Get implements the get_project tool.
func (h *Handlers) Get(_ context.Context, _ *mcp.CallToolRequest, in GetInput) (*mcp.CallToolResult, GetOutput, error) {
	content, meta, err := h.store.GetProject(in.Client, in.Project, in.Service)
	if err != nil {
		return nil, GetOutput{}, err
	}
	return nil, GetOutput{
		Name:     meta.Name,
		Customer: meta.Customer,
		Service:  in.Service,
		Content:  content,
	}, nil
}

// New builds an MCP server exposing the search_projects and get_project tools.
func New(store *wiki.Store) *mcp.Server {
	h := NewHandlers(store)
	srv := mcp.NewServer(&mcp.Implementation{Name: "llmwiki", Version: version.Version}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "search_projects",
		Description: "Search the llmwiki knowledge base for projects. Filter by client and/or project name (both optional). Returns project metadata, tags and a short domain summary for each match.",
	}, h.Search)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_project",
		Description: "Fetch the full extracted wiki content for a single project, or for a specific service within a multi-service project.",
	}, h.Get)
	return srv
}

// Serve runs the MCP server over stdio until the client disconnects or ctx is cancelled.
func Serve(ctx context.Context, store *wiki.Store) error {
	return New(store).Run(ctx, &mcp.StdioTransport{})
}
