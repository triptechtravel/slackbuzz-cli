package text

import "testing"

func TestFormatSlackText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "user mention with name",
			input: "Hey <@U018Y0Y9HSN|Isaac Rowntree>, check this",
			want:  "Hey @Isaac Rowntree, check this",
		},
		{
			name:  "user mention without name",
			input: "Hey <@U018Y0Y9HSN>, check this",
			want:  "Hey @U018Y0Y9HSN, check this",
		},
		{
			name:  "channel mention with name",
			input: "See <#C012345|general> for details",
			want:  "See #general for details",
		},
		{
			name:  "channel mention without name",
			input: "See <#C012345> for details",
			want:  "See #C012345 for details",
		},
		{
			name:  "link with label",
			input: "Visit <https://example.com|our site> now",
			want:  "Visit our site now",
		},
		{
			name:  "link without label",
			input: "Visit <https://example.com> now",
			want:  "Visit https://example.com now",
		},
		{
			name:  "broadcast mention",
			input: "<!here> please review",
			want:  "@here please review",
		},
		{
			name:  "broadcast channel",
			input: "<!channel> important update",
			want:  "@channel important update",
		},
		{
			name:  "emoji shortcode",
			input: "Great work :tada: :rocket:",
			want:  "Great work 🎉 🚀",
		},
		{
			name:  "unknown emoji preserved",
			input: "Custom :custom_emoji: here",
			want:  "Custom :custom_emoji: here",
		},
		{
			name:  "HTML entities",
			input: "if x &gt; 0 &amp;&amp; y &lt; 10",
			want:  "if x > 0 && y < 10",
		},
		{
			name:  "combined formatting",
			input: "<@U123|alice> posted in <#C456|dev>: check <https://github.com/org/repo/pull/42|PR #42> :eyes: :thumbsup:",
			want:  "@alice posted in #dev: check PR #42 👀 👍",
		},
		{
			name:  "plain text unchanged",
			input: "just a normal message",
			want:  "just a normal message",
		},
		{
			name:  "group mention",
			input: "<!subteam^S12345|@backend-team> please review",
			want:  "@backend-team please review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSlackText(tt.input)
			if got != tt.want {
				t.Errorf("FormatSlackText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReplaceEmoji(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{":thumbsup:", "👍"},
		{":rocket:", "🚀"},
		{":fire: and :heart:", "🔥 and ❤️"},
		{":unknown_emoji:", ":unknown_emoji:"},
		{"no emoji here", "no emoji here"},
		{":white_check_mark:", "✅"},
		{":slightly_smiling_face:", "🙂"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ReplaceEmoji(tt.input)
			if got != tt.want {
				t.Errorf("ReplaceEmoji(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
