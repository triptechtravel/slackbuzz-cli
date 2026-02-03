package message

import (
	"fmt"
	"strings"

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
	has     string
	is      string
	during  string
	sort    string
	page    int
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
  slackbuzz message search "fix" --from @alice --since 2026-01-01

  # Search with emoji filter
  slackbuzz message search "deploy" --has :rocket:

  # Search threads from this week
  slackbuzz message search "bug" --is thread --during week

  # Sort by relevance, page 2
  slackbuzz message search "deploy" --sort score --page 2`,
		Args:              cobra.ExactArgs(1),
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
	cmd.Flags().StringVar(&opts.has, "has", "", "Filter by emoji reaction (e.g. :rocket:)")
	cmd.Flags().StringVar(&opts.is, "is", "", "Filter by type: dm, thread, or starred")
	cmd.Flags().StringVar(&opts.during, "during", "", "Filter by time period: today, yesterday, week, or month")
	cmd.Flags().StringVar(&opts.sort, "sort", "timestamp", "Sort order: timestamp or score")
	cmd.Flags().IntVar(&opts.page, "page", 1, "Page number for pagination")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of results per page")
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
	if opts.has != "" {
		emoji := strings.Trim(opts.has, ":")
		query += fmt.Sprintf(" has:%s", emoji)
	}
	if opts.is != "" {
		query += fmt.Sprintf(" is:%s", opts.is)
	}
	if opts.during != "" {
		query += fmt.Sprintf(" during:%s", opts.during)
	}

	params := slack.SearchParameters{
		Sort:          opts.sort,
		SortDirection: "desc",
		Count:         opts.limit,
		Page:          opts.page,
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

	// Pagination footer
	if result.Paging.Pages > 1 {
		fmt.Fprintf(ios.Out, "%s\n",
			cs.Gray(fmt.Sprintf("Page %d of %d (%d total results) — use --page %d to see more",
				result.Paging.Page, result.Paging.Pages, result.Total, result.Paging.Page+1)))
	}

	return nil
}
