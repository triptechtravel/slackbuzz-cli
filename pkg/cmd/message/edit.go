package message

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	slacktext "github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type editOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
	text    string
	asBot   bool
	raw     bool
	json    cmdutil.JSONFlags
}

// NewCmdEdit returns the "message edit" command.
func NewCmdEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "edit <channel> <timestamp> [new-text]",
		Short: "Edit a message",
		Long: `Update the text of an existing Slack message.

The channel argument accepts #channel-name or a channel ID.
If new-text is omitted, reads from stdin (for piping).

The new text goes through the same pipeline as message send: standard
Markdown (**bold**, # headers, [links](url)) is converted to Slack mrkdwn
and @mentions are resolved. Use --raw to disable the Markdown conversion.`,
		Example: `  # Edit a message
  slackbuzz message edit #general 1706000000.000000 "updated text"

  # Pipe new text
  echo "corrected text" | slackbuzz message edit #general 1706000000.000000`,
		Args:              cobra.RangeArgs(2, 3),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			if len(args) > 2 {
				opts.text = args[2]
			}
			return editRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.asBot, "as-bot", false, "Edit as the bot instead of your user account")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Disable Markdown→mrkdwn auto-conversion")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func editRun(opts *editOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	var client *api.Client
	var err error
	if opts.asBot {
		client, err = opts.factory.DefaultClient()
	} else {
		client, err = opts.factory.UserClient()
	}
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.API)

	channelID, _, err := resolver.ResolveTarget(opts.channel)
	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	// Get message text from args or stdin
	text := opts.text
	if text == "" {
		if !ios.IsTerminal() {
			data, readErr := io.ReadAll(ios.In)
			if readErr != nil {
				return fmt.Errorf("failed to read from stdin: %w", readErr)
			}
			text = strings.TrimSpace(string(data))
		} else {
			scanner := bufio.NewScanner(ios.In)
			fmt.Fprint(ios.Out, "New text: ")
			if scanner.Scan() {
				text = strings.TrimSpace(scanner.Text())
			}
		}
	}

	if text == "" {
		return fmt.Errorf("new message text cannot be empty")
	}

	// Same text pipeline as message send: shell unescape, Markdown→mrkdwn,
	// and @mention resolution.
	text, hints := slacktext.NormalizeOutgoing(text, opts.raw)
	cmdutil.PrintFormatHints(ios, hints)

	if strings.Contains(text, "@") {
		resolved, names, err := resolver.ResolveMentions(text)
		if err == nil {
			text = resolved
			for _, name := range names {
				fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
			}
		}
	}

	if _, err := slackapi.ChatUpdate(context.Background(), client.API, &slackapi.ChatUpdateParams{
		Channel: channelID,
		TS:      opts.ts,
		Text:    text,
	}); err != nil {
		return fmt.Errorf("%s", api.FormatError(err))
	}

	if opts.json.WantsJSON() {
		result := map[string]string{
			"channel":   channelID,
			"timestamp": opts.ts,
			"action":    "edited",
		}
		return opts.json.OutputJSON(ios.Out, result)
	}

	fmt.Fprintf(ios.Out, "%s Message %s in %s updated\n",
		cs.Green("✓"), opts.ts, cs.Bold(opts.channel))
	return nil
}
