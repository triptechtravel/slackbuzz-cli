package dm

import (
	"fmt"
	"sort"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdDM returns the "dm" command group.
func NewCmdDM(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dm <command>",
		Short: "Direct message management",
		Long: `List DM conversations with recent activity.

Reading and sending individual DMs works via:
  slackbuzz message list @user
  slackbuzz message send @user "text"`,
	}

	cmd.AddCommand(newCmdList(f))

	return cmd
}

type listOptions struct {
	factory *cmdutil.Factory
	since   string
	limit   int
	json    cmdutil.JSONFlags
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List DM conversations with recent activity",
		Long: `Show your DM conversations grouped by contact.

Uses the search API to find recent DMs.
Requires a user token (xoxp-).`,
		Example: `  # List recent DMs
  slackbuzz dm list

  # DMs from the last day
  slackbuzz dm list --since 1d

  # Output as JSON
  slackbuzz dm list --json`,
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.since, "since", "", "Only show DMs after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of conversations")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

type dmConversation struct {
	User        string `json:"user"`
	MessageCount int   `json:"message_count"`
	LastMessage string `json:"last_message"`
	LastTime    string `json:"last_time"`
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	query := "is:dm"

	if opts.since != "" {
		afterDate, parseErr := activity.ParseSince(opts.since)
		if parseErr != nil {
			return parseErr
		}
		if afterDate != "" {
			query += fmt.Sprintf(" after:%s", afterDate)
		}
	}

	params := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         100, // fetch more to group by user
	}

	result, err := client.Slack.SearchMessages(query, params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if result.Total == 0 {
		fmt.Fprintln(ios.Out, "No DM conversations found.")
		return nil
	}

	// Group messages by username
	type conversationAcc struct {
		count       int
		lastMessage string
		lastTS      string
	}
	grouped := make(map[string]*conversationAcc)

	for _, match := range result.Matches {
		user := match.Username
		if user == "" {
			user = match.User
		}
		if user == "" {
			continue
		}

		acc, ok := grouped[user]
		if !ok {
			acc = &conversationAcc{}
			grouped[user] = acc
		}
		acc.count++
		if acc.lastTS == "" || match.Timestamp > acc.lastTS {
			acc.lastTS = match.Timestamp
			acc.lastMessage = match.Text
		}
	}

	// Convert to slice and sort by most recent
	conversations := make([]dmConversation, 0, len(grouped))
	for user, acc := range grouped {
		msg := text.FormatSlackText(acc.lastMessage)
		if len(msg) > 60 {
			msg = text.Truncate(msg, 60)
		}
		conversations = append(conversations, dmConversation{
			User:         user,
			MessageCount: acc.count,
			LastMessage:  msg,
			LastTime:     acc.lastTS,
		})
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].LastTime > conversations[j].LastTime
	})

	if len(conversations) > opts.limit {
		conversations = conversations[:opts.limit]
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, conversations)
	}

	for _, conv := range conversations {
		ts := activity.ParseSlackTimestamp(conv.LastTime)
		timeStr := text.RelativeTime(ts)

		fmt.Fprintf(ios.Out, "  @%-14s %s    last: %-10s %q\n",
			cs.Bold(conv.User),
			cs.Gray(text.Pluralize(conv.MessageCount, "message")),
			cs.Gray(timeStr),
			conv.LastMessage,
		)
	}

	return nil
}
