package message

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type sendOptions struct {
	factory  *cmdutil.Factory
	channel  string
	text     string
	threadTS string
	asBot    bool
	json     cmdutil.JSONFlags
}

// NewCmdSend returns the "message send" command.
func NewCmdSend(f *cmdutil.Factory) *cobra.Command {
	opts := &sendOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "send <channel|user> [text]",
		Short: "Send a message to a channel or DM",
		Long: `Post a message to a Slack channel, DM, or thread.

The first argument accepts a #channel-name, channel ID, @username, or user ID.
To send a DM, use a username, @username, or user ID as the channel argument.
If text is omitted, reads from stdin (for piping).`,
		Example: `  # Send a message to a channel
  slackbuzz message send #general "Hello, world!"

  # Send a DM by @username
  slackbuzz message send @sarah "Quick question about the API"

  # Send a DM by username (no @ needed)
  slackbuzz message send herman "Hey, got a minute?"

  # Send a DM by user ID
  slackbuzz message send U02P3QC5H24 "Direct message by ID"

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
	cmd.Flags().BoolVar(&opts.asBot, "as-bot", false, "Send as the bot instead of your user account")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func sendRun(opts *sendOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	agentMode := opts.factory.IsAgentMode()

	// In agent mode, prefer bot token (avoids user-token scope gaps);
	// otherwise default to user token so messages post as the user.
	var client *api.Client
	var err error
	if opts.asBot || agentMode {
		client, err = opts.factory.DefaultClient()
	} else {
		client, err = opts.factory.UserClient()
	}
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.Slack)

	// Resolve target to a channel ID — auto-detect DMs vs channels
	var channelID string
	if api.LooksLikeUser(opts.channel) {
		channelID, err = resolver.ResolveDM(opts.channel)
	} else {
		channelID, err = resolver.ResolveChannel(opts.channel)
	}

	// Bot-fallback: if resolution failed with missing_scope on the user token,
	// retry with the bot client (which typically has channels:read).
	if err != nil && api.IsMissingScopeError(err) && !opts.asBot && !agentMode {
		botClient, botErr := opts.factory.BotClient()
		if botErr == nil {
			fmt.Fprintf(ios.ErrOut, "%s User token missing scope for channel lookup — falling back to bot token\n", cs.Yellow("!"))
			botResolver := api.NewResolver(botClient.Slack)
			if api.LooksLikeUser(opts.channel) {
				channelID, err = botResolver.ResolveDM(opts.channel)
			} else {
				channelID, err = botResolver.ResolveChannel(opts.channel)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	// Self-DM: if sending a DM to yourself, switch to bot so you get a notification
	if api.LooksLikeUser(opts.channel) && !opts.asBot {
		if selfID, _, _ := auth.ResolveUserID(); selfID != "" {
			targetID := resolveTargetUserID(resolver, opts.channel)
			if targetID == selfID {
				if botClient, botErr := opts.factory.BotClient(); botErr == nil {
					client = botClient
					fmt.Fprintf(ios.ErrOut, "%s Sending to yourself — using bot so you get a notification\n", cs.Blue("→"))
				}
			}
		}
	}

	// Get message text from args or stdin
	text := opts.text
	if text == "" {
		if agentMode {
			return fmt.Errorf("message text is required (SLACKBUZZ_AGENT=1 disables interactive prompts)")
		}
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

	// Strip common shell escape artifacts (e.g. zsh history expansion turns ! into \!)
	text = unescapeShellArtifacts(text)

	// Resolve @mentions in message body
	if strings.Contains(text, "@") {
		resolved, names, err := resolver.ResolveMentions(text)
		if err == nil {
			text = resolved
			for _, name := range names {
				fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
			}
		}
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
		return fmt.Errorf("%s", api.FormatSendError(err, opts.channel))
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

// resolveTargetUserID resolves a DM target to a user ID without opening a conversation.
// Returns empty string if resolution fails.
func resolveTargetUserID(r *api.Resolver, target string) string {
	id, err := r.ResolveUser(target)
	if err != nil {
		return ""
	}
	return id
}

// unescapeShellArtifacts removes common shell escape sequences that leak into
// CLI arguments. For example, zsh's history expansion escapes ! as \! when
// passed through double-quoted strings.
func unescapeShellArtifacts(text string) string {
	text = strings.ReplaceAll(text, `\!`, "!")
	text = strings.ReplaceAll(text, `\?`, "?")
	return text
}
