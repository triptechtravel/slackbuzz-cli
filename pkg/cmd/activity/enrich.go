package activity

import "regexp"

var (
	clickupRe = regexp.MustCompile(`CU-[a-z0-9]+`)
	ghPRRe    = regexp.MustCompile(`github\.com/[^/]+/[^/]+/pull/(\d+)`)
	ghIssueRe = regexp.MustCompile(`github\.com/[^/]+/[^/]+/issues/(\d+)`)
)

// Hint represents an actionable CLI hint extracted from a message.
type Hint struct {
	Label   string // e.g. "ClickUp" or "GitHub"
	Command string // e.g. "clickup task view CU-abc123"
}

// ExtractHints detects ClickUp task IDs and GitHub PR/issue URLs in text
// and returns actionable CLI hints.
func ExtractHints(text string) []Hint {
	var hints []Hint

	for _, id := range uniqueStrings(clickupRe.FindAllString(text, -1)) {
		hints = append(hints, Hint{
			Label:   "ClickUp",
			Command: "clickup task view " + id,
		})
	}

	for _, m := range ghPRRe.FindAllStringSubmatch(text, -1) {
		hints = append(hints, Hint{
			Label:   "GitHub",
			Command: "gh pr view " + m[1],
		})
	}

	for _, m := range ghIssueRe.FindAllStringSubmatch(text, -1) {
		hints = append(hints, Hint{
			Label:   "GitHub",
			Command: "gh issue view " + m[1],
		})
	}

	return hints
}

// uniqueStrings deduplicates a string slice while preserving order.
func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
