package file

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	slacktext "github.com/triptechtravel/slackbuzz-cli/internal/text"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type uploadOptions struct {
	factory   *cmdutil.Factory
	filePaths []string
	channel   string
	title     string
	comment   string
	threadTS  string
	raw       bool
	json      cmdutil.JSONFlags
}

// NewCmdUpload returns the "file upload" command.
func NewCmdUpload(f *cmdutil.Factory) *cobra.Command {
	opts := &uploadOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "upload <file-path>... <channel|user>",
		Short: "Upload files to a channel or DM",
		Long: `Upload one or more files to a Slack channel or direct message.

The last argument is the channel or user target. All preceding arguments are
file paths to upload. All the files are shared together as ONE message, with
--comment as its text (formatted like message send: Markdown to mrkdwn and
@mentions; --raw keeps it as typed).

To post a message with files attached, message send --file does the same.

The channel argument accepts #channel-name, channel ID, @username, or user ID.`,
		Example: `  # Upload a single file to a channel
  slackbuzz file upload report.pdf #general

  # Upload multiple files
  slackbuzz file upload chart1.png chart2.png chart3.png #analytics

  # Upload with title and comment
  slackbuzz file upload data.csv #sales --title "Q4 Revenue" --comment "Updated data"

  # Upload to a DM
  slackbuzz file upload notes.txt @alice

  # Upload to a thread
  slackbuzz file upload results.png #dev --thread-ts 1706000000.000000`,
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Last arg is channel/user, rest are file paths
			opts.channel = args[len(args)-1]
			opts.filePaths = args[:len(args)-1]
			return uploadRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.title, "title", "", "File title in Slack (applies to first file)")
	cmd.Flags().StringVar(&opts.comment, "comment", "", "Text of the message the file(s) are posted in")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Disable Markdown→mrkdwn conversion of --comment")
	cmd.Flags().StringVar(&opts.threadTS, "thread-ts", "", "Thread timestamp to upload into")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

type uploadResult struct {
	FileID   string `json:"file_id"`
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Channel  string `json:"channel"`
}

func uploadRun(opts *uploadOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Validate all files exist before starting uploads
	if err := cmdutil.CheckUploadFiles(opts.filePaths); err != nil {
		return err
	}

	// File uploads need files:write scope. Prefer user token so uploads
	// appear from the user rather than the bot app.
	client, err := opts.factory.UserClient()
	if err != nil {
		client, err = opts.factory.BotClient()
		if err != nil {
			return err
		}
	}

	// Resolve channel/DM — ResolveTarget handles bare names by trying
	// user (DM) first, then falling back to channel.
	resolver := api.NewResolver(client.API)
	channelID, _, err := resolver.ResolveTarget(opts.channel)

	// If resolution fails with missing scope, try the other token
	if err != nil && api.IsMissingScopeError(err) {
		altClient, altErr := opts.factory.UserClient()
		if altErr != nil {
			altClient, altErr = opts.factory.BotClient()
		}
		if altErr == nil {
			fmt.Fprintf(ios.ErrOut, "%s Retrying with alternate token\n", cs.Yellow("!"))
			altResolver := api.NewResolver(altClient.API)
			channelID, _, err = altResolver.ResolveTarget(opts.channel)
			if err == nil {
				client = altClient
			}
		}
	}

	if err != nil {
		return fmt.Errorf("%s", api.FormatResolveError(err, opts.channel))
	}

	// Same text pipeline as message send: shell unescape, Markdown to mrkdwn,
	// and @mention resolution.
	comment := opts.comment
	if comment != "" {
		var hints []slacktext.FormatHint
		comment, hints = slacktext.NormalizeOutgoing(comment, opts.raw)
		cmdutil.PrintFormatHints(ios, hints)
		if strings.Contains(comment, "@") {
			if resolved, names, err := resolver.ResolveMentions(comment); err == nil {
				comment = resolved
				for _, name := range names {
					fmt.Fprintf(ios.ErrOut, "Mentioning %s\n", cs.Bold("@"+name))
				}
			}
		}
	}

	specs := cmdutil.UploadSpecs(opts.filePaths)
	if opts.title != "" {
		specs[0].Title = opts.title
	}
	resp, err := slackapi.UploadFiles(context.Background(), client.API, &slackapi.UploadFilesParams{
		Files:          specs,
		Channels:       channelID,
		InitialComment: comment,
		ThreadTS:       opts.threadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to upload: %s", api.FormatError(err))
	}

	results := make([]uploadResult, len(resp.Files))
	for i, f := range resp.Files {
		results[i] = uploadResult{FileID: f.ID, Title: f.Title, Filename: filepath.Base(opts.filePaths[i]), Channel: channelID}
		if !opts.json.WantsJSON() {
			fmt.Fprintf(ios.Out, "%s Uploaded %s to %s\n", cs.Green("✓"), cs.Bold(results[i].Filename), cs.Bold(opts.channel))
		}
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, results)
	}

	if len(results) > 1 {
		fmt.Fprintf(ios.Out, "\n%s %d files posted to %s in one message\n",
			cs.Green("✓"), len(results), cs.Bold(opts.channel))
	}

	return nil
}
