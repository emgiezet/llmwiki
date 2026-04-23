package ingestion

import (
	"fmt"
	"strings"

	"github.com/emgiezet/llmwiki/internal/config"
)

// Scope encodes whether a section applies to project prompts, service prompts,
// or both. It's a bitfield so that Section.Scope can express "Both" as the union.
type Scope int

const (
	ScopeProject Scope = 1 << iota
	ScopeService
)

// ScopeBoth is a section that appears in both project and service prompts.
const ScopeBoth = ScopeProject | ScopeService

// Section describes one markdown section the LLM may be asked to produce.
type Section struct {
	ID          string // stable key used in llmwiki.yaml / CLI flags
	Title       string // markdown heading text (without leading "## ")
	Scope       Scope  // which prompt types this section applies to
	Category    string // "core" | "software" | "feature" — used to group presets
	Instruction string // guidance text emitted inside parentheses after the heading
}

// AllSections is the registry of every section the pipeline knows how to
// request. Order matters: this is the order sections are emitted into the
// prompt when a preset or explicit list resolves to multiple sections.
//
// Instruction text for sections in the today's hard-coded prompt is copied
// verbatim so that preset="default" produces semantically identical output.
var AllSections = []Section{
	{
		ID: "domain", Title: "Domain", Scope: ScopeBoth, Category: "core",
		Instruction: "Business context: what this project does, what problems it solves, who uses it, what business domain it operates in. Explain enough that a new engineer understands why this project exists.",
	},
	{
		ID: "purpose", Title: "Purpose", Scope: ScopeService, Category: "core",
		Instruction: "What this service does, why it exists, what problem it solves, its role within the broader project. Enough context that a new engineer understands the service's reason for being.",
	},
	{
		ID: "architecture", Title: "Architecture", Scope: ScopeBoth, Category: "software",
		Instruction: "How the system is structured: key design patterns, module organization, runtime topology, build pipeline. If it is a monolith vs. microservices, describe the decomposition rationale. Mention significant architectural decisions visible from the code structure.",
	},
	{
		ID: "patterns", Title: "Patterns", Scope: ScopeBoth, Category: "software",
		Instruction: "Recurring design and implementation patterns visible in the codebase (e.g. repository pattern, hexagonal architecture, CQRS, event sourcing, dependency injection style, error-handling conventions, concurrency primitives). For each pattern: name it, cite where it is used, and explain the trade-off it represents.",
	},
	{
		ID: "good_practices", Title: "Good Practices", Scope: ScopeBoth, Category: "software",
		Instruction: "Engineering practices the project appears to follow: code style enforcement, pre-commit or CI gates, test coverage expectations, code review conventions, secret-handling discipline, observability standards, documentation conventions. Base this on concrete evidence in the scan (config files, CI pipelines, lint rules) — do not invent generic best-practice advice.",
	},
	{
		ID: "testing", Title: "Testing Principles", Scope: ScopeBoth, Category: "software",
		Instruction: "Testing strategy actually in use: unit vs. integration vs. end-to-end split, frameworks, fixtures/golden files, mocking approach, CI integration, flakiness mitigation, coverage targets. Cite concrete test files or CI configuration. If there are gaps (e.g. no integration tests visible), call them out.",
	},
	{
		ID: "runbook", Title: "Run & Setup", Scope: ScopeProject, Category: "software",
		Instruction: "Concrete commands to clone, install dependencies, configure local env, build, run, and tear down. Include prerequisite versions (language, runtime, tooling), any required services (DB, queue), and the exact developer loop. If the scan shows a Makefile, docker-compose, or similar, extract the real commands — do not paraphrase.",
	},
	{
		ID: "tech_debt", Title: "Technical Debt", Scope: ScopeBoth, Category: "software",
		Instruction: "Concrete pain points and debt visible in the code: TODO/FIXME/HACK comments, deprecated dependencies, duplicated logic, brittle workarounds, known risky areas, migrations in flight. For each item: where it lives, why it is debt, and the consequence of leaving it. Ground every item in the scan — no generic \"tech debt exists\" filler.",
	},
	{
		ID: "services", Title: "Services", Scope: ScopeProject, Category: "core",
		Instruction: "List each service or major component. For each, give: name, one-line purpose, language/framework, key responsibilities. Format: \"- **service-name** — description\"",
	},
	{
		ID: "features", Title: "Features", Scope: ScopeBoth, Category: "feature",
		Instruction: "Key capabilities and user-facing functionality. What can users or other systems do with this project? Group by functional area if applicable.",
	},
	{
		ID: "roadmap", Title: "Roadmap", Scope: ScopeProject, Category: "feature",
		Instruction: "Planned or in-flight work visible from the scan: open TODOs describing upcoming features, RFC/proposal documents, deprecated-but-not-removed code paths, feature flags suggesting staged rollout, comments referencing future phases. For each roadmap item: what is intended, evidence in the scan, and status (planned / in progress / blocked). If nothing is visible, say \"No explicit roadmap signals detected in scan data.\"",
	},
	{
		ID: "flows", Title: "Flows", Scope: ScopeBoth, Category: "core",
		Instruction: "Key end-to-end workflows. For each flow: describe the trigger, the path through services/components, data transformations, and terminal state. Use arrows: \"A → B → C\". Include async flows and error-handling paths where visible.",
	},
	{
		ID: "system_diagram", Title: "System Diagram", Scope: ScopeBoth, Category: "core",
		Instruction: "Mermaid flowchart showing all services/components and external integrations. Output a mermaid code block using flowchart TD or LR. Label edges with protocols. Include databases, queues, and external APIs as nodes. Do not use subgraphs unless there are clear bounded contexts.",
	},
	{
		ID: "data_model", Title: "Data Model", Scope: ScopeService, Category: "core",
		Instruction: "Key domain entities and their relationships. Database tables or collections if visible from migrations or ORM definitions. Storage technology and access patterns.",
	},
	{
		ID: "data_model_diagram", Title: "Data Model Diagram", Scope: ScopeBoth, Category: "core",
		Instruction: "Mermaid erDiagram showing key database entities and their relationships. Output a mermaid code block. Include entity names, key attributes, and relationship cardinality. If no database schema is visible from the scan data, write \"No database schema detected in scan data.\" instead of a diagram.",
	},
	{
		ID: "api_surface", Title: "API Surface", Scope: ScopeService, Category: "core",
		Instruction: "All exposed interfaces. IMPORTANT: If swagger/openapi spec data appears in the scan above, extract EVERY endpoint — list method, path, summary, and key parameters as a markdown table: | Method | Path | Description |. For gRPC: list all services and RPCs with request/response message types. For message consumers: list queue/topic names and message types. Be exhaustive — this is the API reference.",
	},
	{
		ID: "integrations", Title: "Integrations", Scope: ScopeBoth, Category: "core",
		Instruction: "External systems this project connects to. For each: name, protocol (HTTP/gRPC/AMQP/SQL), purpose, authentication method if visible, and failure behavior if documented. Group by: databases, message queues, external APIs, observability.",
	},
	{
		ID: "tech_stack", Title: "Tech Stack", Scope: ScopeBoth, Category: "core",
		Instruction: "Languages with versions, frameworks, infrastructure components, deployment tooling, CI/CD pipeline, testing frameworks, linting/quality tools.",
	},
	{
		ID: "configuration", Title: "Configuration", Scope: ScopeBoth, Category: "core",
		Instruction: "Key environment variables and their purpose. Feature flags, runtime modes, and deployment variants. Note which configs are required vs. optional where visible.",
	},
	{
		ID: "notes", Title: "Notes", Scope: ScopeBoth, Category: "core",
		Instruction: "Architectural decisions, trade-offs, gotchas, known issues, migration history, anything a new engineer should know that does not fit above.",
	},
	{
		ID: "tags", Title: "Tags", Scope: ScopeBoth, Category: "core",
		Instruction: "Comma-separated list of technology and architectural pattern tags. Include: languages, frameworks, infrastructure, protocols, and patterns like event-driven, microservices, monolith. Example: go, gin, grpc, kubernetes, rabbitmq, event-driven. Output ONLY the comma-separated list, no bullets or explanation.",
	},

	// v1.3.0 discovery-oriented sections — used when status: discovery.
	{
		ID: "open_questions", Title: "Open Questions", Scope: ScopeProject, Category: "discovery",
		Instruction: "List the unresolved questions about scope, requirements, constraints, users, or integrations that need answers before build can start. If a PRD/requirements.md is present in the scan, extract questions flagged as TBD / TODO / unclear. One bullet per question, prefixed with a short topic tag. Err on the side of listing too many — it's better to surface ambiguity than hide it.",
	},
	{
		ID: "requirements", Title: "Requirements", Scope: ScopeProject, Category: "discovery",
		Instruction: "Structured summary of functional and non-functional requirements extracted from the PRD, requirements doc, meeting notes, or README. Split into subsections: **Functional** (what the system does), **Non-functional** (performance, security, compliance, availability), **Constraints** (budget, timeline, technology choices already made). Cite the source doc when quoting a specific requirement.",
	},
	{
		ID: "scope", Title: "Scope", Scope: ScopeProject, Category: "discovery",
		Instruction: "What is IN and what is OUT of this project's scope. Two explicit lists: **In scope** and **Out of scope**. Pull from explicit scope statements in the PRD; for unstated items, mark them as \"Assumed in scope\" or \"Assumed out of scope\" so the user can correct.",
	},
	{
		ID: "assumptions", Title: "Assumptions", Scope: ScopeProject, Category: "discovery",
		Instruction: "Things the team is taking as given without formal validation. These are risks in disguise — each assumption should be one line with a short note on what breaks if the assumption turns out to be wrong.",
	},
	{
		ID: "stakeholders", Title: "Stakeholders", Scope: ScopeProject, Category: "discovery",
		Instruction: "Who is involved and why. Include: sponsor / decision-maker, end users, upstream systems' owners, downstream consumers' owners, compliance/security contacts. For each, name + role + what they expect from this project. Extract from project docs or README.",
	},

	// v1.3.0 POC-oriented sections — used when status: poc.
	{
		ID: "scope_assumptions", Title: "Scope & Assumptions", Scope: ScopeProject, Category: "poc",
		Instruction: "Compact summary of what this POC aims to prove, what's explicitly out of scope, and the assumptions baked in. Two short subsections: **What the POC tests** (the hypothesis) and **Explicit non-goals** (things deliberately deferred).",
	},
	{
		ID: "success_criteria", Title: "Success Criteria", Scope: ScopeProject, Category: "poc",
		Instruction: "The concrete, measurable outcomes that determine whether the POC succeeded. Each criterion should be binary (met / not met) and tied to something testable. If quantitative thresholds exist in the PRD or tickets, extract them verbatim.",
	},

	// v1.3.0 production-oriented placeholder — populated in v1.4.0 by a
	// bug-summary extractor (git log + optional MCP issue tracker).
	{
		ID: "bug_summary", Title: "Recent Bugs", Scope: ScopeProject, Category: "production",
		Instruction: "Summary of the last three months of bug fixes. For each notable bug: what broke, how it surfaced, the fix, and any regression guard added. If this is an ingest-from-scratch run, note that the full history will populate once the project has been re-ingested after v1.4.0 ships the git-log extractor.",
	},
}

