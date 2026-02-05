package auth

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type loginOptions struct {
	factory   *cmdutil.Factory
	botToken  bool
	userToken bool
	withToken bool // read from stdin (CI)
}

// NewCmdLogin returns the "auth login" command.
func NewCmdLogin(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Slack",
		Long: `Store Slack API tokens for CLI access.

Slack uses two types of tokens:
  Bot token (xoxb-)  — posting, reading channels, users
  User token (xoxp-) — searching messages (search:read scope)

The token type is auto-detected from its prefix. You can paste either
type and it will be stored correctly.

Get tokens from your Slack App: api.slack.com/apps > OAuth & Permissions.

In non-interactive environments (CI), pipe a token via stdin:
  echo "xoxb-..." | slackbuzz auth login --with-token`,
		Example: `  # Interactive (auto-detects token type)
  slackbuzz auth login

  # Hint which type you're providing
  slackbuzz auth login --bot-token
  slackbuzz auth login --user-token

  # Pipe token for CI
  echo "xoxb-..." | slackbuzz auth login --with-token`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return loginRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.botToken, "bot-token", false, "Hint: storing a bot token (xoxb-)")
	cmd.Flags().BoolVar(&opts.userToken, "user-token", false, "Hint: storing a user token (xoxp-)")
	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, "Read token from standard input (for CI)")

	return cmd
}

func loginRun(opts *loginOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	var token string

	if opts.withToken {
		// Read token from stdin (CI mode)
		scanner := bufio.NewScanner(ios.In)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read token from stdin")
		}
		token = strings.TrimSpace(scanner.Text())
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}
	} else {
		// Interactive token entry
		hint := "bot or user"
		if opts.botToken {
			hint = "bot (xoxb-)"
		} else if opts.userToken {
			hint = "user (xoxp-)"
		}
		fmt.Fprintln(ios.Out, "Get your token from: api.slack.com/apps > OAuth & Permissions")
		fmt.Fprintln(ios.Out)

		p := prompter.New(ios)
		var err error
		token, err = p.Password(fmt.Sprintf("Paste your %s token:", hint))
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}
	}

	// Auto-detect token type from prefix
	tokenType := auth.DetectTokenType(token)
	if tokenType == "unknown" {
		return fmt.Errorf("unrecognized token prefix. Expected xoxb- (bot) or xoxp- (user)")
	}

	// Validate the token via auth.test
	fmt.Fprintln(ios.Out, "Validating token...")
	info, err := auth.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Store the token based on detected type
	if tokenType == "bot" {
		if err := auth.StoreBotToken(token); err != nil {
			return fmt.Errorf("failed to store bot token: %w", err)
		}
	} else {
		if err := auth.StoreUserToken(token); err != nil {
			return fmt.Errorf("failed to store user token: %w", err)
		}
	}

	// Store team info and token-specific user info
	_ = auth.StoreTeamInfo(info.TeamID, info.Team)
	if tokenType == "bot" {
		_ = auth.StoreBotUserInfo(info.UserID, info.User)
	} else {
		_ = auth.StoreUserInfo(info.UserID, info.User)
	}

	fmt.Fprintf(ios.Out, "%s Logged in as %s (%s token) — team: %s\n",
		cs.Green("✓"),
		cs.Bold(info.User),
		tokenType,
		cs.Bold(info.Team),
	)

	return nil
}
