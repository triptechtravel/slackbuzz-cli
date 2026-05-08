package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type uploadOptions struct {
	factory   *cmdutil.Factory
	filePaths []string
	channel   string
	title     string
	comment   string
	threadTS  string
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
file paths to upload. Each file is uploaded individually and shared to the
target channel.

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
	cmd.Flags().StringVar(&opts.comment, "comment", "", "Initial comment posted with the file(s)")
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
	for _, fp := range opts.filePaths {
		info, err := os.Stat(fp)
		if err != nil {
			return fmt.Errorf("file not found: %s", fp)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory, not a file", fp)
		}
		if info.Size() == 0 {
			return fmt.Errorf("%s is empty", fp)
		}
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

	var results []uploadResult
	ctx := context.Background()

	for i, fp := range opts.filePaths {
		_, _ = os.Stat(fp) // already validated above
		fileName := filepath.Base(fp)

		title := fileName
		if opts.title != "" && i == 0 {
			title = opts.title
		}

		params := &slackapi.UploadFileParams{
			FilePath: fp,
			Filename: fileName,
			Title:    title,
			Channels: channelID,
		}
		if opts.comment != "" && i == 0 {
			params.InitialComment = opts.comment
		}
		if opts.threadTS != "" {
			params.ThreadTS = opts.threadTS
		}

		summary, err := slackapi.UploadFile(ctx, client.API, params)
		if err != nil {
			return fmt.Errorf("failed to upload %s: %s", fileName, api.FormatError(err))
		}

		results = append(results, uploadResult{
			FileID:   summary.File.ID,
			Title:    summary.File.Title,
			Filename: fileName,
			Channel:  channelID,
		})

		if !opts.json.WantsJSON() {
			fmt.Fprintf(ios.Out, "%s Uploaded %s to %s\n",
				cs.Green("✓"), cs.Bold(fileName), cs.Bold(opts.channel))
		}
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, results)
	}

	if len(results) > 1 {
		fmt.Fprintf(ios.Out, "\n%s %d files uploaded to %s\n",
			cs.Green("✓"), len(results), cs.Bold(opts.channel))
	}

	return nil
}
