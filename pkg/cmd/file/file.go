package file

import (
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdFile returns the top-level "file" command.
func NewCmdFile(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <command>",
		Short: "Search, upload, and manage files",
		Long:  "Search files shared in Slack, upload files to channels and DMs, or delete them.",
	}

	cmd.AddCommand(NewCmdSearch(f))
	cmd.AddCommand(NewCmdUpload(f))
	cmd.AddCommand(NewCmdDelete(f))

	return cmd
}
