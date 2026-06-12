package text

import (
	"strings"
	"testing"
)

func TestUnescapeShellArtifacts(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "backslash bang",
			text: `Build passed\!`,
			want: "Build passed!",
		},
		{
			name: "backslash question mark",
			text: `Is this working\?`,
			want: "Is this working?",
		},
		{
			name: "multiple escapes",
			text: `Hey\! What\? Really\!`,
			want: "Hey! What? Really!",
		},
		{
			name: "no escapes unchanged",
			text: "Hello world",
			want: "Hello world",
		},
		{
			name: "escaped backslash collapses",
			text: `path\\to\\file`,
			want: `path\to\file`,
		},
		{
			name: "mixed content",
			text: `Build 104 going out\! We can pair after`,
			want: "Build 104 going out! We can pair after",
		},
		{
			name: "empty string",
			text: "",
			want: "",
		},
		{
			name: "only escape",
			text: `\!`,
			want: "!",
		},
		{
			name: "literal newlines",
			text: `line one\n\nline two`,
			want: "line one\n\nline two",
		},
		{
			name: "literal tab",
			text: `col1\tcol2`,
			want: "col1\tcol2",
		},
		{
			name: "mixed escapes with newlines",
			text: `Build passed\!\n\nDetails at link`,
			want: "Build passed!\n\nDetails at link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnescapeShellArtifacts(tt.text)
			if got != tt.want {
				t.Errorf("UnescapeShellArtifacts(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestNormalizeOutgoing_ConvertsMarkdown(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "bold",
			text: "this is **important** news",
			want: "this is *important* news",
		},
		{
			name: "header becomes bold line",
			text: "# Release Notes\nbody",
			want: "*Release Notes*\nbody",
		},
		{
			name: "link",
			text: "see [the docs](https://example.com)",
			want: "see <https://example.com|the docs>",
		},
		{
			name: "plain text untouched",
			text: "nothing fancy here",
			want: "nothing fancy here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := NormalizeOutgoing(tt.text, false)
			if got != tt.want {
				t.Errorf("NormalizeOutgoing(%q, false) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestNormalizeOutgoing_RawSkipsConversionButStillUnescapes(t *testing.T) {
	got, hints := NormalizeOutgoing(`**stay bold**\!`, true)
	if got != "**stay bold**!" {
		t.Errorf("raw mode: got %q, want %q", got, "**stay bold**!")
	}
	if len(hints) != 0 {
		t.Errorf("raw mode should produce no hints, got %d", len(hints))
	}
}

func TestNormalizeOutgoing_UnescapesBeforeConverting(t *testing.T) {
	// The \n must become a real newline BEFORE conversion so the header
	// regex (which anchors on line starts) sees a line boundary.
	got, _ := NormalizeOutgoing(`intro\n## Section`, false)
	if !strings.Contains(got, "\n*Section*") {
		t.Errorf("expected header on its own line to convert after unescape, got %q", got)
	}
}

func TestNormalizeOutgoing_ReportsHints(t *testing.T) {
	_, hints := NormalizeOutgoing("some **bold** text", false)
	if len(hints) == 0 {
		t.Fatal("expected at least one format hint for **bold** input")
	}
}

func TestNormalizeOutgoing_NoHintsForPlainText(t *testing.T) {
	_, hints := NormalizeOutgoing("just words", false)
	if len(hints) != 0 {
		t.Errorf("expected no hints for plain text, got %d", len(hints))
	}
}
