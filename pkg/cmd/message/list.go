package message

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type listOptions struct {
	factory  *cmdutil.Factory
	channel  string
	limit    int
	since    string
	threadTS string
	json     cmdutil.JSONFlags
}

// NewCmdList returns the "message list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "list <channel>",
		Short: "List messages in a channel",
		Long: `Read message history from a Slack channel or thread.

The channel argument accepts #channel-name or a channel ID.`,
		Example: `  # List recent messages
  slackbuzz message list #general

  # List with limit
  slackbuzz message list #general --limit 20

  # List since a date
  slackbuzz message list #general --since 2026-01-01

  # List thread replies
  slackbuzz message list #general --thread 1234567890.123456

  # Output as JSON
  slackbuzz message list #general --json`,
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			return listRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 10, "Number of messages to fetch")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only show messages after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.threadTS, "thread", "", "Thread timestamp to read replies from")
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

	resolver := api.NewResolver(client.Slack)

	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	var messages []slack.Message

	if opts.threadTS != "" {
		// Fetch thread replies
		msgs, _, _, err := client.Slack.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: channelID,
			Timestamp: opts.threadTS,
			Limit:     opts.limit,
		})
		if err != nil {
			return fmt.Errorf("failed to get thread replies: %w", err)
		}
		messages = msgs
	} else {
		// Fetch channel history
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Limit:     opts.limit,
		}

		if opts.since != "" {
			sinceTime, parseErr := time.Parse("2006-01-02", opts.since)
			if parseErr != nil {
				return fmt.Errorf("invalid --since date (use YYYY-MM-DD): %w", parseErr)
			}
			params.Oldest = fmt.Sprintf("%d.000000", sinceTime.Unix())
		}

		resp, err := client.Slack.GetConversationHistory(params)
		if err != nil {
			return fmt.Errorf("failed to get channel history: %w", err)
		}
		messages = resp.Messages
	}

	if len(messages) == 0 {
		fmt.Fprintln(ios.Out, "No messages found.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, messages)
	}

	teamID, _ := auth.GetTeamInfo()

	// Print messages in reverse order (oldest first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		ts := parseSlackTimestamp(msg.Timestamp)
		user := msg.User
		if user == "" {
			user = msg.BotID
		}
		if user == "" {
			user = "unknown"
		}

		timeStr := text.RelativeTime(ts)
		msgText := text.FormatSlackText(msg.Text)

		deeplink := text.SlackDeeplink(teamID, channelID, text.FormatMessageTS(msg.Timestamp), "")
		tsDisplay := msg.Timestamp
		if deeplink != "" {
			tsDisplay = text.Hyperlink(deeplink, msg.Timestamp)
		}

		fmt.Fprintf(ios.Out, "%s  %s  %s\n",
			cs.Bold(user),
			cs.Gray(timeStr),
			cs.Gray(tsDisplay),
		)
		fmt.Fprintf(ios.Out, "  %s\n", msgText)

		// Show reactions if present
		if len(msg.Reactions) > 0 {
			var parts []string
			for _, r := range msg.Reactions {
				parts = append(parts, fmt.Sprintf("%s %d", text.EmojiToUnicode(r.Name), r.Count))
			}
			fmt.Fprintf(ios.Out, "  %s\n", strings.Join(parts, "  "))
		}
		fmt.Fprintln(ios.Out)
	}

	return nil
}

func parseSlackTimestamp(ts string) time.Time {
	// Slack timestamps are Unix timestamps like "1234567890.123456"
	var sec, usec int64
	fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	return time.Unix(sec, usec*1000)
}
