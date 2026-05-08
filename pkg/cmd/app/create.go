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
	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/browser"
	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
	"github.com/triptechtravel/slackbuzz-cli/pkg/cmdutil"
)

type createOptions struct {
	factory    *cmdutil.Factory
	configToken string
	withToken   bool
}

// NewCmdCreate returns the "app create" command.
func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{factory: f}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Slack app with required scopes",
		Long: `Create a new Slack app pre-configured with all scopes needed by the CLI.

This command uses the Slack apps.manifest API to create an app. You need
an App Configuration Token from: api.slack.com/apps > Your App Configuration Tokens.

After the app is created, it opens the install page in your browser.
Once installed, copy the Bot and User OAuth tokens for 'slackbuzz auth login'.`,
		Example: `  # Interactive (prompts for config token)
  slackbuzz app create

  # Pipe config token for CI
  echo "xoxe.xoxp-..." | slackbuzz app create --with-token`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return createRun(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, "Read config token from stdin")

	return cmd
}

func createRun(opts *createOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Get config token — check saved token first
	var configToken string

	ac, _ := config.LoadAuth()
	if ac.ConfigToken != "" && !opts.withToken {
		configToken = ac.ConfigToken
		fmt.Fprintf(ios.Out, "Using saved config token (from %s)\n", config.AuthFile())
	} else if opts.withToken {
		scanner := bufio.NewScanner(ios.In)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read config token from stdin")
		}
		configToken = strings.TrimSpace(scanner.Text())
	} else {
		fmt.Fprintln(ios.Out, "To create a Slack app, you need an App Configuration Token.")
		fmt.Fprintln(ios.Out, "Where to find it:")
		fmt.Fprintf(ios.Out, "  1. Go to %s (the apps LIST page)\n", cs.Cyan("https://api.slack.com/apps"))
		fmt.Fprintln(ios.Out, "  2. Scroll to the bottom — section is titled \"Your App Configuration Tokens\"")
		fmt.Fprintln(ios.Out, "  3. Click \"Generate Token\" next to your workspace, copy the Access Token (xoxe.xoxp-...)")
		fmt.Fprintln(ios.Out, "  Note: this is a workspace-level admin token — NOT inside any individual app's credentials.")
		fmt.Fprintln(ios.Out)

		p := prompter.New(ios)
		var err error
		configToken, err = p.Password("Paste your App Configuration Token:")
		if err != nil {
			return fmt.Errorf("could not read token: %w", err)
		}
		configToken = strings.TrimSpace(configToken)
	}

	if configToken == "" {
		return fmt.Errorf("config token cannot be empty")
	}

	// Save the config token immediately so it's not lost if the API call fails
	ac.ConfigToken = configToken
	if err := ac.Save(); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save config token: %v\n", err)
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
	// without re-prompting.
	ac.AppID = result.AppID
	if saveErr := ac.Save(); saveErr != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save app ID: %v\n", saveErr)
	}

	// Open the app install page
	installURL := fmt.Sprintf("https://api.slack.com/apps/%s/install-on-team", result.AppID)
	fmt.Fprintf(ios.Out, "\n%s Opening install page in your browser...\n", cs.Blue("→"))
	fmt.Fprintf(ios.Out, "  %s\n\n", installURL)
	_ = browser.Open(installURL)

	fmt.Fprintln(ios.Out, "Install the app to your workspace, then come back here to paste the tokens.")
	fmt.Fprintln(ios.Out)

	// Prompt for tokens inline — auto-detect type from prefix
	p := prompter.New(ios)
	storedBot := false
	storedUser := false

	oauthURL := fmt.Sprintf("https://api.slack.com/apps/%s/oauth", result.AppID)
	fmt.Fprintln(ios.Out, cs.Bold("After installing, grab the OAuth tokens from the OAuth & Permissions page:"))
	fmt.Fprintf(ios.Out, "  %s\n", cs.Cyan(oauthURL))
	fmt.Fprintln(ios.Out, "  • Bot User OAuth Token (xoxb-...) — top of page")
	fmt.Fprintln(ios.Out, "  • User OAuth Token (xoxp-...) — just below it")
	fmt.Fprintln(ios.Out, "Token type is auto-detected from prefix.")
	fmt.Fprintln(ios.Out)

	// First token (required — bot token)
	token1, err := p.Password("Paste your first token:")
	if err != nil {
		fmt.Fprintln(ios.ErrOut, "Skipping tokens. You can add them later with: slackbuzz auth login")
	} else {
		token1 = strings.TrimSpace(token1)
		if token1 != "" {
			storedBot, storedUser = storeDetectedToken(ios, cs, token1, storedBot, storedUser)
		}
	}

	// Second token (optional — user token for search)
	if !storedBot || !storedUser {
		missing := "user (xoxp-)"
		if !storedBot {
			missing = "bot (xoxb-)"
		}
		fmt.Fprintf(ios.Out, "(Optional) Paste your %s token for full functionality:\n", missing)
		token2, err := p.Password("Token (enter to skip):")
		if err == nil {
			token2 = strings.TrimSpace(token2)
			if token2 != "" {
				storedBot, storedUser = storeDetectedToken(ios, cs, token2, storedBot, storedUser)
			}
		}
	}

	fmt.Fprintf(ios.Out, "%s Setup complete! Try: %s\n", cs.Green("✓"), cs.Cyan("slackbuzz channel list"))

	return nil
}

// storeDetectedToken validates and stores a token based on its detected type (xoxb- or xoxp-).
// Returns updated storedBot, storedUser flags.
func storeDetectedToken(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, token string, storedBot, storedUser bool) (bool, bool) {
	tokenType := auth.DetectTokenType(token)
	if tokenType == "unknown" {
		fmt.Fprintf(ios.ErrOut, "Warning: unrecognized token prefix. Expected xoxb- (bot) or xoxp- (user). Skipping.\n")
		return storedBot, storedUser
	}

	info, err := auth.ValidateToken(token)
	if err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: token validation failed: %v\n", err)
		return storedBot, storedUser
	}

	if tokenType == "bot" {
		if storeErr := auth.StoreBotToken(token); storeErr != nil {
			fmt.Fprintf(ios.ErrOut, "Warning: failed to store bot token: %v\n", storeErr)
		} else {
			_ = auth.StoreTeamInfo(info.TeamID, info.Team)
			_ = auth.StoreBotUserInfo(info.UserID, info.User)
			fmt.Fprintf(ios.Out, "%s Bot token saved (%s on %s)\n\n", cs.Green("✓"), cs.Bold(info.User), cs.Bold(info.Team))
			storedBot = true
		}
	} else {
		if storeErr := auth.StoreUserToken(token); storeErr != nil {
			fmt.Fprintf(ios.ErrOut, "Warning: failed to store user token: %v\n", storeErr)
		} else {
			_ = auth.StoreUserInfo(info.UserID, info.User)
			fmt.Fprintf(ios.Out, "%s User token saved (%s)\n\n", cs.Green("✓"), cs.Bold(info.User))
			storedUser = true
		}
	}

	return storedBot, storedUser
}

type manifestCreateResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	AppID string `json:"app_id,omitempty"`
	Credentials struct {
		ClientID          string `json:"client_id"`
		ClientSecret      string `json:"client_secret"`
		VerificationToken string `json:"verification_token"`
		SigningSecret     string `json:"signing_secret"`
	} `json:"credentials,omitempty"`
	OAuthAuthorizeURL string `json:"oauth_authorize_url,omitempty"`
	Errors []struct {
		Message string `json:"message"`
		Pointer string `json:"pointer"`
	} `json:"errors,omitempty"`
}
