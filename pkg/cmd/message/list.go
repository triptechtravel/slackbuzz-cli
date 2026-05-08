package message

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/recent"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
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
		Use:   "list <channel-or-user>",
		Short: "List messages in a channel or DM",
		Long: `Read message history from a Slack channel, DM, or thread.

The argument accepts:
  #channel-name or channel ID — read channel history
  @user, user ID, or bare username — read DM history (auto-opens the DM conversation)`,
		Example: `  # List recent messages in a channel
  slackbuzz message list #general

  # List recent DM history with a user
  slackbuzz message list @alice

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

	// DM reads need the user token both for resolution (so conversations.open
	// returns the *user's* DM channel, not the bot's separate one) and for
	// history (the bot isn't a member of the user's 1:1s). For channels, the
	// default (bot-first) client is fine.
	var client *api.Client
	useUserClient := api.LooksLikeUser(opts.channel)
	if useUserClient {
		uc, err := opts.factory.UserClient()
		if err != nil {
			return err
		}
		client = uc
	} else {
		c, err := opts.factory.DefaultClient()
		if err != nil {
			return err
		}
		client = c
	}

	resolver := api.NewResolver(client.API)

	// Resolve target to a channel ID — auto-detect DMs vs channels.
	channelID, isDM, err := resolver.ResolveTarget(opts.channel)
	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	// Record the target as a recent context default. The slot we write to
	// depends on whether this was a DM or a channel — `slackbuzz dm` (no
	// args) reads "dm"; future `slackbuzz channel` defaults could use "channel".
	slot := "channel"
	if isDM {
		slot = "dm"
	}
	_ = recent.Push(slot, opts.channel)

	var messages []*slackapi.Message
	ctx := context.Background()

	if opts.threadTS != "" {
		resp, err := slackapi.ConversationsReplies(ctx, client.API, &slackapi.ConversationsRepliesParams{
			Channel: channelID,
			TS:      opts.threadTS,
			Limit:   opts.limit,
		})
		if err != nil {
			return fmt.Errorf("failed to get thread replies: %w", err)
		}
		messages = resp.Messages
	} else {
		params := &slackapi.ConversationsHistoryParams{
			Channel: channelID,
			Limit:   opts.limit,
		}
		if opts.since != "" {
			sinceTime, parseErr := time.Parse("2006-01-02", opts.since)
			if parseErr != nil {
				return fmt.Errorf("invalid --since date (use YYYY-MM-DD): %w", parseErr)
			}
			params.Oldest = fmt.Sprintf("%d.000000", sinceTime.Unix())
		}

		resp, err := slackapi.ConversationsHistory(ctx, client.API, params)
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
		ts := parseSlackTimestamp(msg.TS)
		user := msg.User
		if user == "" {
			user = msg.BotID
		}
		if user == "" {
			user = "unknown"
		}

		timeStr := text.RelativeTime(ts)
		msgText := text.FormatSlackText(msg.Text)

		deeplink := text.SlackDeeplink(teamID, channelID, text.FormatMessageTS(msg.TS), "")
		tsDisplay := msg.TS
		if deeplink != "" {
			tsDisplay = text.Hyperlink(deeplink, msg.TS)
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

	// Quick actions footer
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz message send %s \"text\" --thread-ts <ts>\n", cs.Gray("Reply:"), opts.channel)
	fmt.Fprintf(ios.Out, "  %s  slackbuzz react %s <ts> :emoji:\n", cs.Gray("React:"), opts.channel)
	fmt.Fprintf(ios.Out, "  %s  slackbuzz message edit %s <ts> \"new text\"\n", cs.Gray("Edit:"), opts.channel)

	return nil
}

func parseSlackTimestamp(ts string) time.Time {
	// Slack timestamps are Unix timestamps like "1234567890.123456"
	var sec, usec int64
	fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	return time.Unix(sec, usec*1000)
}
