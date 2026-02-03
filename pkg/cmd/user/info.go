package user

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/api"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type infoOptions struct {
	factory *cmdutil.Factory
	user    string
	json    cmdutil.JSONFlags
}

// NewCmdInfo returns the "user info" command.
func NewCmdInfo(f *cmdutil.Factory) *cobra.Command {
	opts := &infoOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "info <user>",
		Short: "Show user profile",
		Long: `Display detailed information about a Slack user.

Accepts @username or a user ID.`,
		Example: `  slackbuzz user info @alice
  slackbuzz user info U012345 --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.user = args[0]
			return infoRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func infoRun(opts *infoOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	resolver := api.NewResolver(client.Slack)
	userID, err := resolver.ResolveUser(opts.user)
	if err != nil {
		return err
	}

	user, err := client.Slack.GetUserInfo(userID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, user)
	}

	displayName := user.Profile.DisplayName
	if displayName == "" {
		displayName = user.RealName
	}

	fmt.Fprintf(ios.Out, "%s\n", cs.Bold(displayName))
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("ID:"), user.ID)
	fmt.Fprintf(ios.Out, "  %-16s @%s\n", cs.Bold("Username:"), user.Name)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Real name:"), user.RealName)
	if user.Profile.Email != "" {
		fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Email:"), user.Profile.Email)
	}
	if user.Profile.Title != "" {
		fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Title:"), user.Profile.Title)
	}
	if user.Profile.StatusText != "" {
		fmt.Fprintf(ios.Out, "  %-16s %s %s\n", cs.Bold("Status:"), user.Profile.StatusEmoji, user.Profile.StatusText)
	}
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Timezone:"), user.TZ)

	status := cs.Green("active")
	if user.Deleted {
		status = cs.Red("deactivated")
	}
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Status:"), status)

	return nil
}
