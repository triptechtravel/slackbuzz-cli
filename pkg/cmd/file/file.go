package file

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdFile returns the top-level "file" command.
func NewCmdFile(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <command>",
		Short: "Search, upload, and manage files",
		Long:  "Search files shared in Slack, or upload files to channels and DMs.",
	}

	cmd.AddCommand(NewCmdSearch(f))
	cmd.AddCommand(NewCmdUpload(f))

	return cmd
}
