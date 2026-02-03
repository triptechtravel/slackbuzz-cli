package cmdutil

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
)

// NeedsAuth returns a pre-run function that validates any token is available.
func NeedsAuth(f *Factory) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		bot, _ := auth.GetBotToken()
		user, _ := auth.GetUserToken()
		if bot == "" && user == "" {
			return &AuthError{}
		}
		return nil
	}
}

// NeedsBotToken returns a pre-run function that validates a bot token is available.
func NeedsBotToken(f *Factory) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		token, _ := auth.GetBotToken()
		if token == "" {
			return &TokenTypeError{Need: "bot"}
		}
		return nil
	}
}

// NeedsUserToken returns a pre-run function that validates a user token is available.
func NeedsUserToken(f *Factory) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		token, _ := auth.GetUserToken()
		if token == "" {
			return &TokenTypeError{Need: "user"}
		}
		return nil
	}
}
