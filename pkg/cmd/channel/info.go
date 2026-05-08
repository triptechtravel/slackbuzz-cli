package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type infoOptions struct {
	factory *cmdutil.Factory
	channel string
	json    cmdutil.JSONFlags
}

// NewCmdInfo returns the "channel info" command.
func NewCmdInfo(f *cmdutil.Factory) *cobra.Command {
	opts := &infoOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "info <channel>",
		Short: "Show channel details",
		Long:  "Display detailed information about a Slack channel.",
		Example: `  slackbuzz channel info #general
  slackbuzz channel info C012345 --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			return infoRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func infoRun(opts *infoOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.API)
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	resp, err := slackapi.ConversationsInfo(context.Background(), client.API, &slackapi.ConversationsInfoParams{
		Channel: channelID,
	})
	if err != nil {
		return fmt.Errorf("failed to get channel info: %w", err)
	}
	if resp.Channel == nil {
		return fmt.Errorf("channel %s not found", channelID)
	}
	ch := resp.Channel

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, ch)
	}

	created := time.Unix(int64(ch.Created), 0)

	fmt.Fprintf(ios.Out, "%s\n", cs.Bold(ch.Name))
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("ID:"), ch.ID)
	fmt.Fprintf(ios.Out, "  %-16s %d\n", cs.Bold("Members:"), ch.NumMembers)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Created:"), created.Format("2006-01-02"))
	if ch.Topic != nil && ch.Topic.Value != "" {
		fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Topic:"), ch.Topic.Value)
	}
	if ch.Purpose != nil && ch.Purpose.Value != "" {
		fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Purpose:"), ch.Purpose.Value)
	}
	fmt.Fprintf(ios.Out, "  %-16s %v\n", cs.Bold("Archived:"), ch.IsArchived)
	fmt.Fprintf(ios.Out, "  %-16s %v\n", cs.Bold("Private:"), ch.IsPrivate)

	return nil
}
