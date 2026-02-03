package channel

import (
	"fmt"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/tableprinter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type listOptions struct {
	factory     *cmdutil.Factory
	channelType string
	limit       int
	json        cmdutil.JSONFlags
}

// NewCmdList returns the "channel list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		Long:  "List Slack channels in the workspace.",
		Example: `  # List public channels
  slackbuzz channel list

  # List private channels
  slackbuzz channel list --type private

  # List DMs
  slackbuzz channel list --type im

  # Output as JSON
  slackbuzz channel list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.channelType, "type", "public", "Channel type: public, private, im, mpim")
	cmd.Flags().IntVar(&opts.limit, "limit", 100, "Maximum number of channels to list")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	typeMap := map[string]string{
		"public":  "public_channel",
		"private": "private_channel",
		"im":      "im",
		"mpim":    "mpim",
	}

	slackType, ok := typeMap[opts.channelType]
	if !ok {
		return fmt.Errorf("invalid channel type %q (use: public, private, im, mpim)", opts.channelType)
	}

	params := &slack.GetConversationsParameters{
		Types:           []string{slackType},
		Limit:           opts.limit,
		ExcludeArchived: true,
	}

	var allChannels []slack.Channel
	for {
		channels, nextCursor, err := client.Slack.GetConversations(params)
		if err != nil {
			return fmt.Errorf("failed to list channels: %w", err)
		}
		allChannels = append(allChannels, channels...)
		if nextCursor == "" || len(allChannels) >= opts.limit {
			break
		}
		params.Cursor = nextCursor
	}

	if len(allChannels) > opts.limit {
		allChannels = allChannels[:opts.limit]
	}

	if len(allChannels) == 0 {
		fmt.Fprintln(ios.Out, "No channels found.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, allChannels)
	}

	tp := tableprinter.New(ios)
	for _, ch := range allChannels {
		tp.AddField(cs.Bold(ch.ID))
		name := ch.Name
		if ch.IsPrivate {
			name = "🔒 " + name
		}
		tp.AddField(name)
		tp.AddField(fmt.Sprintf("%d members", ch.NumMembers))
		purpose := ch.Purpose.Value
		tp.AddField(cs.Gray(purpose))
		tp.EndRow()
	}

	return tp.Render()
}
