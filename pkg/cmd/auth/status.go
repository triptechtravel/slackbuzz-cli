package auth

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdStatus returns the "auth status" command.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Display information about the current authentication state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return statusRun(f)
		},
	}

	return cmd
}

func statusRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	teamID, teamName := auth.GetTeamInfo()

	// Check bot token
	botToken, botErr := auth.GetBotToken()
	botStatus := cs.Red("not set")
	botUser := ""
	if botErr == nil && botToken != "" {
		info, err := auth.ValidateToken(botToken)
		if err != nil {
			botStatus = cs.Yellow("invalid")
		} else {
			botStatus = cs.Green("valid")
			botUser = info.User
			if teamID == "" {
				teamID = info.TeamID
				teamName = info.Team
			}
		}
	}

	// Check user token
	userToken, userErr := auth.GetUserToken()
	userStatus := cs.Red("not set")
	userUser := ""
	if userErr == nil && userToken != "" {
		info, err := auth.ValidateToken(userToken)
		if err != nil {
			userStatus = cs.Yellow("invalid")
		} else {
			userStatus = cs.Green("valid")
			userUser = info.User
			if teamID == "" {
				teamID = info.TeamID
				teamName = info.Team
			}
		}
	}

	if botToken == "" && userToken == "" {
		fmt.Fprintf(ios.Out, "%s Not logged in. Run 'slackbuzz auth login' to authenticate.\n", cs.Red("✗"))
		return nil
	}

	team := teamName
	if team == "" {
		team = "(unknown)"
	}
	if teamID != "" {
		team = fmt.Sprintf("%s (%s)", team, teamID)
	}

	fmt.Fprintf(ios.Out, "%s Slack authentication status\n", cs.Green("✓"))
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Team:"), team)
	fmt.Fprintf(ios.Out, "  %-16s %s", cs.Bold("Bot token:"), botStatus)
	if botUser != "" {
		fmt.Fprintf(ios.Out, " (%s)", botUser)
	}
	fmt.Fprintln(ios.Out)
	fmt.Fprintf(ios.Out, "  %-16s %s", cs.Bold("User token:"), userStatus)
	if userUser != "" {
		fmt.Fprintf(ios.Out, " (%s)", userUser)
	}
	fmt.Fprintln(ios.Out)

	return nil
}
