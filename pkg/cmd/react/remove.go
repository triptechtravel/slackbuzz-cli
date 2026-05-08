package react

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type removeOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
	emoji   string
}

// NewCmdRemove returns the "react remove" subcommand.
func NewCmdRemove(f *cmdutil.Factory) *cobra.Command {
	opts := &removeOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "remove <channel> <timestamp> <emoji>",
		Short: "Remove a reaction from a message",
		Long: `Remove an emoji reaction from a message.

The emoji should be in :emoji: format (colons are stripped automatically).
Requires a bot token (xoxb-) with reactions:write scope.`,
		Example: `  slackbuzz react remove #general 1706000000.000000 :eyes:
  slackbuzz react remove #general 1706000000.000000 thumbsup`,
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: cmdutil.NeedsBotToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			opts.emoji = args[2]
			return removeRun(opts)
		},
	}

	return cmd
}

func removeRun(opts *removeOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.BotClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.API)
	channelID, _, err := resolver.ResolveTarget(opts.channel)
	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	emoji := strings.Trim(opts.emoji, ":")

	if _, err := slackapi.ReactionsRemove(context.Background(), client.API, &slackapi.ReactionsRemoveParams{
		Channel:   channelID,
		Timestamp: opts.ts,
		Name:      emoji,
	}); err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Removed :%s: from message %s in %s\n",
		cs.Green("✓"), emoji, opts.ts, opts.channel)
	return nil
}
