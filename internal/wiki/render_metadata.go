package wiki

import (
	"fmt"
	"sort"
	"strings"
)

// LinkEntry is one row to render in the Links section. FromClient triggers
// the "*(inherited from client)*" annotation.
type LinkEntry struct {
	Key        string
	URL        string
	FromClient bool
}

// TeamData mirrors the team block with per-field inheritance flags.
// Empty scalar fields are skipped in rendering.
type TeamData struct {
	Lead              string
	LeadFromClient    bool
	OncallChannel     string
	OncallFromClient  bool
	Escalation        string
	EscFromClient     bool
	Notes             string
	NotesFromClient   bool
}

// CostData mirrors the cost block. All numeric fields are optional — if
// none are set, RenderCost returns the how-to-estimate framework instead
// of a calculation.
type CostData struct {
	InfraMonthlyUSD       float64
	TeamFTE               float64
	TeamFTERateMonthlyUSD float64
	Notes                 string
	FromClient            bool // true if any field was inherited
}

// wellKnownLinkLabels maps well-known link keys to their human labels.
// Unknown keys fall back to Title-casing the key.
var wellKnownLinkLabels = map[string]string{
	"github":     "GitHub",
	"gitlab":     "GitLab",
	"bitbucket":  "Bitbucket",
	"jira":       "Jira",
	"confluence": "Confluence",
	"clickup":    "ClickUp",
	"trello":     "Trello",
	"notion":     "Notion",
	"linear":     "Linear",
	"slack":      "Slack",
	"wiki":       "Wiki",
	"ci":         "CI",
	"staging":    "Staging",
	"prod":       "Production",
	"docs":       "Docs",
}

// RenderMetadataSections emits Links / Team / Cost markdown sections in
// that order, separated by blank lines. Each section is omitted when
// there's nothing to show, so unused metadata leaves no empty headers.
//
// The output is intended to be appended to the LLM-generated body before
// the final ## Tags section — see ingestion.go for the insertion point.
func RenderMetadataSections(links []LinkEntry, team TeamData, cost CostData) string {
	var parts []string
	if s := RenderLinks(links); s != "" {
		parts = append(parts, s)
	}
	if s := RenderTeam(team); s != "" {
		parts = append(parts, s)
	}
	if s := RenderCost(cost); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n\n") + "\n"
}

