package file

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type searchOptions struct {
	factory  *cmdutil.Factory
	query    string
	channel  string
	from     string
	fileType string
	limit    int
	page     int
	json     cmdutil.JSONFlags
}

// NewCmdSearch returns the "file search" command.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	opts := &searchOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search files (requires user token)",
		Long: `Search files shared in Slack using the search.files API.

This command requires a user token (xoxp-) with search:read scope.`,
		Example: `  # Search for files
  slackbuzz file search "design spec"

  # Filter by file type
  slackbuzz file search "report" --type pdf

  # Filter by channel
  slackbuzz file search "report" --channel #engineering

  # Filter by uploader
  slackbuzz file search "diagram" --from @alice

  # Pagination
  slackbuzz file search "report" --page 2`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.query = args[0]
			return searchRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.channel, "channel", "", "Filter by channel")
	cmd.Flags().StringVar(&opts.from, "from", "", "Filter by uploader")
	cmd.Flags().StringVar(&opts.fileType, "type", "", "Filter by file type (e.g. pdf, png, zip)")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of results per page")
	cmd.Flags().IntVar(&opts.page, "page", 1, "Page number for pagination")
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
	if opts.fileType != "" {
		query += fmt.Sprintf(" type:%s", opts.fileType)
	}

	params := slack.SearchParameters{
		Sort:          "timestamp",
		SortDirection: "desc",
		Count:         opts.limit,
		Page:          opts.page,
	}

	result, err := client.Slack.SearchFiles(query, params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if result.Total == 0 {
		fmt.Fprintln(ios.Out, "No files found.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, result.Matches)
	}

	fmt.Fprintf(ios.Out, "%s results for %q\n\n", cs.Bold(fmt.Sprintf("%d", result.Total)), opts.query)

	for _, f := range result.Matches {
		title := f.Title
		if title == "" {
			title = f.Name
		}

		ts := time.Unix(int64(f.Timestamp), 0)
		timeStr := text.RelativeTime(ts)

		channelDisplay := ""
		if len(f.Channels) > 0 {
			channelDisplay = "#" + f.Channels[0]
		}

		permalink := f.Permalink

		fmt.Fprintf(ios.Out, "  %-40s  %-8s  @%s\n",
			cs.Bold(title),
			cs.Gray(f.PrettyType),
			cs.Cyan(f.User),
		)
		if channelDisplay != "" {
			fmt.Fprintf(ios.Out, "  %-40s  %s\n",
				cs.Gray(channelDisplay),
				cs.Gray(timeStr),
			)
		} else {
			fmt.Fprintf(ios.Out, "  %s\n", cs.Gray(timeStr))
		}
		if permalink != "" {
			fmt.Fprintf(ios.Out, "  %s\n", cs.Cyan(permalink))
		}
		fmt.Fprintln(ios.Out)
	}

	// Pagination footer
	if result.Paging.Pages > 1 {
		fmt.Fprintf(ios.Out, "%s\n",
			cs.Gray(fmt.Sprintf("Page %d of %d (%d total results) — use --page %d to see more",
				result.Paging.Page, result.Paging.Pages, result.Total, result.Paging.Page+1)))
	}

	return nil
}
