package channel

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdChannel returns the top-level "channel" command.
func NewCmdChannel(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel <command>",
		Short:   "Manage Slack channels",
		Long:    "List channels and view channel details.",
		Aliases: []string{"ch"},
	}

	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdInfo(f))

	return cmd
}
