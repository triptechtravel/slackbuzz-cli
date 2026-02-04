package notify

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
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

	resolver := api.NewResolver(client.Slack)
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	// Resolve @mentions in custom message text
	message := opts.message
	if message != "" && strings.Contains(message, "@") {
		resolved, names, resolveErr := resolver.ResolveMentions(message)
		if resolveErr == nil {
			message = resolved
			for _, name := range names {
				fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
			}
		}
	}

	var blocks []slack.Block
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

	msgOpts := []slack.MsgOption{
		slack.MsgOptionText(fallbackText, false),
		slack.MsgOptionBlocks(blocks...),
	}

	respChannel, respTS, err := client.Slack.PostMessage(channelID, msgOpts...)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

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

func buildReleaseBlocks(version string) []slack.Block {
	headerText := slack.NewTextBlockObject("plain_text", fmt.Sprintf("🚀 Release %s", version), true, false)
	header := slack.NewHeaderBlock(headerText)

	body := slack.NewTextBlockObject("mrkdwn",
		fmt.Sprintf("*Version %s* has been released.\n\nCheck the release notes for details.", version),
		false, false)
	section := slack.NewSectionBlock(body, nil, nil)

	divider := slack.NewDividerBlock()

	return []slack.Block{header, divider, section}
}

func buildTaskBlocks(taskID, status string) []slack.Block {
	taskURL := fmt.Sprintf("https://app.clickup.com/t/%s", taskID)

	var statusLine string
	if status != "" {
		statusLine = fmt.Sprintf("\n*Status:* %s", status)
	}

	body := slack.NewTextBlockObject("mrkdwn",
		fmt.Sprintf("📋 *Task Update*\n\nTask: <%s|%s>%s",
			taskURL, taskID, statusLine),
		false, false)
	section := slack.NewSectionBlock(body, nil, nil)

	return []slack.Block{section}
}

func buildMessageBlocks(message string) []slack.Block {
	// If message is short, use a section. If multi-line, preserve formatting.
	body := slack.NewTextBlockObject("mrkdwn", message, false, false)
	section := slack.NewSectionBlock(body, nil, nil)

	if strings.Contains(message, "\n") {
		divider := slack.NewDividerBlock()
		return []slack.Block{divider, section, divider}
	}

	return []slack.Block{section}
}
