package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/build"
)

// NewCmdVersion returns the version command.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of slackbuzz CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "slackbuzz version %s (%s)\nbuilt %s\n",
				build.Version, build.Commit, build.Date)
		},
	}
}
