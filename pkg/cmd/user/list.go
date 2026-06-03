package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
	"github.com/triptechtravel/slackbuzz-cli/internal/tableprinter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type listOptions struct {
	factory            *cmdutil.Factory
	includeDeactivated bool
	limit              int
	json               cmdutil.JSONFlags
}

// NewCmdList returns the "user list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspace users",
		Long:  "List all active users in the Slack workspace.",
		Example: `  # List active users
  slackbuzz user list

  # Include deactivated users
  slackbuzz user list --include-deactivated

  # Output as JSON
  slackbuzz user list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.includeDeactivated, "include-deactivated", false, "Include deactivated users")
	cmd.Flags().IntVar(&opts.limit, "limit", 200, "Maximum number of users to list")
	cmdutil.AddJSONFlags(cmd, &opts.json)

	return cmd
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.DefaultClient()
	if err != nil {
		return err
	}

	var users []*slackapi.User
	cursor := ""
	for {
		resp, err := slackapi.UsersList(context.Background(), client.API, &slackapi.UsersListParams{
			Limit:  1000,
			Cursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		users = append(users, resp.Members...)
		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	type userRow struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		RealName    string `json:"real_name"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		IsBot       bool   `json:"is_bot"`
		Deleted     bool   `json:"deleted"`
	}

	var rows []userRow
	for _, u := range users {
		if u == nil || u.IsBot || u.ID == "USLACKBOT" {
			continue
		}
		if u.Deleted && !opts.includeDeactivated {
			continue
		}
		row := userRow{
			ID:       u.ID,
			Name:     u.Name,
			RealName: u.RealName,
			IsBot:    u.IsBot,
			Deleted:  u.Deleted,
		}
		if u.Profile != nil {
			row.DisplayName = u.Profile.DisplayName
			row.Email = u.Profile.Email
		}
		rows = append(rows, row)
		if len(rows) >= opts.limit {
			break
		}
	}
	if len(rows) == 0 {
		fmt.Fprintln(ios.Out, "No users found.")
		return nil
	}

	if opts.json.WantsJSON() {
		return opts.json.OutputJSON(ios.Out, rows)
	}

	tp := tableprinter.New(ios)
	for _, u := range rows {
		tp.AddField(u.ID)
		tp.AddField(cs.Bold("@" + u.Name))
		displayName := u.RealName
		if u.DisplayName != "" {
			displayName = u.DisplayName
		}
		tp.AddField(displayName)
		status := cs.Green("active")
		if u.Deleted {
			status = cs.Red("deactivated")
		}
		tp.AddField(status)
		tp.EndRow()
	}

	return tp.Render()
}
