package message

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/recent"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type sendOptions struct {
	factory  *cmdutil.Factory
	channel  string
	text     string
	threadTS string
	asBot    bool
	blocks   bool
	raw      bool
	files    []string
	title    string // the first file's title (file upload --title)
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
If text is omitted, reads from stdin (for piping).

Formatting: Standard Markdown (**bold**, # headers, [links](url)) is
automatically converted to Slack mrkdwn (*bold*, etc). Use --raw to
disable auto-conversion and send text as-is.

Use --blocks to send via Block Kit for richer formatting (sections with
mrkdwn rendering). Long messages are automatically split into multiple
blocks (Slack's 3000-char limit per block).`,
		Example: `  # Send a message to a channel
  slackbuzz message send #general "Hello, world!"

  # Send a DM by @username
  slackbuzz message send @sarah "Quick question about the API"

  # Send with Slack mrkdwn formatting (auto-converted from Markdown)
  slackbuzz message send #releases "## 🚀 v5.4.0\n- **New feature**: offline images"

  # Send via Block Kit for richer rendering
  slackbuzz message send #releases --blocks "## Release Notes\n- *Feature*: offline images"

  # Send raw text without Markdown→mrkdwn conversion
  slackbuzz message send #dev --raw "**this stays as double asterisks**"

  # Send to a thread
  slackbuzz message send #general "Reply here" --thread-ts 1234567890.123456

  # Attach images (one message: the text with the files under it)
  slackbuzz message send #released "New release" --file before.png --file after.png

  # Pipe content
  echo "Build passed" | slackbuzz message send #deploys

  # Pipe from a command
  git log --oneline -5 | slackbuzz message send #dev-logs`,
		Args:              cobra.RangeArgs(1, 2),
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
	cmd.Flags().BoolVar(&opts.blocks, "blocks", false, "Send via Block Kit sections (richer mrkdwn rendering)")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Disable Markdown→mrkdwn auto-conversion")
	cmd.Flags().StringArrayVar(&opts.files, "file", nil, "Attach a file to the message (repeatable; all go in the one message)")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func sendRun(opts *sendOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	if len(opts.files) > 0 && opts.blocks {
		return fmt.Errorf("--file can't be combined with --blocks: Slack posts a file's message as plain mrkdwn")
	}
	if err := cmdutil.CheckUploadFiles(opts.files); err != nil {
		return err
	}

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

	resolver := api.NewResolver(client.API)

	// Resolve target to a channel ID — auto-detect DMs vs channels.
	// ResolveTarget handles bare names by trying user (DM) first, then channel.
	channelID, _, err := resolver.ResolveTarget(opts.channel)

	// Bot-fallback: if resolution failed with missing_scope on the user token,
	// retry with the bot client (which typically has channels:read).
	if err != nil && api.IsMissingScopeError(err) && !opts.asBot && !agentMode {
		botClient, botErr := opts.factory.BotClient()
		if botErr == nil {
			fmt.Fprintf(ios.ErrOut, "%s User token missing scope for channel lookup — falling back to bot token\n", cs.Yellow("!"))
			botResolver := api.NewResolver(botClient.API)
			channelID, _, err = botResolver.ResolveTarget(opts.channel)
		}
	}

	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	// Self-DM: if sending a DM to yourself, switch to bot so you get a notification.
	// The bot opens its own DM channel with you so the message appears from the bot.
	if api.LooksLikeUser(opts.channel) && !opts.asBot && !agentMode {
		targetID := resolveTargetUserID(resolver, opts.channel)
		selfID := client.AuthUserID()
		if targetID != "" && selfID != "" && targetID == selfID {
			if botClient, botErr := opts.factory.BotClient(); botErr == nil {
				botResolver := api.NewResolver(botClient.API)
				if botChannelID, botErr := botResolver.ResolveDM(opts.channel); botErr == nil {
					client = botClient
					channelID = botChannelID
					fmt.Fprintf(ios.ErrOut, "%s Sending to yourself — using bot so you get a notification\n", cs.Blue("→"))
				}
			}
		}
	}

	// Get message text from args or stdin. With files the text is optional
	// (and stdin isn't read): the files can go alone.
	text := opts.text
	if text == "" && len(opts.files) == 0 {
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

	if text == "" && len(opts.files) == 0 {
		return fmt.Errorf("message text cannot be empty")
	}

	// Shell unescape, Markdown→mrkdwn (unless --raw), @mentions.
	if text != "" {
		text = cmdutil.PrepareOutgoing(ios, resolver, text, opts.raw)
	}

	// With files, the text is the files' message: one files.completeUploadExternal.
	if len(opts.files) > 0 {
		return sendFiles(opts, client, channelID, text)
	}

	// Build chat.postMessage params.
	postParams := &slackapi.ChatPostMessageParams{
		Channel:  channelID,
		Text:     text, // also serves as fallback for notifications/previews when blocks are set
		ThreadTS: opts.threadTS,
	}
	if opts.blocks {
		postParams.Blocks = slackapi.BlocksJSON(buildMrkdwnBlocks(text))
	}

	// Post the message
	resp, err := slackapi.ChatPostMessage(context.Background(), client.API, postParams)
	if err != nil {
		return fmt.Errorf("%s", api.FormatSendError(err, opts.channel))
	}
	respChannel := resp.Channel
	respTS := resp.TS

	// Record the target for recent-context defaults (slackbuzz dm / send).
	_ = recent.Push(recentSlot(opts.channel), opts.channel)

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

// sendFiles posts text and files as one message, then reports the message's
// timestamp like a plain send (Slack fills in where a file was shared a
// moment after the upload completes, so it's read back from files.info).
func sendFiles(opts *sendOptions, client *api.Client, channelID, text string) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()
	specs := make([]slackapi.UploadFileSpec, len(opts.files))
	for i, fp := range opts.files {
		specs[i] = slackapi.UploadFileSpec{FilePath: fp}
	}
	specs[0].Title = opts.title

	ctx := context.Background()
	uploaded, err := slackapi.UploadFiles(ctx, client.API, &slackapi.UploadFilesParams{
		Files:          specs,
		ChannelID:      channelID,
		InitialComment: text,
		ThreadTS:       opts.threadTS,
	})
	if err != nil {
		return fmt.Errorf("%s", api.FormatSendError(err, opts.channel))
	}
	_ = recent.Push(recentSlot(opts.channel), opts.channel)
	ts := sharedTS(ctx, client.API, uploaded.Files[0].ID, channelID)

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, map[string]any{"channel": channelID, "timestamp": ts, "files": uploaded.Files})
	}
	for _, f := range uploaded.Files {
		fmt.Fprintf(ios.Out, "%s Attached %s\n", cs.Green("✓"), cs.Bold(f.Name))
	}
	fmt.Fprintf(ios.Out, "%s Message with %d file(s) sent to %s (ts: %s)\n", cs.Green("✓"), len(uploaded.Files), cs.Bold(opts.channel), ts)
	return nil
}

// sharedTS is the timestamp of the message a file was shared in, polled from
// files.info for a few seconds; "" if Slack hasn't reported it yet.
func sharedTS(ctx context.Context, c *slackapi.Client, fileID, channelID string) string {
	for attempt := 0; attempt < 10; attempt++ {
		if info, err := slackapi.FilesInfo(ctx, c, &slackapi.FilesInfoParams{File: fileID}); err == nil && info.File != nil {
			if ts := shareTS(info.File.Shares, channelID); ts != "" {
				return ts
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return ""
}

// shareTS reads a file's shares ({"public"|"private": {channel: [{ts}]}}).
func shareTS(shares map[string]any, channelID string) string {
	for _, visibility := range []string{"public", "private"} {
		byChannel, _ := shares[visibility].(map[string]any)
		list, _ := byChannel[channelID].([]any)
		if len(list) == 0 {
			continue
		}
		if share, _ := list[0].(map[string]any); share != nil {
			if ts, _ := share["ts"].(string); ts != "" {
				return ts
			}
		}
	}
	return ""
}

// recentSlot is the recent-context slot a target is remembered under.
func recentSlot(target string) string {
	if api.LooksLikeUser(target) {
		return "dm"
	}
	return "send"
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

// maxBlockTextLen is Slack's limit for text in a single section block.
const maxBlockTextLen = 3000

// buildMrkdwnBlocks splits message text into Block Kit section blocks with
// mrkdwn type. Long messages are split at paragraph boundaries to stay within
// Slack's 3000-character-per-block limit.
func buildMrkdwnBlocks(text string) []slackapi.Block {
	chunks := splitForBlocks(text, maxBlockTextLen)
	blocks := make([]slackapi.Block, 0, len(chunks))
	for _, chunk := range chunks {
		body := slackapi.NewMrkdwnText(chunk)
		blocks = append(blocks, slackapi.NewSectionBlock(body, nil, nil))
	}
	return blocks
}

// splitForBlocks splits text into chunks that fit within maxLen, preferring
// to split at double-newline (paragraph) boundaries, then single newlines.
func splitForBlocks(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > maxLen {
		// Try to split at a paragraph boundary (double newline)
		var splitAt int
		if idx := strings.LastIndex(remaining[:maxLen], "\n\n"); idx > 0 {
			splitAt = idx
		} else if idx := strings.LastIndex(remaining[:maxLen], "\n"); idx > 0 {
			// Fall back to single newline
			splitAt = idx
		} else {
			// Last resort: hard split at maxLen
			splitAt = maxLen
		}

		chunks = append(chunks, strings.TrimSpace(remaining[:splitAt]))
		remaining = strings.TrimSpace(remaining[splitAt:])
	}

	if remaining != "" {
		chunks = append(chunks, remaining)
	}

	return chunks
}

// PostFiles posts files, with optional text, as one message through the send
// path (token choice, agent mode, self-DM, formatting): file upload's backend.
func PostFiles(f *cmdutil.Factory, target string, paths []string, text, title, threadTS string, raw bool, json cmdutil.JSONFlags) error {
	return sendRun(&sendOptions{factory: f, channel: target, text: text, files: paths, title: title, threadTS: threadTS, raw: raw, json: json})
}
