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
		Use:   "react [<channel> <timestamp> <emoji>]",
		Short: "React to a message",
		Long: `Add an emoji reaction to a message.

The emoji should be in :emoji: format (colons are stripped automatically).
Requires a bot token (xoxb-) with reactions:write scope.

Use "react remove" to remove a reaction.`,
		Example: `  slackbuzz react #general 1706000000.000000 :eyes:
  slackbuzz react #general 1706000000.000000 :white_check_mark:
  slackbuzz react #general 1706000000.000000 thumbsup
  slackbuzz react remove #general 1706000000.000000 :eyes:`,
		Args:              cobra.ExactArgs(3),
		PersistentPreRunE: cmdutil.NeedsBotToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			opts.emoji = args[2]
			return reactRun(opts)
		},
	}

	cmd.AddCommand(NewCmdRemove(f))

	return cmd
}

func reactRun(opts *reactOptions) error {
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

	// Strip colons from emoji name
	emoji := strings.Trim(opts.emoji, ":")

	if _, err := slackapi.ReactionsAdd(context.Background(), client.API, &slackapi.ReactionsAddParams{
		Channel:   channelID,
		Timestamp: opts.ts,
		Name:      emoji,
	}); err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Reacted with :%s: to message %s in %s\n",
		cs.Green("✓"), emoji, opts.ts, opts.channel)
	return nil
}
