package config

import (
	"fmt"
	"net/url"
)

// ValidateMemoryMode returns an error if mode is not one of the known values.
// Empty string is allowed; it resolves to MemoryModeProject in Merge.
func ValidateMemoryMode(mode string) error {
	switch mode {
	case "", MemoryModeProject, MemoryModeGlobal:
		return nil
	}
	return fmt.Errorf("invalid memory_mode %q — must be one of: project, global", mode)
}

// ValidStatuses enumerates the allowed ProjectStatus values. Empty is
// also allowed — it's interpreted as "production" at the consumer.
var ValidStatuses = []ProjectStatus{
	StatusProduction,
	StatusPOC,
	StatusDiscovery,
}

// ValidateStatus returns an error when s is not one of ValidStatuses (or
// empty). Called from the YAML loaders before merge so a typoed status
// fails fast with a clear message.
func ValidateStatus(s ProjectStatus) error {
	if s == "" {
		return nil
	}
	for _, v := range ValidStatuses {
		if s == v {
			return nil
		}
	}
	return fmt.Errorf("invalid status %q — must be one of: production, poc, discovery", s)
}

// ValidateCost enforces non-negative numeric fields. Zero means "not
// set" (the wiki falls back to the how-to-estimate framework) so only
// negative values are errors.
func ValidateCost(c CostConfig) error {
	if c.InfraMonthlyUSD < 0 {
		return fmt.Errorf("cost.infra_monthly_usd must be ≥ 0, got %v", c.InfraMonthlyUSD)
	}
	if c.TeamFTE < 0 {
		return fmt.Errorf("cost.team_fte must be ≥ 0, got %v", c.TeamFTE)
	}
	if c.TeamFTERateMonthlyUSD < 0 {
		return fmt.Errorf("cost.team_fte_rate_usd_monthly must be ≥ 0, got %v", c.TeamFTERateMonthlyUSD)
	}
	return nil
}

// ValidateLinks soft-checks every URL in the map; returns a slice of
// warnings (as strings) rather than an error because a malformed link
// shouldn't block an ingest — it just gets skipped in rendering. Callers
// print the warnings to stderr.
func ValidateLinks(l LinksConfig) []string {
	var warnings []string
	for k, v := range l {
		if v == "" {
			warnings = append(warnings, fmt.Sprintf("links.%s: empty URL — will be skipped in rendering", k))
			continue
		}
		u, err := url.Parse(v)
		if err != nil || !u.IsAbs() {
			warnings = append(warnings, fmt.Sprintf("links.%s = %q: not a valid absolute URL — will be skipped in rendering", k, v))
		}
	}
	return warnings
}
