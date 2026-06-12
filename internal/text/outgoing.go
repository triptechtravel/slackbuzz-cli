package text

import "strings"

// UnescapeShellArtifacts removes common shell escape sequences that leak into
// CLI arguments. For example, zsh's history expansion escapes ! as \! when
// passed through double-quoted strings, and literal \n sequences appear when
// users include newlines in quoted arguments.
func UnescapeShellArtifacts(text string) string {
	// Protect escaped backslashes (\\) before processing other sequences
	const placeholder = "\x00BACKSLASH\x00"
	text = strings.ReplaceAll(text, `\\`, placeholder)
	text = strings.ReplaceAll(text, `\!`, "!")
	text = strings.ReplaceAll(text, `\?`, "?")
	text = strings.ReplaceAll(text, `\n`, "\n")
	text = strings.ReplaceAll(text, `\t`, "\t")
	text = strings.ReplaceAll(text, placeholder, `\`)
	return text
}

// NormalizeOutgoing prepares user-supplied message text for posting to Slack:
// it strips shell escape artifacts and (unless raw is set) converts Markdown
// to Slack mrkdwn. The returned hints describe conversions worth surfacing to
// the user; they are non-empty only when conversion ran and found Markdown
// syntax. Every command that posts or updates user-supplied text must go
// through this — chat.postMessage and chat.update render identically, so
// skipping it on any path (as message edit and notify once did) ships raw
// Markdown to Slack.
func NormalizeOutgoing(text string, raw bool) (string, []FormatHint) {
	text = UnescapeShellArtifacts(text)
	if raw {
		return text, nil
	}
	hints := DetectFormatHints(text)
	return ConvertMarkdownToMrkdwn(text), hints
}
