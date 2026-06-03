package digest

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type digestOptions struct {
	factory     *cmdutil.Factory
	since       string
	slackOnly   bool
	clickupOnly bool
	githubOnly  bool
	json        cmdutil.JSONFlags
}

// NewCmdDigest returns the "digest" command.
func NewCmdDigest(f *cmdutil.Factory) *cobra.Command {
	opts := &digestOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Cross-tool morning briefing (Slack + ClickUp + GitHub)",
		Long: `Show a combined briefing of what needs your attention across Slack,
ClickUp, and GitHub.

Slack data comes from the search API (requires user token).
ClickUp and GitHub data come from their respective CLIs if installed.
Sections are gracefully skipped if the CLI isn't available.`,
		Example: `  # Full briefing
  slackbuzz digest

  # Only Slack
  slackbuzz digest --slack-only

  # Only GitHub
  slackbuzz digest --github-only

  # Since yesterday
  slackbuzz digest --since 1d

  # Output as JSON
  slackbuzz digest --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return digestRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.since, "since", "1d", "Time window for activity (2h, 1d, 7d, 2w, or YYYY-MM-DD)")
	cmd.Flags().BoolVar(&opts.slackOnly, "slack-only", false, "Only show Slack activity")
	cmd.Flags().BoolVar(&opts.clickupOnly, "clickup-only", false, "Only show ClickUp tasks")
	cmd.Flags().BoolVar(&opts.githubOnly, "github-only", false, "Only show GitHub PRs")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

type digestData struct {
	Slack   []slackItem   `json:"slack,omitempty"`
	ClickUp []clickupTask `json:"clickup,omitempty"`
	GitHub  []githubPR    `json:"github,omitempty"`
}

type slackItem struct {
	Type    string `json:"type"`
	User    string `json:"user"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
	Time    string `json:"time"`
}

type clickupTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Name   string `json:"name"`
}

type githubPR struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func digestRun(opts *digestOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	data := digestData{}
	showAll := !opts.slackOnly && !opts.clickupOnly && !opts.githubOnly

	// --- Slack ---
	if showAll || opts.slackOnly {
		items, err := fetchSlackActivity(opts)
		if err != nil {
			fmt.Fprintf(ios.ErrOut, "Warning: Slack activity fetch failed: %v\n", err)
		} else {
			data.Slack = items
		}
	}

	// --- ClickUp ---
	if showAll || opts.clickupOnly {
		tasks, err := fetchClickUpTasks()
		if err == nil {
			data.ClickUp = tasks
		}
		// Silently skip if clickup CLI not found
	}

	// --- GitHub ---
	if showAll || opts.githubOnly {
		prs, err := fetchGitHubPRs()
		if err == nil {
			data.GitHub = prs
		}
		// Silently skip if gh CLI not found
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, data)
	}

	// Render TTY output
	hasOutput := false

	if len(data.Slack) > 0 {
		hasOutput = true
		fmt.Fprintf(ios.Out, "%s — %s\n",
			cs.Bold("SLACK"),
			cs.Gray(text.Pluralize(len(data.Slack), "unread mention")),
		)
		for _, item := range data.Slack {
			channelDisplay := item.Channel
			if item.Type == "DM" {
				channelDisplay = "DM"
			} else {
				channelDisplay = "#" + channelDisplay
			}

			msgText := text.FormatSlackText(item.Text)

			fmt.Fprintf(ios.Out, "  @%-12s %-18s %q  %s\n",
				cs.Bold(item.User),
				cs.Cyan(channelDisplay),
				msgText,
				cs.Gray(item.Time),
			)
		}
		fmt.Fprintln(ios.Out)
	} else if showAll || opts.slackOnly {
		fmt.Fprintf(ios.Out, "%s — no recent mentions\n\n", cs.Bold("SLACK"))
	}

	if len(data.ClickUp) > 0 {
		hasOutput = true
		fmt.Fprintf(ios.Out, "%s — %s\n",
			cs.Bold("CLICKUP"),
			cs.Gray(text.Pluralize(len(data.ClickUp), "task assigned to you")),
		)
		for _, task := range data.ClickUp {
			fmt.Fprintf(ios.Out, "  %-12s %-14s %s\n",
				cs.Yellow(task.ID),
				cs.Gray(task.Status),
				task.Name,
			)
		}
		fmt.Fprintln(ios.Out)
	} else if showAll || opts.clickupOnly {
		if _, err := exec.LookPath("clickup"); err != nil {
			fmt.Fprintf(ios.Out, "%s — CLI not installed (skipped)\n\n", cs.Bold("CLICKUP"))
		} else {
			fmt.Fprintf(ios.Out, "%s — no tasks assigned\n\n", cs.Bold("CLICKUP"))
		}
	}

	if len(data.GitHub) > 0 {
		hasOutput = true
		fmt.Fprintf(ios.Out, "%s — %s\n",
			cs.Bold("GITHUB"),
			cs.Gray(text.Pluralize(len(data.GitHub), "PR needs your review")),
		)
		for _, pr := range data.GitHub {
			fmt.Fprintf(ios.Out, "  #%-6d %-40s %s  %s\n",
				pr.Number,
				pr.Title,
				cs.Bold(pr.Author),
				cs.Green(fmt.Sprintf("+%d -%d", pr.Additions, pr.Deletions)),
			)
		}
		fmt.Fprintln(ios.Out)
	} else if showAll || opts.githubOnly {
		if _, err := exec.LookPath("gh"); err != nil {
			fmt.Fprintf(ios.Out, "%s — CLI not installed (skipped)\n\n", cs.Bold("GITHUB"))
		} else {
			fmt.Fprintf(ios.Out, "%s — no PRs need review\n\n", cs.Bold("GITHUB"))
		}
	}

	if hasOutput {
		fmt.Fprintln(ios.Out, cs.Gray("---"))
		fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
		fmt.Fprintf(ios.Out, "  slackbuzz message send <channel> \"text\"\n")
		if len(data.ClickUp) > 0 {
			fmt.Fprintf(ios.Out, "  clickup task update %s --status complete\n", data.ClickUp[0].ID)
		}
		if len(data.GitHub) > 0 {
			fmt.Fprintf(ios.Out, "  gh pr review %d --approve\n", data.GitHub[0].Number)
		}
	}

	return nil
}

