package auth

import (
	"fmt"

	"github.com/isaacrowntree/slackbuzz-cli/internal/auth"
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdLogout returns the "auth logout" command.
func NewCmdLogout(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of Slack",
		Long:  "Remove all stored authentication credentials for the Slack CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutRun(f)
		},
	}

	return cmd
}

func logoutRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	if err := auth.ClearTokens(); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Logged out of Slack\n", cs.Green("✓"))
	return nil
}
