package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/browser"
	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type updateOptions struct {
	factory       *cmdutil.Factory
	appID         string
	withToken     bool
	skipReinstall bool
}

// NewCmdUpdate returns the "app update" command.
//
// `app update` pushes the latest scope manifest to an existing Slack app
// (the one this CLI was installed against) via apps.manifest.update, then
// walks the user through reinstall + new tokens.
//
// Flow mirrors `app create`:
//
//  1. Push the latest manifest
//  2. Open the install/reinstall page
//  3. Prompt for new bot + user tokens
//  4. Validate + store
//
// Use it whenever the CLI's required scopes have grown (e.g. adding
// `im:history` to the user side so `message list @user` works).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	opts := &updateOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Push the latest scope manifest and re-authenticate",
		Long: `Update an existing Slack app's manifest with the CLI's current scope set,
then walk through reinstall + new tokens. Use this after the CLI gains new
scope requirements (e.g. extending DM read support).

The command uses your saved App Configuration Token and App ID. If neither is
saved, you'll be prompted, or pass --app-id and pipe the config token via stdin
with --with-token.`,
		Example: `  # Interactive (uses saved app ID + config token)
  slackbuzz app update

  # Specify the app ID explicitly
  slackbuzz app update --app-id A0123456789

  # CI / scripted (config token via stdin)
  echo "xoxe.xoxp-..." | slackbuzz app update --with-token --app-id A0123456789`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.appID, "app-id", "", "Slack app ID to update (defaults to saved value from `app create`)")
	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, "Read App Configuration Token from stdin")
	cmd.Flags().BoolVar(&opts.skipReinstall, "skip-reinstall", false, "Skip opening the reinstall page and the new-token prompts (manifest push only)")

	return cmd
}

