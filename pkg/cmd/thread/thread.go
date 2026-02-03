package thread

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdThread returns the top-level "thread" command.
func NewCmdThread(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thread <command>",
		Short: "Work with Slack threads",
		Long:  "Link Slack threads to external tools like ClickUp.",
	}

	cmd.AddCommand(NewCmdLink(f))

	return cmd
}
