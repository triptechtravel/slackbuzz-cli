package cmdutil

import (
	"fmt"
	"strings"

	"github.com/isaacrowntree/slackbuzz-cli/internal/api"
	"github.com/isaacrowntree/slackbuzz-cli/internal/iostreams"
	slacktext "github.com/isaacrowntree/slackbuzz-cli/internal/text"
)

// PrintFormatHints surfaces Markdown→mrkdwn conversion notes on stderr. Used
// by every command that normalizes outgoing message text.
func PrintFormatHints(ios *iostreams.IOStreams, hints []slacktext.FormatHint) {
	if len(hints) == 0 {
		return
	}
	cs := ios.ColorScheme()
	fmt.Fprintf(ios.ErrOut, "%s Auto-converting Markdown → Slack mrkdwn\n", cs.Blue("→"))
	for _, h := range hints {
		fmt.Fprintf(ios.ErrOut, "  %s %s (%s)\n", cs.Yellow("⚡"), h.Issue, h.Example)
	}
}

// PrepareOutgoing runs message text through the pipeline every command that
// posts text uses: shell unescape, Markdown→mrkdwn (unless raw), then @mention
// resolution, telling the user on stderr what it changed.
func PrepareOutgoing(ios *iostreams.IOStreams, resolver *api.Resolver, text string, raw bool) string {
	text, hints := slacktext.NormalizeOutgoing(text, raw)
	PrintFormatHints(ios, hints)
	if !strings.Contains(text, "@") {
		return text
	}
	resolved, names, err := resolver.ResolveMentions(text)
	if err != nil {
		return text
	}
	cs := ios.ColorScheme()
	for _, name := range names {
		fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
	}
	return resolved
}
