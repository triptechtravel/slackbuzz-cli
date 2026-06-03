package activity

import (
	"testing"
)

func TestExtractHints(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []Hint
	}{
		{
			name: "clickup task ID",
			text: "Can you check CU-abc123?",
			want: []Hint{{Label: "ClickUp", Command: "clickup task view CU-abc123"}},
		},
		{
			name: "github PR URL",
			text: "See https://github.com/org/repo/pull/42 for details",
			want: []Hint{{Label: "GitHub", Command: "gh pr view 42"}},
		},
		{
			name: "github issue URL",
			text: "Related to https://github.com/org/repo/issues/99",
			want: []Hint{{Label: "GitHub", Command: "gh issue view 99"}},
		},
		{
			name: "multiple hints",
			text: "CU-xyz789 and https://github.com/org/repo/pull/10",
			want: []Hint{
				{Label: "ClickUp", Command: "clickup task view CU-xyz789"},
				{Label: "GitHub", Command: "gh pr view 10"},
			},
		},
		{
			name: "duplicate clickup IDs",
			text: "CU-abc123 mentioned again CU-abc123",
			want: []Hint{{Label: "ClickUp", Command: "clickup task view CU-abc123"}},
		},
		{
			name: "no hints",
			text: "just a regular message",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHints(tt.text)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractHints() returned %d hints, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Label != tt.want[i].Label || got[i].Command != tt.want[i].Command {
					t.Errorf("hint[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	got := uniqueStrings([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("uniqueStrings() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("uniqueStrings()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
