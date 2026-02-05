package doctor

import (
	"fmt"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdDoctor returns the "doctor" command.
func NewCmdDoctor(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check token health and required scopes",
		Long: `Validate bot and user tokens, then probe key API scopes to detect
permission gaps before they cause errors.

Checks:
  - Bot token validity and channels:read scope
  - User token validity and channels:read scope
  - Reports pass/fail with remediation advice

Exit code 1 if any check fails.`,
		Example: `  slackbuzz doctor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doctorRun(f)
		},
	}
	return cmd
}

type checkResult struct {
	name    string
	passed  bool
	detail  string
	fix     string
}

func doctorRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	var results []checkResult
	anyFailed := false

	// --- Bot token ---
	botToken, botErr := auth.GetBotToken()
	if botErr != nil || botToken == "" {
		results = append(results, checkResult{
			name:   "Bot token",
			passed: false,
			detail: "not configured",
			fix:    "slackbuzz auth login --bot-token",
		})
		anyFailed = true
	} else {
		results = append(results, checkResult{
			name:   "Bot token",
			passed: true,
			detail: "configured",
		})

		// Probe channels:read with bot token
		botClient := slack.New(botToken)
		_, _, err := botClient.GetConversations(&slack.GetConversationsParameters{
			Types: []string{"public_channel"},
			Limit: 1,
		})
		if err != nil {
			results = append(results, checkResult{
				name:   "Bot channels:read",
				passed: false,
				detail: err.Error(),
				fix:    "Add channels:read to bot scopes at api.slack.com/apps > OAuth & Permissions",
			})
			anyFailed = true
		} else {
			results = append(results, checkResult{
				name:   "Bot channels:read",
				passed: true,
				detail: "OK",
			})
		}
	}

	// --- User token ---
	userToken, userErr := auth.GetUserToken()
	if userErr != nil || userToken == "" {
		results = append(results, checkResult{
			name:   "User token",
			passed: false,
			detail: "not configured",
			fix:    "slackbuzz auth login --user-token",
		})
		anyFailed = true
	} else {
		results = append(results, checkResult{
			name:   "User token",
			passed: true,
			detail: "configured",
		})

		// Probe channels:read with user token
		userClient := slack.New(userToken)
		_, _, err := userClient.GetConversations(&slack.GetConversationsParameters{
			Types: []string{"public_channel"},
			Limit: 1,
		})
		if err != nil {
			results = append(results, checkResult{
				name:   "User channels:read",
				passed: false,
				detail: err.Error(),
				fix:    "Add channels:read to user scopes, or re-run: slackbuzz app create",
			})
			anyFailed = true
		} else {
			results = append(results, checkResult{
				name:   "User channels:read",
				passed: true,
				detail: "OK",
			})
		}
	}

	// --- Print results ---
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Bold("SlackBuzz Doctor"))
	fmt.Fprintln(ios.Out, "────────────────────────────────────────")

	for _, r := range results {
		var icon string
		if r.passed {
			icon = cs.Green("PASS")
		} else {
			icon = cs.Red("FAIL")
		}
		fmt.Fprintf(ios.Out, "  [%s] %-22s %s\n", icon, r.name, r.detail)
		if !r.passed && r.fix != "" {
			fmt.Fprintf(ios.Out, "         %s %s\n", cs.Yellow("Fix:"), r.fix)
		}
	}

	fmt.Fprintln(ios.Out)

	if anyFailed {
		fmt.Fprintf(ios.ErrOut, "%s Some checks failed. See above for fixes.\n", cs.Red("!"))
		return &cmdutil.SilentError{Err: fmt.Errorf("doctor checks failed")}
	}

	fmt.Fprintf(ios.Out, "%s All checks passed.\n", cs.Green("✓"))
	return nil
}
