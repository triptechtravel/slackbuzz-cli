package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type updateOptions struct {
	factory       *cmdutil.Factory
	appID         string
	source        tokenSource
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
//  2. Reinstall via the browser OAuth flow (or paste tokens if the app's
//     client credentials aren't saved)
//  3. Validate + store
//
// Use it whenever the CLI's required scopes have grown (e.g. adding
// `im:history` to the user side so `message list @user` works).
func NewCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	opts := &updateOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Push the latest scope manifest and re-authenticate",
		Long: `Update an existing Slack app's manifest with the CLI's current scope set,
then reinstall it and store new tokens. Use this after the CLI gains new
scope requirements (e.g. extending DM read support).

The command uses your saved App Configuration Token and App ID. If neither is
saved, you'll be prompted, or pass --app-id and supply the config token with
--clipboard, --with-token (stdin) or $SLACKBUZZ_CONFIG_TOKEN.

Reinstalling opens Slack in your browser — click Allow and the new tokens are
stored automatically. Apps made before this flow existed don't have their
Client ID and Secret saved; you'll be asked for them once (Basic Information >
App Credentials), or can press enter to paste tokens instead.`,
		Example: `  # Interactive (uses saved app ID + config token)
  slackbuzz app update

  # Specify the app ID explicitly
  slackbuzz app update --app-id A0123456789

  # Copy the config token, then
  slackbuzz app update --clipboard

  # CI / scripted (config token via stdin)
  echo "xoxe.xoxp-..." | slackbuzz app update --with-token --app-id A0123456789`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateRun(opts)
		},
	}

	cmd.Flags().StringVar(&opts.appID, "app-id", "", "Slack app ID to update (defaults to saved value from `app create`)")
	cmd.Flags().BoolVar(&opts.source.withToken, "with-token", false, "Read App Configuration Token from stdin")
	cmd.Flags().BoolVar(&opts.source.fromClipboard, "clipboard", false, "Read App Configuration Token from the system clipboard")
	cmd.MarkFlagsMutuallyExclusive("with-token", "clipboard")
	cmd.Flags().BoolVar(&opts.skipReinstall, "skip-reinstall", false, "Skip opening the reinstall page and the new-token prompts (manifest push only)")

	return cmd
}

func updateRun(opts *updateOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	ac, err := loadAuth()
	if err != nil {
		return err
	}
	// Saved client credentials belong to the saved app; don't reinstall a
	// different app with them.
	savedAppID := ac.AppID

	// Resolve App ID — flag wins, then saved value, then prompt
	appID := opts.appID
	if appID == "" {
		appID = ac.AppID
	}
	if appID == "" {
		if !ios.CanPrompt() {
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

	// Resolve config token — stdin/clipboard/env/saved, then prompt
	configToken, save, err := configTokenFrom(ios, opts.source, ac)
	if errors.Is(err, errNoConfigToken) {
		if !ios.CanPrompt() {
			return nonInteractiveTokenHelp()
		}
		configToken, err = promptConfigToken(ios, cs)
		save = true
	}
	if err != nil {
		return err
	}
	if save {
		ac.ConfigToken = configToken
	}
	if appID != savedAppID {
		ac.ClientID, ac.ClientSecret = "", ""
	}
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

	// Reinstall so the new scopes take effect. Apps created before the OAuth
	// flow existed have no saved client credentials — ask once, and save
	// them only after Slack accepts them.
	app := oauthApp{TeamID: savedTeamID()}
	prompted := false
	if appID == savedAppID && ac.ClientID != "" {
		app.ClientID, app.ClientSecret = ac.ClientID, auth.GetClientSecret()
	}
	if (app.ClientID == "" || app.ClientSecret == "") && ios.CanPrompt() {
		app.ClientID, app.ClientSecret = promptClientCredentials(ios, cs, appID)
		prompted = true
	}
	installApp(ios, cs, appID, app, prompted)

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

// savedTeamID is the workspace the stored tokens belong to, if known.
func savedTeamID() string {
	id, _ := auth.GetTeamInfo()
	return id
}

// promptClientCredentials asks for the app's Client ID and Secret so the
// reinstall can run through OAuth. Empty results mean "paste tokens instead".
func promptClientCredentials(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, appID string) (string, string) {
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, "To reinstall in the browser (no token copying), slackbuzz needs this app's credentials, once:")
	fmt.Fprintf(ios.Out, "  %s — under App Credentials\n", cs.Cyan(fmt.Sprintf("https://api.slack.com/apps/%s/general", appID)))
	p := prompter.New(ios)
	clientID, err := p.Input("Client ID (enter to paste tokens instead):", "")
	if err != nil || strings.TrimSpace(clientID) == "" {
		return "", ""
	}
	clientSecret, err := p.Password("Client Secret:")
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
}
