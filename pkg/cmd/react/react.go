package react

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type reactOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
	emoji   string
}

// NewCmdReact returns the "react" command.
func NewCmdReact(f *cmdutil.Factory) *cobra.Command {
	opts := &reactOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "react <channel> <timestamp> <emoji>",
		Short: "React to a message",
		Long: `Add an emoji reaction to a message.

The emoji should be in :emoji: format (colons are stripped automatically).
Requires a bot token (xoxb-) with reactions:write scope.`,
		Example: `  slackbuzz react #general 1706000000.000000 :eyes:
  slackbuzz react #general 1706000000.000000 :white_check_mark:
  slackbuzz react #general 1706000000.000000 thumbsup`,
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: cmdutil.NeedsBotToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			opts.emoji = args[2]
			return reactRun(opts)
		},
	}

	return cmd
}

func reactRun(opts *reactOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.BotClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.Slack)
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	// Strip colons from emoji name
	emoji := strings.Trim(opts.emoji, ":")

	ref := slack.ItemRef{
		Channel:   channelID,
		Timestamp: opts.ts,
	}

	if err := client.Slack.AddReaction(emoji, ref); err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Reacted with :%s: to message %s in %s\n",
		cs.Green("✓"), emoji, opts.ts, opts.channel)
	return nil
}
