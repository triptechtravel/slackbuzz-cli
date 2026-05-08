//go:generate go run ../../../cmd/gen-manifest -spec ../../../api/specs/slack_web.json -cmd-root ../../../pkg/cmd -out manifest.go

package app

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
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
