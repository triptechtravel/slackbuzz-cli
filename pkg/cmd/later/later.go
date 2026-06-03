package later

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdLater returns the "later" command group.
func NewCmdLater(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "later <command>",
		Short: "Saved/bookmarked items",
		Long: `Manage your saved (starred) items in Slack.

Save messages from activity for follow-up. Uses the stars API.
Requires a user token (xoxp-) with stars:read and stars:write scopes.`,
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdAdd(f))
	cmd.AddCommand(newCmdRemove(f))

	return cmd
}

// --- list ---

type listOptions struct {
	factory *cmdutil.Factory
	limit   int
	json    cmdutil.JSONFlags
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show saved items",
		Long:  "List your starred/saved items in Slack.",
		Example: `  # Show saved items
  slackbuzz later list

  # Output as JSON
  slackbuzz later list --json`,
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of items")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

type savedItem struct {
	Type      string `json:"type"`
	User      string `json:"user,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Text      string `json:"text,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	// Use raw HTTP because slack-go's StarredItems type may not expose all fields cleanly
	items, err := fetchStars(token, opts.limit)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Fprintln(ios.Out, "No saved items.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, items)
	}

	// Reverse so most recent is at the bottom (natural terminal reading order)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	for i, item := range items {
		ts := activity.ParseSlackTimestamp(item.Timestamp)
		timeStr := text.RelativeTime(ts)

		msgText := text.FormatSlackText(item.Text)

		channelDisplay := item.Channel
		if channelDisplay != "" {
			channelDisplay = "#" + channelDisplay
		}

		fmt.Fprintf(ios.Out, "  %d.  @%s in %-18s %s\n",
			i+1,
			cs.Bold(item.User),
			cs.Cyan(channelDisplay),
			cs.Gray(timeStr),
		)
		fmt.Fprintf(ios.Out, "      %q\n\n", msgText)
	}

	fmt.Fprintf(ios.Out, "%s\n", text.Pluralize(len(items), "saved item"))

	return nil
}

// fetchStars calls stars.list via HTTP with the user token.
func fetchStars(token string, limit int) ([]savedItem, error) {
	v := url.Values{}
	v.Set("token", token)
	v.Set("count", fmt.Sprintf("%d", limit))

	resp, err := http.PostForm("https://slack.com/api/stars.list", v)
	if err != nil {
		return nil, fmt.Errorf("stars.list failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result starsListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("stars.list error: %s", result.Error)
	}

	items := make([]savedItem, 0, len(result.Items))
	for _, star := range result.Items {
		if star.Type != "message" {
			continue
		}
		items = append(items, savedItem{
			Type:      star.Type,
			User:      star.Message.Username,
			Channel:   star.Channel,
			Text:      star.Message.Text,
			Timestamp: star.Message.Timestamp,
		})
	}

	return items, nil
}

type starsListResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Items []struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		Message struct {
			Text      string `json:"text"`
			Username  string `json:"username"`
			Timestamp string `json:"ts"`
		} `json:"message"`
	} `json:"items"`
}

// --- add ---

type addOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
}

func newCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{factory: f}

	cmd := &cobra.Command{
		Use:               "add <channel> <timestamp>",
		Short:             "Save a message for later",
		Long:              "Star/bookmark a message by channel and timestamp.",
		Example:           `  slackbuzz later add #general 1706000000.000000`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			return addRun(opts)
		},
	}

	return cmd
}

func addRun(opts *addOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.API)
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	if err := starAction(token, "stars.add", channelID, opts.ts); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Saved message %s in %s\n",
		cs.Green("✓"), opts.ts, opts.channel)
	return nil
}

// --- remove ---

type removeOptions struct {
	factory *cmdutil.Factory
	channel string
	ts      string
}

func newCmdRemove(f *cmdutil.Factory) *cobra.Command {
	opts := &removeOptions{factory: f}

	cmd := &cobra.Command{
		Use:               "remove <channel> <timestamp>",
		Short:             "Unsave a message",
		Long:              "Remove a star/bookmark from a message.",
		Example:           `  slackbuzz later remove #general 1706000000.000000`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			opts.ts = args[1]
			return removeRun(opts)
		},
	}

	return cmd
}

func removeRun(opts *removeOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.API)
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	if err := starAction(token, "stars.remove", channelID, opts.ts); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Removed saved message %s in %s\n",
		cs.Green("✓"), opts.ts, opts.channel)
	return nil
}

// starAction calls stars.add or stars.remove via HTTP.
func starAction(token, method, channel, timestamp string) error {
	v := url.Values{}
	v.Set("token", token)
	v.Set("channel", channel)
	v.Set("timestamp", timestamp)

	apiURL := fmt.Sprintf("https://slack.com/api/%s", method)
	resp, err := http.PostForm(apiURL, v)
	if err != nil {
		return fmt.Errorf("%s failed: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("%s error: %s", method, result.Error)
	}

	return nil
}
