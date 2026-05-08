package threads

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type threadsOptions struct {
	factory *cmdutil.Factory
	since   string
	limit   int
	json    cmdutil.JSONFlags
}

// NewCmdThreads returns the "threads" command.
func NewCmdThreads(f *cmdutil.Factory) *cobra.Command {
	opts := &threadsOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "threads",
		Short: "Show threads you're participating in",
		Long: `List threads where you've posted, mirroring Slack's Threads sidebar.

Uses the search API to find threads where you're active.
Requires a user token (xoxp-).`,
		Example: `  # Recent threads
  slackbuzz threads

  # Threads from the last day
  slackbuzz threads --since 1d

  # Output as JSON
  slackbuzz threads --json`,
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return threadsRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.since, "since", "", "Only show threads after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of results")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

type threadItem struct {
	Channel   string          `json:"channel"`
	ChannelID string          `json:"channel_id"`
	RootUser  string          `json:"root_user"`
	RootText  string          `json:"root_text"`
	YourReply string          `json:"your_reply,omitempty"`
	Timestamp string          `json:"timestamp"`
	Permalink string          `json:"permalink,omitempty"`
	Hints     []activity.Hint `json:"hints,omitempty"`
}

func threadsRun(opts *threadsOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	userID, userName, resolveErr := auth.ResolveUserID()
	if resolveErr != nil {
		return fmt.Errorf("could not determine your Slack user ID: %w", resolveErr)
	}

	teamID, _ := auth.GetTeamInfo()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	text.LoadCustomEmoji(client.Token())

	query := fmt.Sprintf("from:<@%s> is:thread", userID)

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
		Count:   opts.limit,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	type searchMatch struct {
		Username  string `json:"username"`
		User      string `json:"user"`
		Text      string `json:"text"`
		TS        string `json:"ts"`
		Permalink string `json:"permalink"`
		Channel   struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"channel"`
	}
	var searchResult struct {
		Messages struct {
			Total   int           `json:"total"`
			Matches []searchMatch `json:"matches"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Raw, &searchResult); err != nil {
		return fmt.Errorf("decode search result: %w", err)
	}

	if searchResult.Messages.Total == 0 {
		fmt.Fprintln(ios.Out, "No threads found.")
		return nil
	}

	items := make([]threadItem, 0, len(searchResult.Messages.Matches))
	for _, match := range searchResult.Messages.Matches {
		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		yourReply := ""
		if match.Username == userName || match.User == userID {
			yourReply = match.Text
		}

		item := threadItem{
			Channel:   channelName,
			ChannelID: match.Channel.ID,
			RootUser:  match.Username,
			RootText:  match.Text,
			YourReply: yourReply,
			Timestamp: match.TS,
			Permalink: match.Permalink,
			Hints:     activity.ExtractHints(match.Text),
		}
		items = append(items, item)
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, items)
	}

	// Reverse items so most recent is at the bottom (natural terminal reading order)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	for _, item := range items {
		ts := activity.ParseSlackTimestamp(item.Timestamp)
		timeStr := text.RelativeTime(ts)

		rootText := text.FormatSlackText(item.RootText)

		fmt.Fprintf(ios.Out, "  #%-18s %s    %s: %q\n",
			cs.Cyan(item.Channel),
			cs.Gray(timeStr),
			cs.Bold(item.RootUser),
			rootText,
		)

		if item.YourReply != "" {
			fmt.Fprintf(ios.Out, "    You replied: %q\n", text.FormatSlackText(item.YourReply))
		}

		deeplink := text.SlackDeeplink(teamID, item.ChannelID, text.FormatMessageTS(item.Timestamp), "")
		if deeplink != "" {
			fmt.Fprintf(ios.Out, "    %s %s\n", cs.Gray("→"), cs.Cyan(text.Hyperlink(deeplink, "Open in Slack")))
		}

		for _, hint := range item.Hints {
			fmt.Fprintf(ios.Out, "    %s %s\n", cs.Gray("→"), cs.Green(hint.Command))
		}

		fmt.Fprintln(ios.Out)
	}

	return nil
}
