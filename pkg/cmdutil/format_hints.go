package cmdutil

import (
	"fmt"

	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	slacktext "github.com/triptechtravel/slackbuzz-cli/internal/text"
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
