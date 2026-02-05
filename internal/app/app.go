package app

import (
	"encoding/json"
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

		exitCode := 1
		errType := "error"

		// Auth expired (401 from Slack API)
		var authExpired *api.AuthExpiredError
		if errors.As(err, &authExpired) {
			exitCode = 4
			errType = "auth_expired"
		} else if cmdutil.IsAuthError(err) {
			exitCode = 4
			errType = "auth_required"
		} else if cmdutil.IsTokenTypeError(err) {
			exitCode = 4
			errType = "token_type"
		} else if api.IsMissingScopeError(err) {
			errType = "missing_scope"
		}

		// Agent mode: structured JSON to stderr
		if f.IsAgentMode() {
			suggestion := ""
			switch errType {
			case "auth_expired", "auth_required":
				suggestion = "Run 'slackbuzz auth login' to re-authenticate"
			case "token_type":
				suggestion = "Run 'slackbuzz auth login' with the required token type"
			case "missing_scope":
				suggestion = "Re-install app with updated scopes, or use --as-bot"
			}
			agentErr := map[string]string{
				"error": err.Error(),
				"type":  errType,
			}
			if suggestion != "" {
				agentErr["suggestion"] = suggestion
			}
			data, _ := json.Marshal(agentErr)
			fmt.Fprintln(ios.ErrOut, string(data))
			return exitCode
		}

		fmt.Fprintln(ios.ErrOut, err.Error())
		return exitCode
	}

	return 0
}