func fetchSlackActivity(opts *digestOptions) ([]slackItem, error) {
	userID, _, resolveErr := auth.ResolveUserID()
	if resolveErr != nil {
		return nil, fmt.Errorf("could not determine user ID: %w", resolveErr)
	}

	client, err := opts.factory.UserClient()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("<@%s>", userID)

	if opts.since != "" {
		afterDate, parseErr := activity.ParseSince(opts.since)
		if parseErr != nil {
			return nil, parseErr
		}
		if afterDate != "" {
			query += fmt.Sprintf(" after:%s", afterDate)
		}
	}

	resp, err := slackapi.SearchMessages(context.Background(), client.API, &slackapi.SearchMessagesParams{
		Query:   query,
		Sort:    "timestamp",
		SortDir: "desc",
		Count:   5,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	type searchMatch struct {
		Username string `json:"username"`
		Text     string `json:"text"`
		TS       string `json:"ts"`
		Channel  struct {
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

	items := make([]slackItem, 0, len(searchResult.Messages.Matches))
	for _, match := range searchResult.Messages.Matches {
		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		ts := activity.ParseSlackTimestamp(match.TS)
		timeStr := text.RelativeTime(ts)

		items = append(items, slackItem{
			Type:    "mention",
			User:    match.Username,
			Channel: channelName,
			Text:    match.Text,
			Time:    timeStr,
		})
	}

	return items, nil
}

func fetchClickUpTasks() ([]clickupTask, error) {
	path, err := exec.LookPath("clickup")
	if err != nil {
		return nil, fmt.Errorf("clickup CLI not found")
	}

	out, err := exec.Command(path, "task", "list", "--assignee", "me", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("clickup task list failed: %w", err)
	}

	var tasks []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status struct {
			Status string `json:"status"`
		} `json:"status"`
	}
	if err := json.Unmarshal(out, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse clickup output: %w", err)
	}

	result := make([]clickupTask, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, clickupTask{
			ID:     t.ID,
			Status: t.Status.Status,
			Name:   t.Name,
		})
	}

	return result, nil
}

func fetchGitHubPRs() ([]githubPR, error) {
	path, err := exec.LookPath("gh")
	if err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}

	out, err := exec.Command(path, "pr", "list", "--review-requested=@me", "--json", "number,title,author,additions,deletions").Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}

	var prs []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	result := make([]githubPR, 0, len(prs))
	for _, pr := range prs {
		result = append(result, githubPR{
			Number:    pr.Number,
			Title:     pr.Title,
			Author:    pr.Author.Login,
			Additions: pr.Additions,
			Deletions: pr.Deletions,
		})
	}

	return result, nil
}
