package file

import (
	"context"
	"fmt"
	"strings"

	"github.com/isaacrowntree/slackbuzz-cli/internal/api"
	"github.com/isaacrowntree/slackbuzz-cli/internal/prompter"
	"github.com/isaacrowntree/slackbuzz-cli/internal/slackapi"
	"github.com/isaacrowntree/slackbuzz-cli/pkg/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmdDelete returns the "file delete" command.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	var confirm, asBot bool
	cmd := &cobra.Command{
		Use:   "delete <file-id>...",
		Short: "Delete files",
		Long: `Delete files by ID (F…). Deleting a message leaves its files in Slack;
this removes them. The IDs are in send --file and file upload --json output.

Prompts for confirmation in interactive mode; --confirm is required otherwise.`,
		Example:           `  slackbuzz file delete F0CK25W3ABH F0CKKDV2K8R --confirm`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, ids []string) error {
			ios := f.IOStreams
			client, err := f.UserClient()
			if asBot {
				client, err = f.DefaultClient()
			}
			if err != nil {
				return err
			}
			if !confirm {
				if !ios.IsTerminal() {
					return fmt.Errorf("--confirm is required in non-interactive mode")
				}
				ok, err := prompter.New(ios).Confirm(fmt.Sprintf("Delete %s?", strings.Join(ids, ", ")), false)
				if err != nil || !ok {
					return err
				}
			}
			for _, id := range ids {
				if _, err := slackapi.FilesDelete(context.Background(), client.API, &slackapi.FilesDeleteParams{File: id}); err != nil {
					return fmt.Errorf("delete %s: %s", id, api.FormatError(err))
				}
				fmt.Fprintf(ios.Out, "%s Deleted %s\n", ios.ColorScheme().Green("✓"), id)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&asBot, "as-bot", false, "Delete as the bot (for files the bot posted)")
	return cmd
}
