package api

import (
	"strings"
	"testing"
)

// testResolver creates a Resolver with pre-populated users, bypassing the API.
func testResolver(users map[string]string) *Resolver {
	return &Resolver{
		channels:    make(map[string]string),
		users:       users,
		usersLoaded: true,
	}
}

func TestResolveMentions(t *testing.T) {
	tests := []struct {
		name          string
		users         map[string]string
		text          string
		wantText      string
		wantResolved  []string
		wantUnchanged bool
	}{
		{
			name:  "single mention",
			users: map[string]string{"alice": "U001"},
			text:  "hey @alice check this",
			wantText:     "hey <@U001> check this",
			wantResolved: []string{"alice"},
		},
		{
			name:  "multiple mentions",
			users: map[string]string{"alice": "U001", "bob": "U002"},
			text:  "@alice @bob please review",
			wantText:     "<@U001> <@U002> please review",
			wantResolved: []string{"alice", "bob"},
		},
		{
			name:          "no mentions",
			users:         map[string]string{"alice": "U001"},
			text:          "hello world",
			wantText:      "hello world",
			wantUnchanged: true,
		},
		{
			name:          "no @ symbol at all",
			users:         map[string]string{"alice": "U001"},
			text:          "plain text here",
			wantText:      "plain text here",
			wantUnchanged: true,
		},
		{
			name:  "unresolved name left as-is",
			users: map[string]string{"alice": "U001"},
			text:  "@unknown please help",
			wantText:      "@unknown please help",
			wantUnchanged: true,
		},
		{
			name:  "mixed resolved and unresolved",
			users: map[string]string{"alice": "U001"},
			text:  "@alice and @unknown test",
			wantText:     "<@U001> and @unknown test",
			wantResolved: []string{"alice"},
		},
		{
			name:  "case insensitive matching",
			users: map[string]string{"alice": "U001"},
			text:  "@Alice check this",
			wantText:     "<@U001> check this",
			wantResolved: []string{"Alice"},
		},
		{
			name:  "word boundary respected",
			users: map[string]string{"al": "U001"},
			text:  "@alice should not match",
			wantText:      "@alice should not match",
			wantUnchanged: true,
		},
		{
			name:  "mention at end of text",
			users: map[string]string{"bob": "U002"},
			text:  "ping @bob",
			wantText:     "ping <@U002>",
			wantResolved: []string{"bob"},
		},
		{
			name:  "already formatted slack mention skipped",
			users: map[string]string{"alice": "U001"},
			text:  "already <@U001> formatted",
			wantText:      "already <@U001> formatted",
			wantUnchanged: true,
		},
		{
			name:  "greedy longest match first",
			users: map[string]string{"alice": "U001", "alice.smith": "U003"},
			text:  "@alice.smith and @alice",
			wantText:     "<@U003> and <@U001>",
			wantResolved: []string{"alice.smith", "alice"},
		},
		{
			name:  "username with hyphens",
			users: map[string]string{"mary-jane": "U004"},
			text:  "hi @mary-jane",
			wantText:     "hi <@U004>",
			wantResolved: []string{"mary-jane"},
		},
		{
			name:  "username with periods",
			users: map[string]string{"john.doe": "U005"},
			text:  "ask @john.doe",
			wantText:     "ask <@U005>",
			wantResolved: []string{"john.doe"},
		},
		{
			name:  "mention followed by punctuation",
			users: map[string]string{"alice": "U001"},
			text:  "thanks @alice!",
			wantText:     "thanks <@U001>!",
			wantResolved: []string{"alice"},
		},
		{
			name:  "mention followed by comma",
			users: map[string]string{"alice": "U001", "bob": "U002"},
			text:  "@alice, @bob: please look",
			wantText:     "<@U001>, <@U002>: please look",
			wantResolved: []string{"alice", "bob"},
		},
		{
			name:          "email address not resolved",
			users:         map[string]string{"alice": "U001"},
			text:          "email me at user@alice.com",
			wantText:      "email me at user@alice.com",
			wantUnchanged: true,
		},
		// Partial first-name matching (simulates keys added by loadUsers)
		{
			name: "full display name with space resolves",
			users: map[string]string{
				"herman.gorbatovskii": "U006",
				"herman gorbatovskii": "U006",
				"herman":              "U006",
			},
			text:         "@herman gorbatovskii check this",
			wantText:     "<@U006> check this",
			wantResolved: []string{"herman gorbatovskii"},
		},
		{
			name: "partial first name resolves when followed by punctuation",
			users: map[string]string{
				"herman.gorbatovskii": "U006",
				"herman gorbatovskii": "U006",
				"herman":              "U006",
			},
			text:         "hey @herman, check this",
			wantText:     "hey <@U006>, check this",
			wantResolved: []string{"herman"},
		},
		{
			name: "partial first name from dotted username",
			users: map[string]string{
				"john.smith": "U007",
				"john":       "U007",
			},
			text:         "ask @john about it",
			wantText:     "ask <@U007> about it",
			wantResolved: []string{"john"},
		},
		{
			name: "full dotted username preferred over partial",
			users: map[string]string{
				"john.smith": "U007",
				"john":       "U007",
			},
			text:         "ask @john.smith about it",
			wantText:     "ask <@U007> about it",
			wantResolved: []string{"john.smith"},
		},
		{
			name: "ambiguous first name not resolved",
			users: map[string]string{
				"herman.gorbatovskii": "U006",
				"herman gorbatovskii": "U006",
				"herman.jones":        "U008",
				// no "herman" partial — ambiguous across two users
			},
			text:          "hey @herman, check this",
			wantText:      "hey @herman, check this",
			wantUnchanged: true,
		},
		{
			name: "partial first name at end of text",
			users: map[string]string{
				"sarah.connor": "U009",
				"sarah":        "U009",
			},
			text:         "ping @sarah",
			wantText:     "ping <@U009>",
			wantResolved: []string{"sarah"},
		},
		{
			name: "display name with space and partial both resolve in same message",
			users: map[string]string{
				"herman.gorbatovskii": "U006",
				"herman gorbatovskii": "U006",
				"herman":              "U006",
				"alice":               "U001",
			},
			text:         "@herman and @alice please review",
			wantText:     "<@U006> and <@U001> please review",
			wantResolved: []string{"herman", "alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testResolver(tt.users)
			got, resolved, err := r.ResolveMentions(tt.text)
			if err != nil {
				t.Fatalf("ResolveMentions() error = %v", err)
			}
			if got != tt.wantText {
				t.Errorf("text = %q, want %q", got, tt.wantText)
			}
			if tt.wantUnchanged {
				if len(resolved) != 0 {
					t.Errorf("resolved = %v, want empty", resolved)
				}
			} else {
				if len(resolved) != len(tt.wantResolved) {
					t.Fatalf("resolved count = %d, want %d: %v", len(resolved), len(tt.wantResolved), resolved)
				}
				for i, name := range tt.wantResolved {
					if !strings.EqualFold(resolved[i], name) {
						t.Errorf("resolved[%d] = %q, want %q", i, resolved[i], name)
					}
				}
			}
		})
	}
}

