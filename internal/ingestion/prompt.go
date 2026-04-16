package ingestion

import "fmt"

// BuildProjectPrompt builds the LLM prompt for generating/updating a project _index.md.
func BuildProjectPrompt(projectName, scanSummary, existingWiki string) string {
	updateNote := ""
	if existingWiki != "" {
		updateNote = fmt.Sprintf(`
EXISTING WIKI ENTRY (update this — preserve accurate information, correct outdated sections):
%s

`, existingWiki)
	}

	return fmt.Sprintf(`You are maintaining a technical wiki for a software project.
Project name: %s

PROJECT SCAN (files collected from the project directory):
%s

%sGenerate a wiki entry using EXACTLY these markdown sections. Be concise and technical. Do not add extra sections.

## Domain
(What this project is about: business domain, purpose, main users)

## Services
(List each service/component with a one-line description. Format: "- service-name: description")

## Flows
(Key end-to-end workflows between services. Use arrows: "A → B → C")

## Integrations
(External systems: databases, queues, APIs, third-party services)

## Tech Stack
(Languages, frameworks, infrastructure, deployment)

## Notes
(Architectural decisions, gotchas, known issues, anything a new engineer needs to know)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		projectName, scanSummary, updateNote)
}

// BuildServicePrompt builds the LLM prompt for generating/updating a per-service wiki file.
func BuildServicePrompt(serviceName, projectName, scanSummary, existingWiki string) string {
	updateNote := ""
	if existingWiki != "" {
		updateNote = fmt.Sprintf(`
EXISTING WIKI ENTRY (update this — preserve accurate information, correct outdated sections):
%s

`, existingWiki)
	}

	return fmt.Sprintf(`You are maintaining a technical wiki for a microservice.
Service: %s (part of project: %s)

SERVICE SCAN (files from the service directory):
%s

%sGenerate a wiki entry using EXACTLY these markdown sections. Be concise and technical.

## Purpose
(What this service does and why it exists)

## API Surface
(Endpoints, proto definitions, gRPC methods, contracts — include paths and methods where visible)

## Integrations
(What this service calls downstream, what calls it upstream)

## Notes
(Gotchas, decisions, known issues, deployment notes)

Output ONLY the markdown sections above. No preamble, no explanation.`,
		serviceName, projectName, scanSummary, updateNote)
}
