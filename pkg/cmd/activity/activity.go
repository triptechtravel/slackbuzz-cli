package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type activityOptions struct {
	factory *cmdutil.Factory
	dms     bool
	threads bool
	all     bool
	since   string
	channel string
	from    string
	limit   int
	json    cmdutil.JSONFlags
}

// NewCmdActivity returns the "activity" command (aliased as "inbox").
func NewCmdActivity(f *cmdutil.Factory) *cobra.Command {
	opts := &activityOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Show what needs your attention — mentions, DMs, threads",
		Long: `Show messages that need your attention: mentions, DMs, and threads.

By default shows messages where you were mentioned. Use flags to filter.
Detects ClickUp task IDs and GitHub PR/issue URLs and shows actionable hints.

Requires a user token (xoxp-) for the search API.`,
		Example: `  # Show mentions (default)
  slackbuzz activity

  # Show DMs
  slackbuzz activity --dms

  # Show threads you're in
  slackbuzz activity --threads

  # Everything from the last day
  slackbuzz activity --all --since 1d

  # Filter by channel or user
  slackbuzz activity --channel #engineering --from @alice

  # Output as JSON
  slackbuzz activity --json`,
		Aliases:           []string{"inbox"},
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return activityRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dms, "dms", false, "Show direct messages")
	cmd.Flags().BoolVar(&opts.threads, "threads", false, "Show threads you're mentioned in")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Show mentions, DMs, and threads combined")
	cmd.Flags().StringVar(&opts.since, "since", "", "Only show messages after this time (2h, 1d, 7d, 2w, or YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.channel, "channel", "", "Filter by channel (e.g. #engineering)")
	cmd.Flags().StringVar(&opts.from, "from", "", "Filter by sender (e.g. @alice)")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of results")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

// activityItem represents a single activity entry for rendering and JSON output.
type activityItem struct {
	Type      string     `json:"type"`
	User      string     `json:"user"`
	Channel   string     `json:"channel"`
	ChannelID string     `json:"channel_id"`
	Text      string     `json:"text"`
	Timestamp string     `json:"timestamp"`
	ThreadTS  string     `json:"thread_ts,omitempty"`
	Permalink string     `json:"permalink,omitempty"`
	Reactions []Reaction `json:"reactions,omitempty"`
	Hints     []Hint     `json:"hints,omitempty"`
}

// Reaction represents a single reaction on a message.
type Reaction struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Emoji string `json:"emoji"`
}

func activityRun(opts *activityOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	userID, _, resolveErr := auth.ResolveUserID()
	if resolveErr != nil {
		return fmt.Errorf("could not determine your Slack user ID: %w", resolveErr)
	}

	teamID, _ := auth.GetTeamInfo()

	client, err := opts.factory.UserClient()
	if err != nil {
		return err
	}

	// Load custom workspace emoji for display (best-effort, non-blocking)
	text.LoadCustomEmoji(client.Token())

	afterDate, err := ParseSince(opts.since)
	if err != nil {
		return err
	}

	var items []activityItem

	if opts.all {
		// Merge mentions + DMs + threads
		mentionItems, err := searchActivity(client, buildQuery(fmt.Sprintf("<@%s>", userID), opts, afterDate), "mention", opts.limit)
		if err != nil {
			return err
		}
		dmItems, err := searchActivity(client, buildQuery("is:dm", opts, afterDate), "DM", opts.limit)
		if err != nil {
			return err
		}
		threadItems, err := searchActivity(client, buildQuery(fmt.Sprintf("<@%s> is:thread", userID), opts, afterDate), "thread", opts.limit)
		if err != nil {
			return err
		}
		items = deduplicateItems(append(append(mentionItems, dmItems...), threadItems...))
	} else if opts.dms {
		items, err = searchActivity(client, buildQuery("is:dm", opts, afterDate), "DM", opts.limit)
	} else if opts.threads {
		items, err = searchActivity(client, buildQuery(fmt.Sprintf("<@%s> is:thread", userID), opts, afterDate), "thread", opts.limit)
	} else {
		// Default: mentions
		items, err = searchActivity(client, buildQuery(fmt.Sprintf("<@%s>", userID), opts, afterDate), "mention", opts.limit)
	}
	if err != nil {
		return err
	}

	if len(items) > opts.limit {
		items = items[:opts.limit]
	}

	if len(items) == 0 {
		fmt.Fprintln(ios.Out, "No activity found.")
		return nil
	}

	// Fetch reactions for each item using bot client (best-effort)
	botClient, botErr := opts.factory.BotClient()
	if botErr == nil {
		enrichReactions(botClient, items)
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, items)
	}

	// Reverse items so most recent is at the bottom (natural terminal reading order)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	// Render TTY output
	for _, item := range items {
		ts := ParseSlackTimestamp(item.Timestamp)
		timeStr := text.RelativeTime(ts)

		typeLabel := "@mention"
		switch item.Type {
		case "DM":
			typeLabel = "DM"
		case "thread":
			typeLabel = "thread"
		}

		channelDisplay := item.Channel
		if item.Type == "DM" {
			channelDisplay = "(DM)"
		} else if channelDisplay != "" {
			channelDisplay = "#" + channelDisplay
		}

		// Build deeplink for "Open in Slack"
		deeplink := text.SlackDeeplink(teamID, item.ChannelID, text.FormatMessageTS(item.Timestamp), "")
		openLink := ""
		if deeplink != "" {
			openLink = text.Hyperlink(deeplink, "Open in Slack")
		}

		fmt.Fprintf(ios.Out, "  %-10s %-12s %-20s %s\n",
			cs.Yellow(typeLabel),
			cs.Bold(item.User),
			cs.Cyan(channelDisplay),
			cs.Gray(timeStr),
		)

		msgText := text.FormatSlackText(item.Text)
		fmt.Fprintf(ios.Out, "             %s\n", msgText)

		// Show reactions inline
		if len(item.Reactions) > 0 {
			var reactionParts []string
			for _, r := range item.Reactions {
				reactionParts = append(reactionParts, fmt.Sprintf("%s %d", r.Emoji, r.Count))
			}
			fmt.Fprintf(ios.Out, "             %s\n", strings.Join(reactionParts, "  "))
		}

		if openLink != "" {
			fmt.Fprintf(ios.Out, "             %s %s\n", cs.Gray("→"), cs.Cyan(openLink))
		}

		for _, hint := range item.Hints {
			fmt.Fprintf(ios.Out, "             %s %s\n", cs.Gray("→"), cs.Green(hint.Command))
		}
		fmt.Fprintln(ios.Out)
	}

	// Hints footer
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz message send <channel> \"text\" --thread-ts <ts>\n", cs.Gray("Reply:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz react <channel> <ts> :emoji:\n", cs.Gray("React:"))
	fmt.Fprintf(ios.Out, "  %s  slackbuzz later add <channel> <ts>\n", cs.Gray("Save:"))

	return nil
}

