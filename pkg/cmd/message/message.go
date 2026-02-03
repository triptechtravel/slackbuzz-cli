package message

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdMessage returns the top-level "message" command.
func NewCmdMessage(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message <command>",
		Short: "Send and read Slack messages",
		Long:  "Post messages to channels, read channel history, and search messages.",
		Aliases: []string{"msg"},
	}

	cmd.AddCommand(NewCmdSend(f))
	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdSearch(f))
	cmd.AddCommand(NewCmdEdit(f))
	cmd.AddCommand(NewCmdDelete(f))

	return cmd
}
