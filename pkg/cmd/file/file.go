package file

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdFile returns the top-level "file" command.
func NewCmdFile(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file <command>",
		Short: "Search and manage files",
		Long:  "Search files shared in Slack.",
	}

	cmd.AddCommand(NewCmdSearch(f))

	return cmd
}
