package app

import (
	"errors"
	"fmt"

	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/root"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// Run bootstraps and executes the CLI. Returns exit code.
func Run() int {
	ios := iostreams.System()
	f := cmdutil.NewFactory(ios)

	rootCmd := root.NewCmdRoot(f)

	if err := rootCmd.Execute(); err != nil {
		if cmdutil.IsSilentError(err) {
			return 1
		}

		// Auth expired (401 from Slack API)
		var authExpired *api.AuthExpiredError
		if errors.As(err, &authExpired) {
			fmt.Fprintln(ios.ErrOut, authExpired.Error())
			return 4
		}

		if cmdutil.IsAuthError(err) {
			fmt.Fprintln(ios.ErrOut, err.Error())
			return 4
		}
		if cmdutil.IsTokenTypeError(err) {
			fmt.Fprintln(ios.ErrOut, err.Error())
			return 4
		}
		fmt.Fprintln(ios.ErrOut, err.Error())
		return 1
	}

	return 0
}
