package message

import (
	"fmt"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type searchOptions struct {
	factory *cmdutil.Factory
	query   string
	channel string
	from    string
	since   string
	until   string
	limit   int
	json    cmdutil.JSONFlags
}

// NewCmdSearch returns the "message search" command.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	opts := &searchOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search messages (requires user token)",
		Long: `Search Slack messages using the search.messages API.

This command requires a user token (xoxp-) because the search API
does not support bot tokens.`,
		Example: `  # Search for messages
  slackbuzz message search "deploy production"

  # Search in a specific channel
  slackbuzz message search "error" --channel #general

  # Search from a user since a date
  slackbuzz message search "fix" --from @alice --since 2026-01-01`,
		Args: cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.query = args[0]
			return searchRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.channel, "channel", "", "Filter by channel")
	cmd.Flags().StringVar(&opts.from, "from", "", "Filter by user")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only show messages after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.until, "until", "", "Only show messages before this date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of results")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func searchRun(opts *searchOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	// Build search query with modifiers
	query := opts.query
	if opts.channel != "" {
		query += fmt.Sprintf(" in:%s", opts.channel)
	}
	if opts.from != "" {
		query += fmt.Sprintf(" from:%s", opts.from)
	}
	if opts.since != "" {
		query += fmt.Sprintf(" after:%s", opts.since)
	}
	if opts.until != "" {
		query += fmt.Sprintf(" before:%s", opts.until)
	}

	params := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         opts.limit,
	}

	result, err := client.Slack.SearchMessages(query, params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if result.Total == 0 {
		fmt.Fprintln(ios.Out, "No messages found.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, result.Matches)
	}

	teamID, _ := auth.GetTeamInfo()

	fmt.Fprintf(ios.Out, "%s results for %q\n\n", cs.Bold(fmt.Sprintf("%d", result.Total)), opts.query)

	for _, match := range result.Matches {
		ts := parseSlackTimestamp(match.Timestamp)
		timeStr := text.RelativeTime(ts)

		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		msgText := text.FormatSlackText(match.Text)

		deeplink := text.SlackDeeplink(teamID, match.Channel.ID, text.FormatMessageTS(match.Timestamp), "")
		channelDisplay := "#" + channelName
		if deeplink != "" {
			channelDisplay = text.Hyperlink(deeplink, channelDisplay)
		}

		fmt.Fprintf(ios.Out, "%s  %s  %s\n",
			cs.Bold(match.Username),
			cs.Cyan(channelDisplay),
			cs.Gray(timeStr),
		)
		fmt.Fprintf(ios.Out, "  %s\n\n", msgText)
	}

	if result.Total > len(result.Matches) {
		remaining := result.Total - len(result.Matches)
		fmt.Fprintf(ios.Out, "%s\n", cs.Gray(fmt.Sprintf("... and %d more results", remaining)))
	}

	return nil
}
