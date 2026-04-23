package config_test

import (
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStatus(t *testing.T) {
	cases := []struct {
		in      config.ProjectStatus
		wantErr bool
	}{
		{"", false},
		{"production", false},
		{"poc", false},
		{"discovery", false},
		{"prod", true},      // typo
		{"archived", true},  // not in the set
		{"PRODUCTION", true}, // case-sensitive
	}
	for _, c := range cases {
		t.Run(string(c.in), func(t *testing.T) {
			err := config.ValidateStatus(c.in)
			if c.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid status")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCost_AllowsZerosAndPositives(t *testing.T) {
	cases := []config.CostConfig{
		{},                                                  // all zero = "not set"
		{InfraMonthlyUSD: 100, TeamFTE: 1.5},                 // positives
		{TeamFTERateMonthlyUSD: 15000, Notes: "fully loaded"}, // mixed
	}
	for _, c := range cases {
		require.NoError(t, config.ValidateCost(c))
	}
}

func TestValidateCost_RejectsNegatives(t *testing.T) {
	for _, c := range []config.CostConfig{
		{InfraMonthlyUSD: -1},
		{TeamFTE: -0.5},
		{TeamFTERateMonthlyUSD: -100},
	} {
		err := config.ValidateCost(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be ≥ 0")
	}
}

func TestValidateLinks_WarnsOnBadURLs(t *testing.T) {
	warnings := config.ValidateLinks(config.LinksConfig{
		"github":     "https://github.com/acme",            // OK
		"jira":       "",                                    // empty
		"confluence": "not-a-url",                           // not absolute
		"clickup":    "ftp://file.example.com/x",            // absolute, any scheme allowed
	})
	// We expect 2 warnings (empty + not-absolute).
	assert.Len(t, warnings, 2)

	// All-valid case produces no warnings.
	warnings2 := config.ValidateLinks(config.LinksConfig{
		"github": "https://github.com/acme",
		"slack":  "slack://channel/general",
	})
	assert.Empty(t, warnings2)
}

func TestMerge_Status_ProjectOverridesClient(t *testing.T) {
	c := config.ClientConfig{Status: config.StatusProduction}
	p := config.ProjectConfig{Status: config.StatusDiscovery}
	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, config.StatusDiscovery, got.Status)
}

func TestMerge_Status_ClientFillsEmptyProject(t *testing.T) {
	c := config.ClientConfig{Status: config.StatusProduction}
	got := config.Merge(config.GlobalConfig{}, c, config.ProjectConfig{})
	assert.Equal(t, config.StatusProduction, got.Status)
}

func TestMerge_Links_KeyByKeyOverride(t *testing.T) {
	c := config.ClientConfig{Links: config.LinksConfig{
		"github":     "https://github.com/acme",
		"confluence": "https://acme.atlassian.net/wiki/space/ACME",
	}}
	p := config.ProjectConfig{Links: config.LinksConfig{
		"jira":       "https://acme.atlassian.net/jira/projects/BILL",
		"confluence": "https://acme.atlassian.net/wiki/space/ACME/page/BILL", // overrides
	}}

	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, "https://github.com/acme", got.Links["github"], "client's github inherited")
	assert.Equal(t, "https://acme.atlassian.net/jira/projects/BILL", got.Links["jira"], "project's jira added")
	assert.Equal(t, "https://acme.atlassian.net/wiki/space/ACME/page/BILL", got.Links["confluence"], "project's confluence overrides client's")

	// Source tracking: github inherited; confluence NOT inherited (project overrode); jira NOT inherited (project-only).
	assert.True(t, got.Source.LinksFromClient["github"])
	assert.False(t, got.Source.LinksFromClient["confluence"])
	assert.False(t, got.Source.LinksFromClient["jira"])
}

func TestMerge_Links_NilWhenBothEmpty(t *testing.T) {
	got := config.Merge(config.GlobalConfig{}, config.ClientConfig{}, config.ProjectConfig{})
	assert.Nil(t, got.Links)
	assert.Nil(t, got.Source.LinksFromClient)
}

func TestMerge_Team_FieldByFieldOverride(t *testing.T) {
	c := config.ClientConfig{Team: config.TeamConfig{
		Lead:          "jane@acme.com",
		OncallChannel: "#acme-ops",
		Escalation:    "ops-mgr@acme.com",
	}}
	p := config.ProjectConfig{Team: config.TeamConfig{
		OncallChannel: "#bill-oncall",
	}}

	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, "jane@acme.com", got.Team.Lead, "inherited from client")
	assert.Equal(t, "#bill-oncall", got.Team.OncallChannel, "project override")
	assert.Equal(t, "ops-mgr@acme.com", got.Team.Escalation, "inherited from client")

	assert.True(t, got.Source.TeamLeadFromClient, "lead flagged as inherited")
	assert.False(t, got.Source.TeamOncallFromClient, "oncall was overridden, not inherited")
	assert.True(t, got.Source.TeamEscFromClient, "escalation flagged as inherited")
}

func TestMerge_Cost_FieldByFieldWithClientFlag(t *testing.T) {
	c := config.ClientConfig{Cost: config.CostConfig{
		TeamFTERateMonthlyUSD: 18000,
		Notes:                 "fully loaded rate",
	}}
	p := config.ProjectConfig{Cost: config.CostConfig{
		InfraMonthlyUSD: 1200,
		TeamFTE:         2.5,
	}}

	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, 1200.0, got.Cost.InfraMonthlyUSD)
	assert.Equal(t, 2.5, got.Cost.TeamFTE)
	assert.Equal(t, 18000.0, got.Cost.TeamFTERateMonthlyUSD, "inherited from client")
	assert.Equal(t, "fully loaded rate", got.Cost.Notes, "inherited from client")
	assert.True(t, got.Source.CostFromClient, "flag set because rate + notes came from client")
}

func TestMerge_Cost_NoInheritanceFlagWhenProjectHasEverything(t *testing.T) {
	c := config.ClientConfig{Cost: config.CostConfig{TeamFTERateMonthlyUSD: 18000}}
	p := config.ProjectConfig{Cost: config.CostConfig{
		InfraMonthlyUSD:       1200,
		TeamFTE:               2.5,
		TeamFTERateMonthlyUSD: 20000, // project overrides client's rate
	}}

	got := config.Merge(config.GlobalConfig{}, c, p)
	assert.Equal(t, 20000.0, got.Cost.TeamFTERateMonthlyUSD)
	assert.False(t, got.Source.CostFromClient, "no fields inherited — all came from project")
}
