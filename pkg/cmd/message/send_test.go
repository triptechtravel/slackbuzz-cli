package message

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			got := unescapeShellArtifacts(tt.text)
			if got != tt.want {
				t.Errorf("unescapeShellArtifacts(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

// ─── splitForBlocks ────────────────────────────────────────────────────────

func TestSplitForBlocks_NoSplitNeeded(t *testing.T) {
	chunks := splitForBlocks("short text", 3000)
	require.Len(t, chunks, 1)
	assert.Equal(t, "short text", chunks[0])
}

func TestSplitForBlocks_SplitsAtParagraphBoundary(t *testing.T) {
	// 1500 + double newline + 1500 chars — splits cleanly at the paragraph
	// boundary rather than mid-sentence.
	first := strings.Repeat("a", 1500)
	second := strings.Repeat("b", 1500)
	input := first + "\n\n" + second

	chunks := splitForBlocks(input, 1800)
	require.Len(t, chunks, 2, "should split into two chunks")
	assert.Equal(t, first, chunks[0])
	assert.Equal(t, second, chunks[1])
}

func TestSplitForBlocks_FallsBackToSingleNewline(t *testing.T) {
	// No paragraph boundary; splitter falls back to single-newline.
	first := strings.Repeat("a", 1500)
	second := strings.Repeat("b", 1500)
	input := first + "\n" + second

	chunks := splitForBlocks(input, 1800)
	require.Len(t, chunks, 2)
	assert.Equal(t, first, chunks[0])
	assert.Equal(t, second, chunks[1])
}

func TestSplitForBlocks_HardSplitAsLastResort(t *testing.T) {
	// Single 4000-char block with no newline — must hard-split.
	input := strings.Repeat("a", 4000)
	chunks := splitForBlocks(input, 3000)
	require.GreaterOrEqual(t, len(chunks), 2)
	for _, c := range chunks {
		assert.LessOrEqual(t, len(c), 3000)
	}
}

// ─── buildMrkdwnBlocks ─────────────────────────────────────────────────────

func TestBuildMrkdwnBlocks_SingleSection(t *testing.T) {
	blocks := buildMrkdwnBlocks("hello world")
	require.Len(t, blocks, 1)
}

func TestBuildMrkdwnBlocks_MultipleSectionsForLongText(t *testing.T) {
	// Force >3000 chars so the splitter creates multiple sections.
	long := strings.Repeat("x", 3500)
	blocks := buildMrkdwnBlocks(long)
	assert.GreaterOrEqual(t, len(blocks), 2)
}
