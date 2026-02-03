package root

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/app"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/auth"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/channel"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/completion"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/digest"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/dm"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/later"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/message"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/notify"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/react"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/status"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/thread"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/threads"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/user"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/version"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdRoot creates the root command for the slack CLI.
func NewCmdRoot(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slackbuzz",
		Short: "Slack CLI - message, search, and manage channels from the command line",
		Long: `Work with Slack channels, messages, and users from your terminal.

Integrates with ClickUp and GitHub CLIs for cross-tool developer workflows.

GETTING STARTED
  slackbuzz app create          Create a Slack app with required scopes
  slackbuzz auth login           Log in with bot/user tokens

INBOX & ACTIVITY
  slackbuzz activity             See mentions, DMs, and threads (alias: inbox)
  slackbuzz threads              Threads you're participating in
  slackbuzz dm list              DM conversations with recent activity
  slackbuzz digest               Cross-tool briefing (Slack + ClickUp + GitHub)

MESSAGING
  slackbuzz message list <chan>  Read channel/thread history
  slackbuzz message send <chan>  Send a message
  slackbuzz message search <q>  Search messages (user token required)
  slackbuzz react <chan> <ts>    React to a message

CHANNELS & USERS
  slackbuzz channel list         List channels
  slackbuzz user list            List workspace members

SAVED ITEMS & STATUS
  slackbuzz later list           Show saved/bookmarked messages
  slackbuzz status               View or set your Slack status

TIPS
  Most commands support --json for JSON output and --jq for filtering.
  Use --help on any command for full usage details.
  Deeplinks let you click to open items directly in Slack.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Group annotations for help display
	cmd.AddGroup(
		&cobra.Group{ID: "inbox", Title: "Inbox & Activity:"},
		&cobra.Group{ID: "messaging", Title: "Messaging:"},
		&cobra.Group{ID: "workspace", Title: "Channels & Users:"},
		&cobra.Group{ID: "tools", Title: "Tools & Integrations:"},
		&cobra.Group{ID: "setup", Title: "Setup:"},
	)

	// Setup commands
	appCmd := app.NewCmdApp(f)
	appCmd.GroupID = "setup"
	cmd.AddCommand(appCmd)

	authCmd := auth.NewCmdAuth(f)
	authCmd.GroupID = "setup"
	cmd.AddCommand(authCmd)

	// Inbox & activity commands
	activityCmd := activity.NewCmdActivity(f)
	activityCmd.GroupID = "inbox"
	cmd.AddCommand(activityCmd)

	threadsCmd := threads.NewCmdThreads(f)
	threadsCmd.GroupID = "inbox"
	cmd.AddCommand(threadsCmd)

	dmCmd := dm.NewCmdDM(f)
	dmCmd.GroupID = "inbox"
	cmd.AddCommand(dmCmd)

	laterCmd := later.NewCmdLater(f)
	laterCmd.GroupID = "inbox"
	cmd.AddCommand(laterCmd)

	digestCmd := digest.NewCmdDigest(f)
	digestCmd.GroupID = "inbox"
	cmd.AddCommand(digestCmd)

	// Messaging commands
	messageCmd := message.NewCmdMessage(f)
	messageCmd.GroupID = "messaging"
	cmd.AddCommand(messageCmd)

	reactCmd := react.NewCmdReact(f)
	reactCmd.GroupID = "messaging"
	cmd.AddCommand(reactCmd)

	threadCmd := thread.NewCmdThread(f)
	threadCmd.GroupID = "messaging"
	cmd.AddCommand(threadCmd)

	notifyCmd := notify.NewCmdNotify(f)
	notifyCmd.GroupID = "messaging"
	cmd.AddCommand(notifyCmd)

	// Workspace commands
	channelCmd := channel.NewCmdChannel(f)
	channelCmd.GroupID = "workspace"
	cmd.AddCommand(channelCmd)

	userCmd := user.NewCmdUser(f)
	userCmd.GroupID = "workspace"
	cmd.AddCommand(userCmd)

	statusCmd := status.NewCmdStatus(f)
	statusCmd.GroupID = "workspace"
	cmd.AddCommand(statusCmd)

	// Utility commands
	versionCmd := version.NewCmdVersion()
	versionCmd.GroupID = "tools"
	cmd.AddCommand(versionCmd)

	completionCmd := completion.NewCmdCompletion(cmd)
	completionCmd.GroupID = "tools"
	cmd.AddCommand(completionCmd)

	return cmd
}
