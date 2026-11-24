package user

import (
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdUser returns the top-level "user" command.
func NewCmdUser(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user <command>",
		Short: "Manage Slack users",
		Long:  "List workspace users and view user profiles.",
	}

	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdInfo(f))

	return cmd
}