// sectionByID indexes AllSections for O(1) lookup.
var sectionByID = func() map[string]Section {
	m := make(map[string]Section, len(AllSections))
	for _, s := range AllSections {
		m[s.ID] = s
	}
	return m
}()

// Presets define named section bundles users can select via extraction.preset.
//
// "default" must match today's hard-coded prompts so existing users who don't
// set an extraction block see unchanged behavior.
var Presets = map[string][]string{
	"default": {
		// project scope — today's BuildProjectPrompt section order
		"domain", "architecture", "services", "features", "flows",
		"system_diagram", "data_model_diagram", "integrations",
		"tech_stack", "configuration", "notes", "tags",
		// service scope — today's BuildServicePrompt adds these; filtered by scope
		"purpose", "api_surface", "data_model",
	},
	"minimal": {"domain", "architecture", "features", "tags"},
	"software": {
		"domain", "architecture", "patterns", "good_practices",
		"testing", "runbook", "tech_debt", "tech_stack", "configuration", "tags",
	},
	"feature": {"domain", "features", "roadmap", "integrations", "notes", "tags"},
	"full":    allIDs(),
	// v1.3.0 status-driven presets. Used by ResolveSections when the project
	// has status:<x> but no explicit preset / sections override.
	"status-production": {
		"domain", "architecture", "services", "features", "flows",
		"system_diagram", "data_model_diagram", "integrations",
		"tech_stack", "configuration", "notes", "tags",
		"purpose", "api_surface", "data_model",
		"bug_summary", // v1.4.0 fills this; until then, a scaffolded section
	},
	"status-discovery": {
		"domain", "open_questions", "requirements", "scope",
		"assumptions", "stakeholders", "integrations", "notes", "tags",
	},
	"status-poc": {
		"domain", "scope_assumptions", "architecture",
		"tech_stack", "success_criteria", "notes", "tags",
	},
}

