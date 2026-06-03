package thread

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type linkOptions struct {
	factory *cmdutil.Factory
	url     string
	taskID  string
	json    cmdutil.JSONFlags
}

// NewCmdLink returns the "thread link" command.
func NewCmdLink(f *cmdutil.Factory) *cobra.Command {
	opts := &linkOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "link <slack-thread-url> <task-id>",
		Short: "Link a Slack thread to a ClickUp task",
		Long: `Parse a Slack thread URL, extract the channel and timestamp,
and post a ClickUp task link as a reply to that thread.

The Slack URL format: https://workspace.slack.com/archives/CHANNEL_ID/pTIMESTAMP`,
		Example:           `  slackbuzz thread link https://myteam.slack.com/archives/C012345/p1706000000000000 CU-abc123`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.url = args[0]
			opts.taskID = args[1]
			return linkRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

// slackURLPattern matches Slack archive URLs like:
// https://workspace.slack.com/archives/C012345/p1706000000000000
var slackURLPattern = regexp.MustCompile(`/archives/([A-Z0-9]+)/p(\d+)`)

func linkRun(opts *linkOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Parse the Slack URL
	matches := slackURLPattern.FindStringSubmatch(opts.url)
	if matches == nil {
		return fmt.Errorf("invalid Slack thread URL. Expected format: https://workspace.slack.com/archives/CHANNEL_ID/pTIMESTAMP")
	}

	channelID := matches[1]
	// Convert Slack URL timestamp (no dot) to API format (with dot)
	rawTS := matches[2]
	threadTS := formatTimestamp(rawTS)

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	// Build the ClickUp task link
	taskURL := fmt.Sprintf("https://app.clickup.com/t/%s", opts.taskID)
	messageText := fmt.Sprintf("📋 Linked to ClickUp task: <%s|%s>", taskURL, opts.taskID)

	// Post as a thread reply
	resp, err := slackapi.ChatPostMessage(context.Background(), client.API, &slackapi.ChatPostMessageParams{
		Channel:  channelID,
		Text:     messageText,
		ThreadTS: threadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to post thread reply: %w", err)
	}
	var posted struct {
		Channel string `json:"channel"`
		TS      string `json:"ts"`
	}
	_ = json.Unmarshal(resp.Raw, &posted)
	respChannel := posted.Channel
	respTS := posted.TS

	if opts.json.WantsJSON() {
		result := map[string]string{
			"channel":   respChannel,
			"timestamp": respTS,
			"thread_ts": threadTS,
			"task_id":   opts.taskID,
			"task_url":  taskURL,
		}
		return opts.json.OutputJSON(ios.Out, result)
	}

	fmt.Fprintf(ios.Out, "%s Linked thread in %s to ClickUp task %s\n",
		cs.Green("✓"), cs.Bold(channelID), cs.Bold(opts.taskID))
	return nil
}

// formatTimestamp converts "1706000000000000" to "1706000000.000000"
func formatTimestamp(raw string) string {
	if strings.Contains(raw, ".") {
		return raw
	}
	if len(raw) <= 10 {
		return raw + ".000000"
	}
	return raw[:10] + "." + raw[10:]
}
