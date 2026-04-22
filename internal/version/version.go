// Package version exposes build-time metadata. The three vars are set via
// `go build -ldflags "-X ..."` during release builds (see .goreleaser.yaml).
// Unreleased dev builds keep the zero-values, which update.Check treats as
// "do not check for updates" — important so dev workflows don't get update
// prompts.
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
