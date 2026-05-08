package api

import (
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

// matchTier identifies how a fuzzy match was reached. Used by callers
// (and `slackbuzz doctor`) to surface "did you mean" hints when a target
// resolved via something looser than an exact match.
type matchTier int

const (
	matchExact matchTier = iota
	matchContains
	matchFuzzy
)

// fuzzyMatch tries to find a single best candidate from `available` for the
// user-supplied `target`. Mirrors the clickup-cli pattern:
//
//  1. Case-insensitive exact match — wins outright.
//  2. Case-insensitive contains match — if exactly one or unambiguous,
//     wins; if multiple, the shortest wins (most-specific).
//  3. Fuzzy ranked match (RankMatchNormalizedFold) — best rank wins.
//
// Returns (matched, tier, ok). The caller can use `tier` to decide whether
// to print a confirmation hint ("matched 'michelle' for 'michell'"); exact
// matches don't deserve one, fuzzy matches usually do.
func fuzzyMatch(target string, available []string) (string, matchTier, bool) {
	if target == "" || len(available) == 0 {
		return "", matchExact, false
	}

	lowerTarget := strings.ToLower(target)
	lowerAvail := make([]string, len(available))
	for i, s := range available {
		lowerAvail[i] = strings.ToLower(s)
	}

	// Tier 1: exact case-insensitive match.
	for i, l := range lowerAvail {
		if l == lowerTarget {
			return available[i], matchExact, true
		}
	}

	// Tier 2: substring match. Prefer single match; if multiple, the
	// shortest candidate wins (most-specific).
	var contains []string
	for i, l := range lowerAvail {
		if strings.Contains(l, lowerTarget) {
			contains = append(contains, available[i])
		}
	}
	if len(contains) == 1 {
		return contains[0], matchContains, true
	}
	if len(contains) > 1 {
		sort.Slice(contains, func(i, j int) bool { return len(contains[i]) < len(contains[j]) })
		return contains[0], matchContains, true
	}

	// Tier 3: ranked fuzzy match.
	type ranked struct {
		name string
		rank int
	}
	var matches []ranked
	for _, s := range available {
		rank := fuzzy.RankMatchNormalizedFold(target, s)
		if rank >= 0 {
			matches = append(matches, ranked{s, rank})
		}
	}
	if len(matches) == 0 {
		return "", matchExact, false
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].rank < matches[j].rank })
	return matches[0].name, matchFuzzy, true
}

// suggestSimilar returns up to n candidate names that look like the target,
// for inclusion in error messages. "Did you mean: alice, alexandra, alex?"
func suggestSimilar(target string, available []string, n int) []string {
	if len(available) == 0 {
		return nil
	}
	type ranked struct {
		name string
		rank int
	}
	var matches []ranked
	for _, s := range available {
		rank := fuzzy.RankMatchNormalizedFold(target, s)
		if rank >= 0 {
			matches = append(matches, ranked{s, rank})
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].rank < matches[j].rank })
	if len(matches) > n {
		matches = matches[:n]
	}
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.name
	}
	return out
}
