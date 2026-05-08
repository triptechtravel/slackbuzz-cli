package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuzzyMatch_ExactWins(t *testing.T) {
	matched, tier, ok := fuzzyMatch("michelle", []string{"alice", "michelle", "bob"})
	require.True(t, ok)
	assert.Equal(t, "michelle", matched)
	assert.Equal(t, matchExact, tier)
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	matched, tier, ok := fuzzyMatch("Michelle", []string{"alice", "michelle", "bob"})
	require.True(t, ok)
	assert.Equal(t, "michelle", matched)
	assert.Equal(t, matchExact, tier)
}

func TestFuzzyMatch_ContainsSingle(t *testing.T) {
	matched, tier, ok := fuzzyMatch("mich", []string{"alice", "michelle", "bob"})
	require.True(t, ok)
	assert.Equal(t, "michelle", matched)
	assert.Equal(t, matchContains, tier)
}

func TestFuzzyMatch_ContainsMultiplePicksShortest(t *testing.T) {
	// "stand" matches both "stand-up" (8) and "standardisation" (15).
	// Shortest wins (most-specific).
	matched, tier, ok := fuzzyMatch("stand", []string{"stand-up", "standardisation"})
	require.True(t, ok)
	assert.Equal(t, "stand-up", matched)
	assert.Equal(t, matchContains, tier)
}

func TestFuzzyMatch_FuzzyLastResort(t *testing.T) {
	// "mihelle" isn't contained in any candidate (missing the 'c') but
	// fuzzy-ranks "michelle" highest by edit distance.
	matched, tier, ok := fuzzyMatch("mihelle", []string{"alice", "michelle", "bob"})
	require.True(t, ok)
	assert.Equal(t, "michelle", matched)
	assert.Equal(t, matchFuzzy, tier)
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	_, _, ok := fuzzyMatch("nonsense", []string{"alice", "michelle", "bob"})
	assert.False(t, ok)
}

func TestFuzzyMatch_EmptyTarget(t *testing.T) {
	_, _, ok := fuzzyMatch("", []string{"alice"})
	assert.False(t, ok)
}

func TestSuggestSimilar(t *testing.T) {
	suggestions := suggestSimilar("alic", []string{"alice", "alex", "bob", "alfred"}, 3)
	require.NotEmpty(t, suggestions)
	assert.Contains(t, suggestions, "alice")
	assert.LessOrEqual(t, len(suggestions), 3)
}

func TestSuggestSimilar_ReturnsEmptyForUnrelated(t *testing.T) {
	// fuzzysearch's RankMatchNormalizedFold returns -1 for completely
	// unrelated strings — those shouldn't show up as suggestions.
	suggestions := suggestSimilar("zzz", []string{"alice", "bob"}, 3)
	assert.Empty(t, suggestions)
}
