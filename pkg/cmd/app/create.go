package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/slackbuzz-cli/internal/browser"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type createOptions struct {
	factory *cmdutil.Factory
	source  tokenSource
	web     bool
}

// NewCmdCreate returns the "app create" command.
func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Slack app with required scopes",
		Long: `Create a new Slack app pre-configured with all scopes needed by the CLI,
install it, and store its tokens.

Two ways to create the app:

  Config token (default) — paste an App Configuration Token from
  api.slack.com/apps > Your App Configuration Tokens. The app is created
  via the manifest API.

  --web — no config token. Slack's "create from manifest" page opens
  pre-filled; click Create, then paste the Client ID and Client Secret.

Either way, Slack then opens in your browser: click Allow and both the bot
and user tokens are stored automatically. No token copying.`,
		Example: `  # Interactive
  slackbuzz app create

  # Copy the config token, then (works without a terminal prompt)
  slackbuzz app create --clipboard

  # No config token: create the app in the browser
  slackbuzz app create --web

  # Scripted
  echo "xoxe.xoxp-..." | slackbuzz app create --with-token
  SLACKBUZZ_CONFIG_TOKEN=xoxe.xoxp-... slackbuzz app create`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return createRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.source.withToken, "with-token", false, "Read config token from stdin")
	cmd.Flags().BoolVar(&opts.source.fromClipboard, "clipboard", false, "Read config token from the system clipboard")
	cmd.Flags().BoolVar(&opts.web, "web", false, "Create the app in the browser instead of with a config token")
	cmd.MarkFlagsMutuallyExclusive("with-token", "clipboard", "web")

	return cmd
}

func createRun(opts *createOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	ac, err := loadAuth()
	if err != nil {
		return err
	}

	if opts.web {
		return createViaWeb(ios, cs)
	}

	configToken, save, err := configTokenFrom(ios, opts.source, ac)
	if errors.Is(err, errNoConfigToken) {
		if !ios.CanPrompt() {
			return nonInteractiveTokenHelp()
		}
		idx, perr := prompter.New(ios).Select("How do you want to create the Slack app?", []string{
			"Paste an App Configuration Token (fastest)",
			"Create it in the browser (no config token)",
		})
		if perr != nil {
			return perr
		}
		if idx == 1 {
			return createViaWeb(ios, cs)
		}
		configToken, err = promptConfigToken(ios, cs)
		save = true
	}
	if err != nil {
		return err
	}

	// Save the config token immediately so it's not lost if the API call fails
	if save {
		ac.ConfigToken = configToken
		if err := ac.Save(); err != nil {
			fmt.Fprintf(ios.ErrOut, "Warning: could not save config token: %v\n", err)
		}
	}

	fmt.Fprintln(ios.Out, "Creating Slack app...")

	// Call apps.manifest.create — the manifest must be a JSON-encoded string
	manifestJSON, err := json.Marshal(appManifest)
	if err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}
	body := map[string]interface{}{
		"manifest": string(manifestJSON),
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/apps.manifest.create", bytes.NewReader(bodyJSON))
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

	var result manifestCreateResponse
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

	fmt.Fprintf(ios.Out, "\n%s Slack app created!\n\n", cs.Green("✓"))
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("App ID:"), result.AppID)
	fmt.Fprintf(ios.Out, "  %-16s %s\n", cs.Bold("Client ID:"), result.Credentials.ClientID)

	// Persist the AppID so `slackbuzz app update` can target this app later
	// without re-prompting, and the client credentials so it can re-run the
	// browser install. They come straight from Slack, so save them now —
	// even if the install below fails.
	app := oauthApp{ClientID: result.Credentials.ClientID, ClientSecret: result.Credentials.ClientSecret}
	saveClientCredentials(ios, result.AppID, app.ClientID, app.ClientSecret)

	installApp(ios, cs, result.AppID, app, false)

	fmt.Fprintf(ios.Out, "%s Setup complete! Try: %s\n", cs.Green("✓"), cs.Cyan("slackbuzz channel list"))
	return nil
}

// createViaWeb creates the app without a config token: Slack's "create from
// manifest" page opens pre-filled, and the user copies back the client
// credentials so the OAuth install can finish the job.
func createViaWeb(ios *iostreams.IOStreams, cs *iostreams.ColorScheme) error {
	if !ios.CanPrompt() {
		return fmt.Errorf("--web needs a terminal to paste the Client ID and Secret into — run it in your own terminal")
	}
	createURL, err := manifestCreateURL()
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "%s Opening Slack's app creator, pre-filled with the slackbuzz manifest.\n", cs.Blue("→"))
	fmt.Fprintln(ios.Out, "  1. Pick your workspace, click Next, then Create")
	fmt.Fprintln(ios.Out, "  2. You'll land on Basic Information — under App Credentials, copy the Client ID, then click Show and copy the Client Secret")
	fmt.Fprintf(ios.Out, "  If the page doesn't open, visit:\n  %s\n\n", createURL)
	_ = browser.Open(createURL)

	p := prompter.New(ios)
	clientID, err := p.Input("Client ID:", "")
	if err != nil {
		return fmt.Errorf("could not read client ID: %w", err)
	}
	clientSecret, err := p.Password("Client Secret:")
	if err != nil {
		return fmt.Errorf("could not read client secret: %w", err)
	}
	clientID, clientSecret = strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID and secret are both required")
	}

	// Credentials are typed by hand, so they're saved only once Slack
	// accepts them; installApp falls back to pasting tokens otherwise.
	installApp(ios, cs, "", oauthApp{ClientID: clientID, ClientSecret: clientSecret}, true)

	fmt.Fprintf(ios.Out, "%s Setup complete! Try: %s\n", cs.Green("✓"), cs.Cyan("slackbuzz auth status"))
	return nil
}

// manifestCreateURL is Slack's "create app from manifest" link, pre-filled.
func manifestCreateURL() (string, error) {
	manifestJSON, err := json.Marshal(appManifest)
	if err != nil {
		return "", fmt.Errorf("failed to encode manifest: %w", err)
	}
	return "https://api.slack.com/apps?new_app=1&manifest_json=" + url.QueryEscape(string(manifestJSON)), nil
}

type manifestCreateResponse struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	AppID       string `json:"app_id,omitempty"`
	Credentials struct {
		ClientID          string `json:"client_id"`
		ClientSecret      string `json:"client_secret"`
		VerificationToken string `json:"verification_token"`
		SigningSecret     string `json:"signing_secret"`
	} `json:"credentials,omitempty"`
	OAuthAuthorizeURL string `json:"oauth_authorize_url,omitempty"`
	Errors            []struct {
		Message string `json:"message"`
		Pointer string `json:"pointer"`
	} `json:"errors,omitempty"`
}
