package thread

import (
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
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