func updateRun(opts *updateOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	ac, _ := config.LoadAuth()

	// Resolve App ID — flag wins, then saved value, then prompt
	appID := opts.appID
	if appID == "" {
		appID = ac.AppID
	}
	if appID == "" {
		if opts.withToken {
			return fmt.Errorf("no app ID saved. Pass --app-id A0123456789")
		}
		fmt.Fprintln(ios.Out, "Find your App ID at:")
		fmt.Fprintf(ios.Out, "  %s — click your app, the App ID is at the top under \"App Credentials\"\n", cs.Cyan("https://api.slack.com/apps"))
		fmt.Fprintln(ios.Out, "  (looks like A0123456789 — 11 chars, starts with A)")
		fmt.Fprintln(ios.Out)
		p := prompter.New(ios)
		entered, err := p.Input("App ID:", "")
		if err != nil {
			return fmt.Errorf("could not read app ID: %w", err)
		}
		appID = strings.TrimSpace(entered)
		if appID == "" {
			return fmt.Errorf("app ID cannot be empty")
		}
		ac.AppID = appID
	}

	// Resolve config token — flag/stdin wins, then saved value, then prompt
	var configToken string
	if opts.withToken {
		scanner := bufio.NewScanner(ios.In)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read config token from stdin")
		}
		configToken = strings.TrimSpace(scanner.Text())
	} else if ac.ConfigToken != "" {
		configToken = ac.ConfigToken
		fmt.Fprintf(ios.Out, "Using saved config token (from %s)\n", config.AuthFile())
	} else {
		fmt.Fprintln(ios.Out, "App Configuration Token needed.")
		fmt.Fprintln(ios.Out, "Where to find it:")
		fmt.Fprintf(ios.Out, "  1. Go to %s (the apps LIST page, not inside an app)\n", cs.Cyan("https://api.slack.com/apps"))
		fmt.Fprintln(ios.Out, "  2. Scroll to the bottom — section is titled \"Your App Configuration Tokens\"")
		fmt.Fprintln(ios.Out, "  3. Click \"Generate Token\" next to your workspace, copy the Access Token (xoxe.xoxp-...)")
		fmt.Fprintln(ios.Out, "  Note: this is NOT in App Credentials inside your app. It's a workspace-level admin token.")
		fmt.Fprintln(ios.Out)
		p := prompter.New(ios)
		entered, err := p.Password("Paste your App Configuration Token:")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
		configToken = strings.TrimSpace(entered)
	}
	if configToken == "" {
		return fmt.Errorf("config token cannot be empty")
	}
	ac.ConfigToken = configToken
	ac.AppID = appID
	if err := ac.Save(); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save auth config: %v\n", err)
	}

	// Push the latest manifest
	fmt.Fprintf(ios.Out, "\nPushing latest manifest to app %s...\n", cs.Bold(appID))

	manifestJSON, err := json.Marshal(appManifest)
	if err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}
	body := map[string]interface{}{
		"app_id":   appID,
		"manifest": string(manifestJSON),
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/apps.manifest.update", bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+configToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var result manifestUpdateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		errMsg := fmt.Sprintf("Slack API error: %s", result.Error)
		for _, e := range result.Errors {
			errMsg += fmt.Sprintf("\n  %s: %s", e.Pointer, e.Message)
		}
		return fmt.Errorf("%s", errMsg)
	}

	fmt.Fprintf(ios.Out, "%s Manifest updated\n", cs.Green("✓"))

	if opts.skipReinstall {
		fmt.Fprintln(ios.Out, "Skipping reinstall (per --skip-reinstall). Reinstall the app and run `slackbuzz auth login` to refresh tokens.")
		return nil
	}

	// Open reinstall page so the new scopes take effect
	installURL := fmt.Sprintf("https://api.slack.com/apps/%s/install-on-team", appID)
	fmt.Fprintf(ios.Out, "\n%s Opening reinstall page in your browser...\n", cs.Blue("→"))
	fmt.Fprintf(ios.Out, "  %s\n\n", installURL)
	_ = browser.Open(installURL)

	fmt.Fprintln(ios.Out, "Reinstall the app to your workspace, then come back here to paste the new tokens.")
	fmt.Fprintln(ios.Out)

	// Reuse storeDetectedToken from create.go for token paste flow
	p := prompter.New(ios)
	storedBot := false
	storedUser := false

	oauthURL := fmt.Sprintf("https://api.slack.com/apps/%s/oauth", appID)
	fmt.Fprintln(ios.Out, cs.Bold("After reinstalling, grab the OAuth tokens from the OAuth & Permissions page:"))
	fmt.Fprintf(ios.Out, "  %s\n", cs.Cyan(oauthURL))
	fmt.Fprintln(ios.Out, "  • Bot User OAuth Token (xoxb-...) — top of page")
	fmt.Fprintln(ios.Out, "  • User OAuth Token (xoxp-...) — just below it")
	fmt.Fprintln(ios.Out, "Token type is auto-detected from prefix.")
	fmt.Fprintln(ios.Out)

	token1, err := p.Password("Paste your first token:")
	if err != nil {
		fmt.Fprintln(ios.ErrOut, "Skipping tokens. You can finish later with: slackbuzz auth login")
	} else {
		token1 = strings.TrimSpace(token1)
		if token1 != "" {
			storedBot, storedUser = storeDetectedToken(ios, cs, token1, storedBot, storedUser)
		}
	}

	if !storedBot || !storedUser {
		missing := "user (xoxp-)"
		if !storedBot {
			missing = "bot (xoxb-)"
		}
		fmt.Fprintf(ios.Out, "(Optional) Paste your %s token:\n", missing)
		token2, err := p.Password("Token (enter to skip):")
		if err == nil {
			token2 = strings.TrimSpace(token2)
			if token2 != "" {
				storedBot, storedUser = storeDetectedToken(ios, cs, token2, storedBot, storedUser)
			}
		}
	}

	fmt.Fprintf(ios.Out, "%s Update complete. Try the command that prompted this update.\n", cs.Green("✓"))

	return nil
}

type manifestUpdateResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	AppID  string `json:"app_id,omitempty"`
	Errors []struct {
		Message string `json:"message"`
		Pointer string `json:"pointer"`
	} `json:"errors,omitempty"`
}
