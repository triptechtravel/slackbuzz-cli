package message

import "testing"

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
			name: "legitimate backslash preserved",
			text: `path\\to\\file`,
			want: `path\\to\\file`,
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
