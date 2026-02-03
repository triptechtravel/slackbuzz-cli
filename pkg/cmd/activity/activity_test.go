package activity

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		err   bool
	}{
		{"2h", 2 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"3m", 90 * 24 * time.Hour, false},
		{"abc", 0, true},
		{"", 0, true},
		{"2x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.err {
				if err == nil {
					t.Fatalf("parseDuration(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDuration(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		input string
		err   bool
	}{
		{"", false},
		{"2026-01-15", false},
		{"1d", false},
		{"2h", false},
		{"7d", false},
		{"2w", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSince(tt.input)
			if tt.err {
				if err == nil {
					t.Fatalf("parseSince(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSince(%q) unexpected error: %v", tt.input, err)
			}
			if tt.input == "" && got != "" {
				t.Errorf("parseSince(\"\") = %q, want \"\"", got)
			}
			if tt.input == "2026-01-15" && got != "2026-01-15" {
				t.Errorf("parseSince(\"2026-01-15\") = %q, want \"2026-01-15\"", got)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		opts      *activityOptions
		afterDate string
		want      string
	}{
		{
			name:      "base only",
			base:      "<@U123>",
			opts:      &activityOptions{},
			afterDate: "",
			want:      "<@U123>",
		},
		{
			name:      "with channel",
			base:      "<@U123>",
			opts:      &activityOptions{channel: "#engineering"},
			afterDate: "",
			want:      "<@U123> in:engineering",
		},
		{
			name:      "with from",
			base:      "<@U123>",
			opts:      &activityOptions{from: "@alice"},
			afterDate: "",
			want:      "<@U123> from:alice",
		},
		{
			name:      "with after date",
			base:      "<@U123>",
			opts:      &activityOptions{},
			afterDate: "2026-01-15",
			want:      "<@U123> after:2026-01-15",
		},
		{
			name:      "all modifiers",
			base:      "is:dm",
			opts:      &activityOptions{channel: "#dev", from: "@bob"},
			afterDate: "2026-01-01",
			want:      "is:dm in:dev from:bob after:2026-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQuery(tt.base, tt.opts, tt.afterDate)
			if got != tt.want {
				t.Errorf("buildQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeduplicateItems(t *testing.T) {
	items := []activityItem{
		{Timestamp: "1234.5678", Channel: "general", Text: "first"},
		{Timestamp: "1234.5678", Channel: "general", Text: "duplicate"},
		{Timestamp: "1234.9999", Channel: "general", Text: "different ts"},
		{Timestamp: "1234.5678", Channel: "random", Text: "different channel"},
	}

	got := deduplicateItems(items)
	if len(got) != 3 {
		t.Fatalf("deduplicateItems() returned %d items, want 3", len(got))
	}
	if got[0].Text != "first" {
		t.Errorf("first item text = %q, want %q", got[0].Text, "first")
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	ts := ParseSlackTimestamp("1706000000.123456")
	if ts.Unix() != 1706000000 {
		t.Errorf("parseSlackTimestamp() Unix = %d, want 1706000000", ts.Unix())
	}
}