// StatusPresetName maps a ProjectStatus to the Presets key used as the
// default section bundle when no explicit preset / sections list is set.
// Returns empty string for unknown / empty status — the caller falls back
// to Presets["default"] to preserve pre-v1.3.0 behaviour.
func StatusPresetName(s config.ProjectStatus) string {
	switch s {
	case config.StatusProduction:
		return "status-production"
	case config.StatusDiscovery:
		return "status-discovery"
	case config.StatusPOC:
		return "status-poc"
	}
	return ""
}

func allIDs() []string {
	ids := make([]string, len(AllSections))
	for i, s := range AllSections {
		ids[i] = s.ID
	}
	return ids
}

// PresetNames returns sorted preset keys for error messages and CLI help.
func PresetNames() []string {
	names := make([]string, 0, len(Presets))
	for k := range Presets {
		names = append(names, k)
	}
	// not sorted — small set; callers can sort if needed
	return names
}

// ResolveSections picks the section list for a given prompt scope.
//
// Resolution rule (first match wins):
//  1. cfg.Sections non-empty → use verbatim
//  2. cfg.Preset non-empty → resolve it
//  3. status non-empty → use StatusPresetName(status) mapping
//  4. fall back to Presets["default"] (= pre-v1.3.0 behaviour)
//
// Sections whose Scope does not include the requested scope are silently
// filtered out. Returns an error for unknown section IDs or preset names
// (fail fast on typos).
func ResolveSections(cfg config.ExtractionConfig, status config.ProjectStatus, scope Scope) ([]Section, error) {
	ids, err := resolveIDs(cfg, status)
	if err != nil {
		return nil, err
	}
	result := make([]Section, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		s, ok := sectionByID[id]
		if !ok {
			return nil, fmt.Errorf("unknown section %q (known: %s)", id, strings.Join(allIDs(), ", "))
		}
		if s.Scope&scope == 0 {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, s)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no sections resolved for scope (check your extraction config)")
	}
	return result, nil
}

func resolveIDs(cfg config.ExtractionConfig, status config.ProjectStatus) ([]string, error) {
	if len(cfg.Sections) > 0 {
		return cfg.Sections, nil
	}
	preset := cfg.Preset
	if preset == "" {
		// v1.3.0: status drives the default preset when the user hasn't
		// explicitly picked one.
		preset = StatusPresetName(status)
	}
	if preset == "" {
		preset = "default"
	}
	ids, ok := Presets[preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q (known: %s)", preset, strings.Join(PresetNames(), ", "))
	}
	return ids, nil
}
