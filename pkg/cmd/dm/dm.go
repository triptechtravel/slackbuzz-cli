package dm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/recent"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// openLastDM is the behaviour of `slackbuzz dm` with no subcommand —
// shows the user's recent DM history (newest first) so they can pick one
// without remembering the exact handle. Single-shot output; doesn't open
// a TUI.
func openLastDM(f *cmdutil.Factory, _ string) error {
	cs := f.IOStreams.ColorScheme()
	out := f.IOStreams.Out

	entries := recent.List("dm", 8)
	if len(entries) == 0 {
		fmt.Fprintln(out, "No recent DMs recorded yet — try `slackbuzz dm list` first.")
		return nil
	}

	fmt.Fprintln(out, cs.Bold("Recent DMs (newest first)"))
	for _, e := range entries {
		when := text.RelativeTime(e.When)
		fmt.Fprintf(out, "  %-24s %s\n", cs.Cyan(e.Target), cs.Gray(when))
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s slackbuzz message list %s\n", cs.Gray("Read:"), entries[0].Target)
	fmt.Fprintf(out, "%s slackbuzz message send %s \"text\"\n", cs.Gray("Send:"), entries[0].Target)
	return nil
}

// NewCmdDM returns the "dm" command group.
func NewCmdDM(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dm [<user>]",
		Short: "Direct message management",
		Long: `Direct-message management.

Reading and sending individual DMs works via:
  slackbuzz message list @user
  slackbuzz message send @user "text"

Running ` + "`slackbuzz dm`" + ` with no subcommand opens the most recent DM
conversation (per the local recents file).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No-arg invocation: open the most recent DM, mirroring clickup-cli's
			// "default to last context" pattern.
			last := recent.Last("dm")
			if last == "" {
				return cmd.Help()
			}
			fmt.Fprintf(f.IOStreams.Out, "Opening last DM: %s\n", last)
			return openLastDM(f, last)
		},
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
	User         string `json:"user"`
	MessageCount int    `json:"message_count"`
	LastMessage  string `json:"last_message"`
	LastTime     string `json:"last_time"`
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

	resp, err := slackapi.SearchMessages(context.Background(), client.API, &slackapi.SearchMessagesParams{
		Query:   query,
		Sort:    "timestamp",
		SortDir: "desc",
		Count:   100, // fetch more to group by user
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	var searchResult struct {
		Messages struct {
			Total   int `json:"total"`
			Matches []struct {
				User     string `json:"user"`
				Username string `json:"username"`
				Text     string `json:"text"`
				TS       string `json:"ts"`
			} `json:"matches"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Raw, &searchResult); err != nil {
		return fmt.Errorf("decode search result: %w", err)
	}

	if searchResult.Messages.Total == 0 {
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

	for _, match := range searchResult.Messages.Matches {
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
		if acc.lastTS == "" || match.TS > acc.lastTS {
			acc.lastTS = match.TS
			acc.lastMessage = match.Text
		}
	}

	// Convert to slice and sort by most recent
	conversations := make([]dmConversation, 0, len(grouped))
	for user, acc := range grouped {
		msg := text.FormatSlackText(acc.lastMessage)
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

	// Reverse so most recent is at the bottom (natural terminal reading order)
	for i, j := 0, len(conversations)-1; i < j; i, j = i+1, j-1 {
		conversations[i], conversations[j] = conversations[j], conversations[i]
	}

	for i, conv := range conversations {
		ts := activity.ParseSlackTimestamp(conv.LastTime)
		timeStr := text.RelativeTime(ts)

		fmt.Fprintf(ios.Out, "  @%-14s %s  %s\n",
			cs.Bold(conv.User),
			cs.Gray(text.Pluralize(conv.MessageCount, "message")),
			cs.Gray(timeStr),
		)
		fmt.Fprintf(ios.Out, "    last: %q\n", text.Truncate(conv.LastMessage, 80))

		if i < len(conversations)-1 {
			fmt.Fprintln(ios.Out)
		}
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz message list @<user>\n", cs.Gray("Read:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz message send @<user> \"text\"\n", cs.Gray("Reply:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz later add @<user> <ts>\n", cs.Gray("Save:"))

	return nil
}
