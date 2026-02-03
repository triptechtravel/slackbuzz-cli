package message

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type editOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
	text    string
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
If new-text is omitted, reads from stdin (for piping).`,
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

	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func editRun(opts *editOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.Slack)

	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
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

	_, _, _, err = client.Slack.UpdateMessage(channelID, opts.ts, slack.MsgOptionText(text, false))
	if err != nil {
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
