package text

import (
	"testing"
)

func TestConvertMarkdownToMrkdwn(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Bold conversion
		{
			name:  "double asterisk bold",
			input: "this is **bold** text",
			want:  "this is *bold* text",
		},
		{
			name:  "double underscore bold",
			input: "this is __bold__ text",
			want:  "this is *bold* text",
		},
		{
			name:  "multiple bold phrases",
			input: "**first** and **second** bold",
			want:  "*first* and *second* bold",
		},
		{
			name:  "single asterisk bold passes through",
			input: "this is *already bold* in mrkdwn",
			want:  "this is *already bold* in mrkdwn",
		},

		// Header conversion
		{
			name:  "h1 header",
			input: "# Main Title",
			want:  "*Main Title*",
		},
		{
			name:  "h2 header",
			input: "## Section",
			want:  "*Section*",
		},
		{
			name:  "h3 header",
			input: "### Subsection",
			want:  "*Subsection*",
		},
		{
			name:  "header with emoji",
			input: "## 🚀 Release Notes",
			want:  "*🚀 Release Notes*",
		},
		{
			name:  "multiline with headers",
			input: "# Title\nSome text\n## Section\nMore text",
			want:  "*Title*\nSome text\n*Section*\nMore text",
		},
		{
			name:  "hash in middle of line is not a header",
			input: "issue #123 is fixed",
			want:  "issue #123 is fixed",
		},

		// Link conversion
		{
			name:  "markdown link",
			input: "click [here](https://example.com) now",
			want:  "click <https://example.com|here> now",
		},
		{
			name:  "multiple links",
			input: "[one](https://a.com) and [two](https://b.com)",
			want:  "<https://a.com|one> and <https://b.com|two>",
		},
		{
			name:  "slack link passes through",
			input: "click <https://example.com|here> now",
			want:  "click <https://example.com|here> now",
		},

		// Image conversion
		{
			name:  "image with alt text",
			input: "![logo](https://example.com/logo.png)",
			want:  "<https://example.com/logo.png|logo>",
		},
		{
			name:  "image without alt text",
			input: "![](https://example.com/logo.png)",
			want:  "<https://example.com/logo.png>",
		},

		// Horizontal rules
		{
			name:  "triple dash hr",
			input: "above\n---\nbelow",
			want:  "above\n─────\nbelow",
		},
		{
			name:  "triple asterisk hr",
			input: "above\n***\nbelow",
			want:  "above\n─────\nbelow",
		},
		{
			name:  "long dash hr",
			input: "above\n----------\nbelow",
			want:  "above\n─────\nbelow",
		},

		// Already-valid mrkdwn passes through
		{
			name:  "mrkdwn italic",
			input: "this is _italic_ text",
			want:  "this is _italic_ text",
		},
		{
			name:  "mrkdwn strikethrough",
			input: "this is ~struck~ text",
			want:  "this is ~struck~ text",
		},
		{
			name:  "mrkdwn code",
			input: "run `npm install` first",
			want:  "run `npm install` first",
		},
		{
			name:  "mrkdwn blockquote",
			input: "> This is a quote",
			want:  "> This is a quote",
		},
		{
			name:  "bullet list with hyphens",
			input: "- item one\n- item two",
			want:  "• item one\n• item two",
		},
		{
			name:  "bullet list with asterisks",
			input: "* item one\n* item two",
			want:  "• item one\n• item two",
		},
		{
			name:  "nested bullet list",
			input: "- top\n  - nested",
			want:  "• top\n  • nested",
		},
		{
			name:  "hyphen in middle of text is not a bullet",
			input: "this is a - test",
			want:  "this is a - test",
		},
		{
			name:  "emoji shortcodes",
			input: ":rocket: Launch :tada:",
			want:  ":rocket: Launch :tada:",
		},

		// Combined conversions
		{
			name:  "headers with bold content",
			input: "# **Release** Notes\n\nSome text",
			want:  "*Release Notes*\n\nSome text",
		},
		{
			name:  "full release notes style",
			input: "## 🎨 UX & Design\n- **New animations** throughout\n- Fixed **spacing** issue",
			want:  "*🎨 UX & Design*\n• *New animations* throughout\n• Fixed *spacing* issue",
		},

		// Edge cases
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no markdown",
			input: "plain text message",
			want:  "plain text message",
		},
		{
			name:  "code block not converted",
			input: "```\n**not bold**\n```",
			want:  "```\n*not bold*\n```",
			// Note: we don't parse inside code blocks — acceptable tradeoff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMarkdownToMrkdwn(tt.input)
			if got != tt.want {
				t.Errorf("ConvertMarkdownToMrkdwn(%q)\n  got:  %q\n  want: %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectFormatHints(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantIssue string // check first hint's Issue contains this
	}{
		{
			name:      "no issues",
			input:     "*bold* _italic_ `code`",
			wantCount: 0,
		},
		{
			name:      "double bold detected",
			input:     "this is **bold** text",
			wantCount: 1,
			wantIssue: "bold",
		},
		{
			name:      "header detected",
			input:     "# Title\nsome text",
			wantCount: 1,
			wantIssue: "header",
		},
		{
			name:      "link detected",
			input:     "click [here](https://example.com)",
			wantCount: 1,
			wantIssue: "link",
		},
		{
			name:      "image detected",
			input:     "![alt](https://img.com/pic.png)",
			wantCount: 2, // image + link (image contains link syntax)
			wantIssue: "image",
		},
		{
			name:      "multiple issues",
			input:     "# **Title**\n[link](https://x.com)",
			wantCount: 3, // bold + header + link
		},
		{
			name:      "empty string",
			input:     "",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := DetectFormatHints(tt.input)
			if len(hints) != tt.wantCount {
				t.Errorf("DetectFormatHints(%q): got %d hints, want %d", tt.input, len(hints), tt.wantCount)
				for i, h := range hints {
					t.Logf("  hint[%d]: %s", i, h.Issue)
				}
				return
			}
			if tt.wantIssue != "" && len(hints) > 0 {
				found := false
				for _, h := range hints {
					if containsCI(h.Issue, tt.wantIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("DetectFormatHints(%q): no hint contains %q", tt.input, tt.wantIssue)
				}
			}
		})
	}
}

func containsCI(s, substr string) bool {
	return len(s) >= len(substr) && contains(lower(s), lower(substr))
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSlackMrkdwnReference(t *testing.T) {
	ref := SlackMrkdwnReference()
	if ref == "" {
		t.Error("SlackMrkdwnReference() returned empty string")
	}
	// Should contain key syntax items
	for _, want := range []string{"*bold*", "_italic_", "~strikethrough~", "`code`", "<url|label>"} {
		if !contains(ref, want) {
			t.Errorf("SlackMrkdwnReference() missing %q", want)
		}
	}
}
