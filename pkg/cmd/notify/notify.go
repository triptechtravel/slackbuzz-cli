package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	slacktext "github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type notifyOptions struct {
	factory *cmdutil.Factory
	channel string
	release string
	task    string
	status  string
	message string
	json    cmdutil.JSONFlags
}

// NewCmdNotify returns the "notify" command.
func NewCmdNotify(f *cmdutil.Factory) *cobra.Command {
	opts := &notifyOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "notify <channel>",
		Short: "Send formatted notifications",
		Long: `Send formatted notification messages to a Slack channel.

Supports release announcements, task status updates, and custom messages.`,
		Example: `  # Release announcement
  slackbuzz notify #releases --release v1.0.0

  # Task status update
  slackbuzz notify #updates --task CU-abc123 --status "deployed"

  # Custom message
  slackbuzz notify #general --message "Maintenance window starting"`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			if opts.release == "" && opts.task == "" && opts.message == "" {
				return fmt.Errorf("specify --release, --task, or --message")
			}
			return notifyRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.release, "release", "", "Release version to announce")
	cmd.Flags().StringVar(&opts.task, "task", "", "ClickUp task ID for status update")
	cmd.Flags().StringVar(&opts.status, "status", "", "Task status (used with --task)")
	cmd.Flags().StringVar(&opts.message, "message", "", "Custom notification message")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func notifyRun(opts *notifyOptions) error {
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

	// Shell unescape + Markdown→mrkdwn, same pipeline as message send/edit.
	// The custom message renders via mrkdwn section blocks, so raw Markdown
	// (e.g. **bold**) would otherwise arrive broken.
	message := opts.message
	if message != "" {
		var hints []slacktext.FormatHint
		message, hints = slacktext.NormalizeOutgoing(message, false)
		cmdutil.PrintFormatHints(ios, hints)
	}

	// Resolve @mentions in custom message text
	if message != "" && strings.Contains(message, "@") {
		resolved, names, resolveErr := resolver.ResolveMentions(message)
		if resolveErr == nil {
			message = resolved
			for _, name := range names {
				fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
			}
		}
	}

	var blocks []slackapi.Block
	var fallbackText string

	switch {
	case opts.release != "":
		blocks = buildReleaseBlocks(opts.release)
		fallbackText = fmt.Sprintf("🚀 Release %s has been released.", opts.release)
	case opts.task != "":
		blocks = buildTaskBlocks(opts.task, opts.status)
		fallbackText = fmt.Sprintf("📋 Task Update: %s", opts.task)
		if opts.status != "" {
			fallbackText += fmt.Sprintf(" — %s", opts.status)
		}
	case message != "":
		blocks = buildMessageBlocks(message)
		fallbackText = message
	}

	resp, err := slackapi.ChatPostMessage(context.Background(), client.API, &slackapi.ChatPostMessageParams{
		Channel: channelID,
		Text:    fallbackText,
		Blocks:  slackapi.BlocksJSON(blocks),
	})
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	respChannel := resp.Channel
	respTS := resp.TS

	if opts.json.WantsJSON() {
		result := map[string]string{
			"channel":   respChannel,
			"timestamp": respTS,
		}
		return opts.json.OutputJSON(ios.Out, result)
	}

	fmt.Fprintf(ios.Out, "%s Notification sent to %s\n",
		cs.Green("✓"), cs.Bold(opts.channel))
	return nil
}

func buildReleaseBlocks(version string) []slackapi.Block {
	header := slackapi.NewHeaderBlock(slackapi.NewPlainText(fmt.Sprintf("🚀 Release %s", version)))
	section := slackapi.NewSectionBlock(
		slackapi.NewMrkdwnText(fmt.Sprintf("*Version %s* has been released.\n\nCheck the release notes for details.", version)),
		nil, nil)
	divider := slackapi.NewDividerBlock()
	return []slackapi.Block{header, divider, section}
}

func buildTaskBlocks(taskID, status string) []slackapi.Block {
	taskURL := fmt.Sprintf("https://app.clickup.com/t/%s", taskID)

	var statusLine string
	if status != "" {
		statusLine = fmt.Sprintf("\n*Status:* %s", status)
	}

	section := slackapi.NewSectionBlock(
		slackapi.NewMrkdwnText(fmt.Sprintf("📋 *Task Update*\n\nTask: <%s|%s>%s", taskURL, taskID, statusLine)),
		nil, nil)
	return []slackapi.Block{section}
}

func buildMessageBlocks(message string) []slackapi.Block {
	section := slackapi.NewSectionBlock(slackapi.NewMrkdwnText(message), nil, nil)
	if strings.Contains(message, "\n") {
		divider := slackapi.NewDividerBlock()
		return []slackapi.Block{divider, section, divider}
	}
	return []slackapi.Block{section}
}
