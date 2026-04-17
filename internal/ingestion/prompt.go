package ingestion

import "fmt"

// BuildProjectPrompt builds the LLM prompt for generating/updating a project _index.md.
func BuildProjectPrompt(projectName, scanSummary, existingWiki string) string {
	updateNote := ""
	if existingWiki != "" {
		updateNote = fmt.Sprintf(`
EXISTING WIKI ENTRY (update this — preserve accurate information, correct outdated sections, expand where new scan data adds detail):
%s

`, existingWiki)
	}

	return fmt.Sprintf(`You are maintaining a comprehensive technical wiki for a software project.
Project name: %s

PROJECT SCAN (files collected from the project directory):
%s

%sGenerate a wiki entry using EXACTLY these markdown sections. Be thorough and detailed — this wiki is a reference for engineers joining the project. Write in a factual, technical style. Do not speculate beyond what the scan data supports, but do explain implications and architectural significance of what you see.

## Domain
(Business context: what this project does, what problems it solves, who uses it, what business domain it operates in. Explain enough that a new engineer understands why this project exists.)

## Architecture
(How the system is structured: key design patterns, module organization, runtime topology, build pipeline. If it is a monolith vs. microservices, describe the decomposition rationale. Mention significant architectural decisions visible from the code structure.)

## Services
(List each service or major component. For each, give: name, one-line purpose, language/framework, key responsibilities. Format: "- **service-name** — description")

## Features
(Key capabilities and user-facing functionality. What can users or other systems do with this project? Group by functional area if applicable.)

## Flows
(Key end-to-end workflows. For each flow: describe the trigger, the path through services/components, data transformations, and terminal state. Use arrows: "A → B → C". Include async flows and error-handling paths where visible.)

## System Diagram
(Mermaid flowchart showing all services/components and external integrations. Output a mermaid code block using flowchart TD or LR. Label edges with protocols. Include databases, queues, and external APIs as nodes. Do not use subgraphs unless there are clear bounded contexts.)

## Data Model Diagram
(Mermaid erDiagram showing key database entities and their relationships. Output a mermaid code block. Include entity names, key attributes, and relationship cardinality. If no database schema is visible from the scan data, write "No database schema detected in scan data." instead of a diagram.)

## Integrations
(External systems this project connects to. For each: name, protocol (HTTP/gRPC/AMQP/SQL), purpose, authentication method if visible, and failure behavior if documented. Group by: databases, message queues, external APIs, observability.)

## Tech Stack
(Languages with versions, frameworks, infrastructure components, deployment tooling, CI/CD pipeline, testing frameworks, linting/quality tools.)

## Configuration
(Key environment variables and their purpose. Feature flags, runtime modes, and deployment variants. Note which configs are required vs. optional where visible.)

## Notes
(Architectural decisions, trade-offs, gotchas, known issues, migration history, anything a new engineer should know that does not fit above.)

## Tags
(Comma-separated list of technology and architectural pattern tags. Include: languages, frameworks, infrastructure, protocols, and patterns like event-driven, microservices, monolith. Example: go, gin, grpc, kubernetes, rabbitmq, event-driven. Output ONLY the comma-separated list, no bullets or explanation.)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		projectName, scanSummary, updateNote)
}

// BuildServicePrompt builds the LLM prompt for generating/updating a per-service wiki file.
func BuildServicePrompt(serviceName, projectName, scanSummary, existingWiki string) string {
	updateNote := ""
	if existingWiki != "" {
		updateNote = fmt.Sprintf(`
EXISTING WIKI ENTRY (update this — preserve accurate information, correct outdated sections, expand where new scan data adds detail):
%s

`, existingWiki)
	}

	return fmt.Sprintf(`You are maintaining a comprehensive technical wiki for a microservice.
Service: %s (part of project: %s)

SERVICE SCAN (files from the service directory):
%s

%sGenerate a wiki entry using EXACTLY these markdown sections. Be thorough and detailed — this wiki is a reference for engineers working on this service. Write in a factual, technical style. Do not speculate beyond what the scan data supports, but do explain implications and architectural significance of what you see.

## Purpose
(What this service does, why it exists, what problem it solves, its role within the broader project. Enough context that a new engineer understands the service's reason for being.)

## Architecture
(Internal structure: key packages/modules, design patterns used, runtime modes, concurrency model if visible. How the code is organized and why.)

## API Surface
(All exposed interfaces. IMPORTANT: If swagger/openapi spec data appears in the scan above, extract EVERY endpoint — list method, path, summary, and key parameters as a markdown table: | Method | Path | Description |. For gRPC: list all services and RPCs with request/response message types. For message consumers: list queue/topic names and message types. Be exhaustive — this is the API reference.)

## System Diagram
(Mermaid flowchart showing this service's connections: upstream callers, downstream dependencies, databases, queues, and external APIs. Output a mermaid code block using flowchart LR. Label edges with protocols and methods.)

## Data Model
(Key domain entities and their relationships. Database tables or collections if visible from migrations or ORM definitions. Storage technology and access patterns.)

## Data Model Diagram
(Mermaid erDiagram showing this service's database entities and their relationships. Output a mermaid code block. If no database schema is visible from the scan data, write "No database schema detected in scan data." instead of a diagram.)

## Integrations
(Upstream: what calls this service, and how. Downstream: what this service calls, protocol, purpose, failure handling. For each integration: name, protocol, direction, purpose.)

## Configuration
(Key environment variables and their purpose. Required vs. optional. Runtime modes and feature flags. Deployment variants.)

## Notes
(Gotchas, decisions, known issues, deployment specifics, migration notes, anything a maintainer should know.)

## Tags
(Comma-separated list of technology and architectural pattern tags. Include: languages, frameworks, infrastructure, protocols, and patterns like event-driven, microservices, monolith. Example: go, gin, grpc, kubernetes, rabbitmq, event-driven. Output ONLY the comma-separated list, no bullets or explanation.)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		serviceName, projectName, scanSummary, updateNote)
}

// BuildMultiProjectIndexPrompt builds the prompt for generating a multi-service project _index.md.
func BuildMultiProjectIndexPrompt(projectName string, serviceSummaries string, existingWiki string) string {
	updateNote := ""
	if existingWiki != "" {
		updateNote = fmt.Sprintf(`
EXISTING INDEX (update this — preserve accurate information, correct outdated sections):
%s

`, existingWiki)
	}

	return fmt.Sprintf(`You are generating a project-level overview for a multi-service software project.
Project name: %s

SERVICE SUMMARIES (extracted from individual service wiki files):
%s

%sGenerate a project index using EXACTLY these markdown sections. Be thorough — this is the entry point for engineers joining this project.

## Domain
(What this project does as a whole. Business context, users, and problems solved — synthesized from all service purposes.)

## Architecture
(How the services relate to each other. Decomposition rationale, communication patterns, shared infrastructure. What design decisions shaped this service topology.)

## Services
(Markdown table listing each service: | Service | Purpose | Key Tech | Wiki Link |
Use relative links to each service file, e.g., [service-name](service-name.md))

## System Diagram
(Mermaid flowchart showing ALL services and their interactions. Output a mermaid code block using flowchart LR. Show data flow between services, external systems, databases, and queues. Label edges with protocols.)

## Integrations
(Consolidated external integrations across all services. Group by: databases, message queues, external APIs, observability.)

## Tech Stack
(Union of technologies used across all services.)

## Tags
(Comma-separated consolidated tags across all services.)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		projectName, serviceSummaries, updateNote)
}

// BuildClientIndexPrompt builds the prompt for generating a per-client _index.md.
func BuildClientIndexPrompt(customer string, projectSummaries string) string {
	return fmt.Sprintf(`You are generating an executive overview for a client's entire technology portfolio.
Client: %s

PROJECT SUMMARIES (extracted from individual project wiki files):
%s

Generate a client-level index using EXACTLY these markdown sections. Be thorough — this document gives leadership and new engineers a complete picture of this client's technology landscape.

## Executive Summary
(Business overview: what this client does, what problems their systems solve, how the projects relate to each other. Written for a technical leader or architect joining the account.)

## C4 Diagram
(Mermaid C4Context diagram showing all projects as systems and their key relationships. Use this EXACT mermaid syntax:

C4Context
    title System Landscape for %s
    System(system_id, "System Name", "Description")
    System_Ext(ext_id, "External System", "Description")
    Rel(from_id, to_id, "Relationship", "Protocol")

Show each project as a System, shared infrastructure as System_Ext, and key data flows as Rel.)

## Architecture Overview
(Common architectural patterns across projects. Shared infrastructure, databases, message queues, observability stack. Technology choices and their rationale. Cross-cutting concerns like authentication, deployment, CI/CD.)

## Projects
(Markdown table: | Project | Description | Tech Stack | Wiki Link |
One row per project with a one-line summary. Use relative links to each project's wiki file.)

## Tags
(Comma-separated consolidated tags across all projects.)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		customer, projectSummaries, customer)
}