func buildQuery(base string, opts *activityOptions, afterDate string) string {
	q := base
	if opts.channel != "" {
		ch := strings.TrimPrefix(opts.channel, "#")
		q += fmt.Sprintf(" in:%s", ch)
	}
	if opts.from != "" {
		from := strings.TrimPrefix(opts.from, "@")
		q += fmt.Sprintf(" from:%s", from)
	}
	if afterDate != "" {
		q += fmt.Sprintf(" after:%s", afterDate)
	}
	return q
}

func searchActivity(client *api.Client, query, activityType string, limit int) ([]activityItem, error) {
	resp, err := slackapi.SearchMessages(context.Background(), client.API, &slackapi.SearchMessagesParams{
		Query:   query,
		Sort:    "timestamp",
		SortDir: "desc",
		Count:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
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
			Matches []searchMatch `json:"matches"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(resp.Raw, &searchResult); err != nil {
		return nil, fmt.Errorf("decode search result: %w", err)
	}

	items := make([]activityItem, 0, len(searchResult.Messages.Matches))
	for _, match := range searchResult.Messages.Matches {
		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		item := activityItem{
			Type:      activityType,
			User:      match.Username,
			Channel:   channelName,
			ChannelID: match.Channel.ID,
			Text:      match.Text,
			Timestamp: match.TS,
			Permalink: match.Permalink,
			Hints:     ExtractHints(match.Text),
		}
		items = append(items, item)
	}

	return items, nil
}

func deduplicateItems(items []activityItem) []activityItem {
	seen := make(map[string]bool, len(items))
	out := make([]activityItem, 0, len(items))
	for _, item := range items {
		key := item.Timestamp + "|" + item.Channel
		if !seen[key] {
			seen[key] = true
			out = append(out, item)
		}
	}
	return out
}

// ParseSince converts duration shorthand (2h, 1d, 7d, 2w) or YYYY-MM-DD to
// a date string suitable for Slack's after: search modifier.
func ParseSince(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	// Try YYYY-MM-DD first
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return s, nil
	}

	// Parse duration shorthand
	d, err := ParseDuration(s)
	if err != nil {
		return "", fmt.Errorf("invalid --since value %q (use: 2h, 1d, 7d, 2w, or YYYY-MM-DD)", s)
	}

	t := time.Now().Add(-d)
	return t.Format("2006-01-02"), nil
}

var durationRe = regexp.MustCompile(`^(\d+)([hdwm])$`)

// ParseDuration parses shorthand like "2h", "1d", "7d", "2w" into time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	n, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "h":
		return time.Duration(n) * time.Hour, nil
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", m[2])
	}
}

// ParseSlackTimestamp converts a Slack timestamp to time.Time.
func ParseSlackTimestamp(ts string) time.Time {
	var sec, usec int64
	fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	return time.Unix(sec, usec*1000)
}

// enrichReactions fetches reactions for each activity item using the bot client.
// Failures are silently ignored per-item.
func enrichReactions(botClient *api.Client, items []activityItem) {
	ctx := context.Background()
	for i := range items {
		if items[i].ChannelID == "" || items[i].Timestamp == "" {
			continue
		}
		resp, err := slackapi.ReactionsGet(ctx, botClient.API, &slackapi.ReactionsGetParams{
			Channel:   items[i].ChannelID,
			Timestamp: items[i].Timestamp,
		})
		if err != nil {
			continue
		}
		var rg struct {
			Message struct {
				Reactions []struct {
					Name  string `json:"name"`
					Count int    `json:"count"`
				} `json:"reactions"`
			} `json:"message"`
		}
		if err := json.Unmarshal(resp.Raw, &rg); err != nil {
			continue
		}
		for _, r := range rg.Message.Reactions {
			items[i].Reactions = append(items[i].Reactions, Reaction{
				Name:  r.Name,
				Count: r.Count,
				Emoji: text.EmojiToUnicode(r.Name),
			})
		}
	}
}