func TestFindSimilarUsers(t *testing.T) {
	tests := []struct {
		name  string
		users map[string]string
		query string
		want  []string
	}{
		{
			name:  "prefix match",
			users: map[string]string{"alice": "U001", "alicia": "U002", "bob": "U003"},
			query: "ali",
			want:  []string{"alice", "alicia"},
		},
		{
			name:  "edit distance match",
			users: map[string]string{"alice": "U001", "bob": "U002"},
			query: "alce",
			want:  []string{"alice"},
		},
		{
			name:  "no matches",
			users: map[string]string{"alice": "U001", "bob": "U002"},
			query: "zzz",
			want:  nil,
		},
		{
			name:  "exact prefix of input matches shorter names",
			users: map[string]string{"al": "U001", "bob": "U002"},
			query: "alice",
			want:  []string{"al"},
		},
		{
			name:  "case insensitive",
			users: map[string]string{"Alice": "U001"},
			query: "alice",
			want:  []string{"Alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testResolver(tt.users)
			got := r.FindSimilarUsers(tt.query)
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("FindSimilarUsers(%q) = %v, want empty", tt.query, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("FindSimilarUsers(%q) = %v (len %d), want %v (len %d)", tt.query, got, len(got), tt.want, len(tt.want))
			}
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					if g == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FindSimilarUsers(%q) = %v, missing %q", tt.query, got, w)
				}
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"alice", "alce", 1},
		{"kitten", "sitting", 3},
		{"alice", "bob", 5},
	}

	for _, tt := range tests {
		t.Run(tt.a+"→"+tt.b, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		c    byte
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', true},
		{'.', true},
		{' ', false},
		{'!', false},
		{',', false},
		{'@', false},
		{'<', false},
		{'>', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.c), func(t *testing.T) {
			got := isWordChar(tt.c)
			if got != tt.want {
				t.Errorf("isWordChar(%q) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}