// RenderLinks emits a Links section. Entries are sorted with well-known
// keys first (in the wellKnownLinkLabels map order), then unknown keys
// alphabetically. Empty slice returns "".
func RenderLinks(links []LinkEntry) string {
	if len(links) == 0 {
		return ""
	}
	sorted := sortLinks(links)

	var b strings.Builder
	b.WriteString("## Links\n\n")
	for _, e := range sorted {
		label := wellKnownLinkLabels[e.Key]
		if label == "" {
			label = strings.Title(e.Key) //nolint:staticcheck // Title is fine for ASCII keys
		}
		line := fmt.Sprintf("- [%s](%s)", label, e.URL)
		if e.FromClient {
			line += " *(inherited from client)*"
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// sortLinks returns links ordered by a stable rule: well-known keys in
// their wellKnownLinkLabels iteration order first, unknown keys after
// sorted alphabetically. Deterministic across runs for diff-friendly
// wiki output.
func sortLinks(links []LinkEntry) []LinkEntry {
	// Build priority for well-known keys.
	knownOrder := []string{
		"github", "gitlab", "bitbucket",
		"jira", "linear", "clickup", "trello", "notion",
		"confluence", "wiki", "docs",
		"slack",
		"ci", "staging", "prod",
	}
	priority := make(map[string]int, len(knownOrder))
	for i, k := range knownOrder {
		priority[k] = i
	}

	out := make([]LinkEntry, len(links))
	copy(out, links)
	sort.SliceStable(out, func(i, j int) bool {
		pi, iKnown := priority[out[i].Key]
		pj, jKnown := priority[out[j].Key]
		switch {
		case iKnown && jKnown:
			return pi < pj
		case iKnown && !jKnown:
			return true
		case !iKnown && jKnown:
			return false
		default:
			return out[i].Key < out[j].Key
		}
	})
	return out
}

// RenderTeam emits a Team section with field-level inheritance flags.
// Returns "" when no fields are set.
func RenderTeam(t TeamData) string {
	var rows []string
	if t.Lead != "" {
		rows = append(rows, teamRow("Lead", formatContact(t.Lead), t.LeadFromClient))
	}
	if t.OncallChannel != "" {
		rows = append(rows, teamRow("On-call", t.OncallChannel, t.OncallFromClient))
	}
	if t.Escalation != "" {
		rows = append(rows, teamRow("Escalation", formatContact(t.Escalation), t.EscFromClient))
	}
	if t.Notes != "" {
		rows = append(rows, teamRow("Notes", t.Notes, t.NotesFromClient))
	}
	if len(rows) == 0 {
		return ""
	}
	return "## Team\n\n" + strings.Join(rows, "\n")
}

func teamRow(label, value string, fromClient bool) string {
	out := "- **" + label + ":** " + value
	if fromClient {
		out += " *(inherited from client)*"
	}
	return out
}

// formatContact turns `user@example.com` into a markdown `mailto:` link;
// non-email strings are passed through unchanged (e.g. `#channel` names,
// free-form notes).
func formatContact(s string) string {
	if strings.Contains(s, "@") && !strings.Contains(s, " ") {
		return fmt.Sprintf("[%s](mailto:%s)", s, s)
	}
	return s
}

// RenderCost emits the Cost section. If any numeric field is set we
// render a calculation table; otherwise we render the how-to-estimate
// framework template (which is the documentation — no separate doc
// file).
func RenderCost(c CostData) string {
	hasData := c.InfraMonthlyUSD > 0 || c.TeamFTE > 0 || c.TeamFTERateMonthlyUSD > 0
	if !hasData {
		return renderCostFramework()
	}

	var b strings.Builder
	b.WriteString("## Cost\n\n| Item | Monthly (USD) |\n|---|---|\n")

	total := 0.0
	rendered := false
	if c.InfraMonthlyUSD > 0 {
		fmt.Fprintf(&b, "| Infrastructure | $%s |\n", formatUSD(c.InfraMonthlyUSD))
		total += c.InfraMonthlyUSD
		rendered = true
	}
	if c.TeamFTE > 0 && c.TeamFTERateMonthlyUSD > 0 {
		teamTotal := c.TeamFTE * c.TeamFTERateMonthlyUSD
		fmt.Fprintf(&b, "| Team (%s FTE × $%s/mo) | $%s |\n",
			formatFloat(c.TeamFTE), formatUSD(c.TeamFTERateMonthlyUSD), formatUSD(teamTotal))
		total += teamTotal
		rendered = true
	} else if c.TeamFTE > 0 {
		fmt.Fprintf(&b, "| Team FTE | %s *(no rate set — see cost.team_fte_rate_usd_monthly)* |\n", formatFloat(c.TeamFTE))
	} else if c.TeamFTERateMonthlyUSD > 0 {
		fmt.Fprintf(&b, "| Team FTE rate | $%s/mo *(no FTE count set)* |\n", formatUSD(c.TeamFTERateMonthlyUSD))
	}
	if rendered && total > 0 {
		fmt.Fprintf(&b, "| **Total** | **$%s** |\n", formatUSD(total))
	}

	if c.Notes != "" {
		fmt.Fprintf(&b, "\n%s", c.Notes)
	}
	if c.FromClient {
		b.WriteString("\n\n*(some fields inherited from client baseline)*")
	}

	return strings.TrimRight(b.String(), "\n")
}

// renderCostFramework is the how-to-estimate doc that displays when no
// numbers have been entered for a project. It shows the exact YAML keys
// the user should fill in — the framework IS the doc.
func renderCostFramework() string {
	return "## Cost\n\n" +
		"_No cost data recorded yet._ To populate this section, add to your project's `llmwiki.yaml`:\n\n" +
		"```yaml\n" +
		"cost:\n" +
		"  infra_monthly_usd: <AWS/GCP/etc bill for this project, monthly average>\n" +
		"  team_fte: <FTE count primarily working on this project>\n" +
		"  team_fte_rate_usd_monthly: <fully-loaded monthly cost per FTE>\n" +
		"```\n\n" +
		"Or set `team_fte_rate_usd_monthly` once at the client level in `~/.llmwiki/clients/<customer>.yaml` and individual projects only need to supply their FTE count + infra number."
}

// formatUSD renders a dollar amount with thousands separators and no
// decimals (cents are noise at the scales wiki-level costs track).
func formatUSD(v float64) string {
	n := int64(v + 0.5) // nearest-integer round
	s := fmt.Sprintf("%d", n)
	// Insert commas every three digits from the right.
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// formatFloat drops trailing zeros after the decimal (2.0 → "2", 2.5 → "2.5").
func formatFloat(v float64) string {
	s := fmt.Sprintf("%g", v)
	return s
}
