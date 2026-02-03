package message

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type sendOptions struct {
	factory  *cmdutil.Factory
	channel  string
	text     string
	threadTS string
	json     cmdutil.JSONFlags
}

// NewCmdSend returns the "message send" command.
func NewCmdSend(f *cmdutil.Factory) *cobra.Command {
	opts := &sendOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "send <channel> [text]",
		Short: "Send a message to a channel",
		Long: `Post a message to a Slack channel, DM, or thread.

The channel argument accepts #channel-name or a channel ID.
If text is omitted, reads from stdin (for piping).`,
		Example: `  # Send a message
  slackbuzz message send #general "Hello, world!"

  # Send to a thread
  slackbuzz message send #general "Reply here" --thread-ts 1234567890.123456

  # Pipe content
  echo "Build passed" | slackbuzz message send #deploys

  # Pipe from a command
  git log --oneline -5 | slackbuzz message send #dev-logs`,
		Args: cobra.RangeArgs(1, 2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.channel = args[0]
			if len(args) > 1 {
				opts.text = args[1]
			}
			return sendRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.threadTS, "thread-ts", "", "Thread timestamp to reply to")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func sendRun(opts *sendOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.Slack)

	// Resolve channel name to ID
	channelID, err := resolver.ResolveChannel(opts.channel)
	if err != nil {
		return err
	}

	// Get message text from args or stdin
	text := opts.text
	if text == "" {
		// Read from stdin
		if !ios.IsTerminal() {
			data, readErr := io.ReadAll(ios.In)
			if readErr != nil {
				return fmt.Errorf("failed to read from stdin: %w", readErr)
			}
			text = strings.TrimSpace(string(data))
		} else {
			// Interactive: read a single line
			scanner := bufio.NewScanner(ios.In)
			fmt.Fprint(ios.Out, "Message: ")
			if scanner.Scan() {
				text = strings.TrimSpace(scanner.Text())
			}
		}
	}

	if text == "" {
		return fmt.Errorf("message text cannot be empty")
	}

	// Build message options
	msgOpts := []slack.MsgOption{
		slack.MsgOptionText(text, false),
	}
	if opts.threadTS != "" {
		msgOpts = append(msgOpts, slack.MsgOptionTS(opts.threadTS))
	}

	// Post the message
	respChannel, respTS, err := client.Slack.PostMessage(channelID, msgOpts...)
	if err != nil {
		return fmt.Errorf("%s", api.FormatError(err))
	}

	if opts.json.WantsJSON() {
		result := map[string]string{
			"channel":   respChannel,
			"timestamp": respTS,
		}
		return opts.json.OutputJSON(ios.Out, result)
	}

	fmt.Fprintf(ios.Out, "%s Message sent to %s (ts: %s)\n",
		cs.Green("✓"), cs.Bold(opts.channel), respTS)
	return nil
}
