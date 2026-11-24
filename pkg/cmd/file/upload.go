package file

import (
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmd/message"
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type uploadOptions struct {
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
	opts := &uploadOptions{}

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
			return message.PostFiles(f, opts.channel, opts.filePaths, opts.comment, opts.title, opts.threadTS, opts.raw, opts.json)
		},
	}

	cmd.Flags().StringVar(&opts.title, "title", "", "File title in Slack (applies to first file)")
	cmd.Flags().StringVar(&opts.comment, "comment", "", "Text of the message the file(s) are posted in")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Disable Markdown→mrkdwn conversion of --comment")
	cmd.Flags().StringVar(&opts.threadTS, "thread-ts", "", "Thread timestamp to upload into")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}
