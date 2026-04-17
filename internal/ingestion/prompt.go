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
(All exposed interfaces. For HTTP: list routes with methods and purpose. For gRPC: list services and key RPCs. For message consumers: list queue/topic names and message types. Include request/response shapes where visible from proto definitions or swagger specs.)

## Data Model
(Key domain entities and their relationships. Database tables or collections if visible from migrations or ORM definitions. Storage technology and access patterns.)

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
