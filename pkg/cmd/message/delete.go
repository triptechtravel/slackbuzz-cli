package message

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type deleteOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
	confirm bool
	asBot   bool
	json    cmdutil.JSONFlags
}

// NewCmdDelete returns the "message delete" command.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "delete <channel> <timestamp>",
		Short: "Delete a message",
		Long: `Delete a Slack message by channel and timestamp.

Prompts for confirmation in interactive mode.
In non-interactive (piped) mode, --confirm is required.`,
		Example: `  # Delete with confirmation prompt
  slackbuzz message delete #general 1706000000.000000

  # Delete without prompt (scripting)
  slackbuzz message delete #general 1706000000.000000 --confirm`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			return deleteRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.confirm, "confirm", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&opts.asBot, "as-bot", false, "Delete as the bot instead of your user account")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func deleteRun(opts *deleteOptions) error {
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

	if !opts.confirm {
		if !ios.IsTerminal() {
			return fmt.Errorf("--confirm is required in non-interactive mode")
		}
		p := prompter.New(ios)
		confirmed, promptErr := p.Confirm(
			fmt.Sprintf("Delete message %s in %s?", opts.ts, opts.channel), false)
		if promptErr != nil {
			return promptErr
		}
		if !confirmed {
			fmt.Fprintln(ios.Out, "Cancelled.")
			return nil
		}
	}

	if _, err := slackapi.ChatDelete(context.Background(), client.API, &slackapi.ChatDeleteParams{
		Channel: channelID,
		TS:      opts.ts,
	}); err != nil {
		return fmt.Errorf("%s", api.FormatError(err))
	}

	if opts.json.WantsJSON() {
		result := map[string]string{
			"channel":   channelID,
			"timestamp": opts.ts,
			"action":    "deleted",
		}
		return opts.json.OutputJSON(ios.Out, result)
	}

	fmt.Fprintf(ios.Out, "%s Message %s in %s deleted\n",
		cs.Green("✓"), opts.ts, cs.Bold(opts.channel))
	return nil
}
