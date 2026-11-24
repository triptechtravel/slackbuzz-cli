//go:generate go run ../../../cmd/gen-manifest -cmd-root ../../../pkg/cmd -out manifest.go

package app

import (
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdApp returns the top-level "app" command.
func NewCmdApp(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app <command>",
		Short: "Manage Slack app setup",
		Long:  "Create and configure the Slack app used by this CLI.",
	}

	cmd.AddCommand(NewCmdCreate(f))
	cmd.AddCommand(NewCmdUpdate(f))

	return cmd
}
