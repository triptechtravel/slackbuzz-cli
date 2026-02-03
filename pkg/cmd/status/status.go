package status

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmd/activity"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

// NewCmdStatus returns the "status" command group.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Slack status management",
		Long: `View, set, or clear your Slack status.

Without a subcommand, shows your current status.
Requires a user token (xoxp-) with users.profile:write scope.`,
		Example: `  # Show current status
  slackbuzz status

  # Set status
  slackbuzz status set "In a meeting" :calendar:

  # Set status with expiration
  slackbuzz status set "Reviewing PRs" :eyes: --until 2h

  # Clear status
  slackbuzz status clear`,
		PersistentPreRunE: cmdutil.NeedsUserToken(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRun(f)
		},
	}

	cmd.AddCommand(newCmdSet(f))
	cmd.AddCommand(newCmdClear(f))

	return cmd
}

// --- show (default) ---

func showRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	profile, err := getProfile(token)
	if err != nil {
		return err
	}

	statusText := profile.StatusText
	statusEmoji := profile.StatusEmoji

	if statusText == "" && statusEmoji == "" {
		fmt.Fprintln(ios.Out, "No status set.")
		return nil
	}

	fmt.Fprintf(ios.Out, "%s %s %s\n",
		cs.Bold("Status:"),
		statusEmoji,
		statusText,
	)

	if profile.StatusExpiration > 0 {
		fmt.Fprintf(ios.Out, "%s expires at Unix %d\n",
			cs.Gray("  "),
			profile.StatusExpiration,
		)
	}

	return nil
}

// --- set ---

type setOptions struct {
	factory *cmdutil.Factory
	text    string
	emoji   string
	until   string
}

func newCmdSet(f *cmdutil.Factory) *cobra.Command {
	opts := &setOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "set <text> [emoji]",
		Short: "Set your Slack status",
		Long: `Set your Slack status text and emoji.

The emoji argument is optional and should be in :emoji: format.
Use --until to set an expiration (e.g. 2h, 1d).`,
		Example: `  slackbuzz status set "In a meeting" :calendar:
  slackbuzz status set "Coding" :computer: --until 2h
  slackbuzz status set "Lunch" :fork_and_knife: --until 1h`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.text = args[0]
			if len(args) > 1 {
				opts.emoji = args[1]
			}
			return setRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.until, "until", "", "Status expiration (e.g. 2h, 1d)")

	return cmd
}

func setRun(opts *setOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	var expiration int64
	if opts.until != "" {
		d, parseErr := activity.ParseDuration(opts.until)
		if parseErr != nil {
			return fmt.Errorf("invalid --until value: %w", parseErr)
		}
		expiration = time.Now().Add(d).Unix()
	}

	profile := map[string]interface{}{
		"status_text":       opts.text,
		"status_emoji":      opts.emoji,
		"status_expiration": expiration,
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	if err := setProfile(token, string(profileJSON)); err != nil {
		return err
	}

	display := opts.text
	if opts.emoji != "" {
		display = opts.emoji + " " + display
	}
	fmt.Fprintf(ios.Out, "%s Status set: %s\n", cs.Green("✓"), display)

	if opts.until != "" {
		fmt.Fprintf(ios.Out, "  Expires in %s\n", opts.until)
	}

	return nil
}

// --- clear ---

func newCmdClear(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear your Slack status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return clearRun(f)
		},
	}

	return cmd
}

func clearRun(f *cmdutil.Factory) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	token, err := auth.GetUserToken()
	if err != nil {
		return err
	}

	profile := map[string]interface{}{
		"status_text":       "",
		"status_emoji":      "",
		"status_expiration": 0,
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	if err := setProfile(token, string(profileJSON)); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Status cleared\n", cs.Green("✓"))
	return nil
}

// --- helpers ---

type profileData struct {
	StatusText       string `json:"status_text"`
	StatusEmoji      string `json:"status_emoji"`
	StatusExpiration int64  `json:"status_expiration"`
}

func getProfile(token string) (*profileData, error) {
	v := url.Values{}
	v.Set("token", token)

	resp, err := http.PostForm("https://slack.com/api/users.profile.get", v)
	if err != nil {
		return nil, fmt.Errorf("users.profile.get failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error,omitempty"`
		Profile profileData `json:"profile"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("users.profile.get error: %s", result.Error)
	}

	return &result.Profile, nil
}

func setProfile(token, profileJSON string) error {
	v := url.Values{}
	v.Set("token", token)
	v.Set("profile", profileJSON)

	resp, err := http.PostForm("https://slack.com/api/users.profile.set", v)
	if err != nil {
		return fmt.Errorf("users.profile.set failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("users.profile.set error: %s", result.Error)
	}

	return nil
}
