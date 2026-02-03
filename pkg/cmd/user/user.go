package user

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
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
