package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/triptechtravel/slackbuzz-cli/internal/auth"
	"github.com/triptechtravel/slackbuzz-cli/internal/clipboard"
	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
	"github.com/triptechtravel/slackbuzz-cli/internal/prompter"
)

// configTokenEnv lets scripts and agents supply the App Configuration Token
// without a TTY or a file. It overrides a saved token and is never saved.
const configTokenEnv = "SLACKBUZZ_CONFIG_TOKEN"

// configTokenPrefix is what an App Configuration *access* token starts
// with (refresh tokens are xoxe-1-…, and won't authenticate the manifest API).
const configTokenPrefix = "xoxe.xoxp-"

// tokenSource is how the user asked us to read the App Configuration Token.
type tokenSource struct {
	withToken     bool // read from stdin
	fromClipboard bool // read from the system clipboard
}

// errNoConfigToken means no non-interactive source supplied a config token.
// Callers decide whether to prompt or offer the browser flow instead.
var errNoConfigToken = errors.New("no App Configuration Token")

// loadAuth reads auth.yml. Unlike ignoring the error, this refuses to carry
// on with an empty config that a later Save would write over a corrupt —
// but possibly token-bearing — file.
func loadAuth() (*config.AuthConfig, error) {
	ac, err := config.LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w (fix or remove it, then retry)", config.AuthFile(), err)
	}
	return ac, nil
}

// configTokenFrom resolves the config token from stdin, clipboard, env or the
// saved config, in that order. It never prompts. save reports whether the
// token should be persisted (not for env or an already-saved token).
func configTokenFrom(ios *iostreams.IOStreams, src tokenSource, ac *config.AuthConfig) (token string, save bool, err error) {
	switch {
	case src.withToken:
		scanner := bufio.NewScanner(ios.In)
		if !scanner.Scan() {
			return "", false, fmt.Errorf("failed to read config token from stdin")
		}
		token, save = strings.TrimSpace(scanner.Text()), true
	case src.fromClipboard:
		t, err := clipboard.Read()
		if err != nil {
			return "", false, fmt.Errorf("could not read config token from clipboard: %w", err)
		}
		fmt.Fprintln(ios.Out, "Using config token from clipboard")
		token, save = t, true
	case strings.TrimSpace(os.Getenv(configTokenEnv)) != "":
		fmt.Fprintf(ios.Out, "Using config token from $%s\n", configTokenEnv)
		token = strings.TrimSpace(os.Getenv(configTokenEnv))
	case ac.ConfigToken != "":
		fmt.Fprintf(ios.Out, "Using saved config token (from %s)\n", config.AuthFile())
		token = ac.ConfigToken
	default:
		return "", false, errNoConfigToken
	}
	if err := checkConfigToken(token); err != nil {
		return "", false, err
	}
	return token, save, nil
}

// checkConfigToken catches the common mix-ups (a bot/user token, or the
// config *refresh* token) before they reach Slack. Never echoes the value.
func checkConfigToken(token string) error {
	if token == "" {
		return fmt.Errorf("config token cannot be empty")
	}
	if !strings.HasPrefix(token, configTokenPrefix) {
		return fmt.Errorf("that isn't an App Configuration access token (expected %s…; the refresh token starts xoxe-1-)", configTokenPrefix)
	}
	return nil
}

// promptConfigToken explains where the token lives and asks for it.
func promptConfigToken(ios *iostreams.IOStreams, cs *iostreams.ColorScheme) (string, error) {
	fmt.Fprintln(ios.Out, "Where to find your App Configuration Token:")
	fmt.Fprintf(ios.Out, "  1. Go to %s (the apps LIST page, not inside an app)\n", cs.Cyan("https://api.slack.com/apps"))
	fmt.Fprintln(ios.Out, "  2. Scroll to the bottom — section is titled \"Your App Configuration Tokens\"")
	fmt.Fprintln(ios.Out, "  3. Click \"Generate Token\" next to your workspace, copy the Access Token (xoxe.xoxp-...)")
	fmt.Fprintln(ios.Out)
	t, err := prompter.New(ios).Password("Paste your App Configuration Token:")
	if err != nil {
		return "", fmt.Errorf("could not read token: %w", err)
	}
	t = strings.TrimSpace(t)
	if err := checkConfigToken(t); err != nil {
		return "", err
	}
	return t, nil
}

// nonInteractiveTokenHelp is shown when there's no TTY to prompt on.
func nonInteractiveTokenHelp() error {
	return fmt.Errorf("no App Configuration Token and no terminal to prompt on.\n"+
		"Supply one with --clipboard (copy it first), --with-token (stdin), or $%s", configTokenEnv)
}

// saveClientCredentials persists the app's OAuth client credentials so a
// later `app update` can reinstall through the browser. The secret goes to
// the keyring when available. Loads fresh: tokens may have just been written.
func saveClientCredentials(ios *iostreams.IOStreams, appID, clientID, clientSecret string) {
	ac, err := loadAuth()
	if err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save client credentials: %v\n", err)
		return
	}
	if appID != "" {
		ac.AppID = appID
	}
	ac.ClientID = clientID
	if err := ac.Save(); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save client ID: %v\n", err)
		return
	}
	if err := auth.StoreClientSecret(clientSecret); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: could not save client secret: %v\n", err)
	}
}

// installApp gets bot + user tokens for the app: via the browser OAuth flow
// when we have its client credentials, falling back to pasting tokens.
// saveOnSuccess persists credentials that weren't already known-good (typed
// by the user) once Slack accepts them. It returns the app ID Slack reported.
func installApp(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, appID string, app oauthApp, saveOnSuccess bool) string {
	var res installResult
	if app.ClientID != "" && app.ClientSecret != "" {
		var err error
		res, err = runOAuthInstall(ios, cs, app)
		if res.AppID != "" && appID != "" && res.AppID != appID {
			fmt.Fprintf(ios.ErrOut, "%s Slack installed app %s, not %s — the saved client credentials belong to a different app.\n", cs.Yellow("!"), res.AppID, appID)
		}
		if err == nil {
			if saveOnSuccess {
				saveClientCredentials(ios, res.AppID, app.ClientID, app.ClientSecret)
			}
			return res.AppID
		}
		fmt.Fprintf(ios.ErrOut, "%s %v\n", cs.Yellow("!"), err)
		fmt.Fprintln(ios.ErrOut, "Falling back to pasting tokens.")
	}
	pasteTokens(ios, cs, appID, res.StoredBot, res.StoredUser)
	return res.AppID
}

// pasteTokens is the manual flow: install from the app page, then paste
// whichever of the bot and user tokens isn't stored yet. Token type is
// detected from the prefix.
func pasteTokens(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, appID string, storedBot, storedUser bool) {
	if storedBot && storedUser {
		return
	}
	if !ios.CanPrompt() {
		fmt.Fprintln(ios.ErrOut, "No terminal to prompt for tokens. Install the app, then run: slackbuzz auth login --clipboard")
		return
	}
	fmt.Fprintln(ios.Out)
	if appID != "" {
		fmt.Fprintln(ios.Out, cs.Bold("Install the app, then copy its tokens from OAuth & Permissions:"))
		fmt.Fprintf(ios.Out, "  %s  (install)\n", cs.Cyan(fmt.Sprintf("https://api.slack.com/apps/%s/install-on-team", appID)))
		fmt.Fprintf(ios.Out, "  %s  (tokens)\n", cs.Cyan(fmt.Sprintf("https://api.slack.com/apps/%s/oauth", appID)))
	} else {
		fmt.Fprintln(ios.Out, cs.Bold("Install the app, then copy its tokens from OAuth & Permissions."))
	}
	fmt.Fprintln(ios.Out, "  • Bot User OAuth Token (xoxb-...) — top of page")
	fmt.Fprintln(ios.Out, "  • User OAuth Token (xoxp-...) — just below it")
	fmt.Fprintln(ios.Out, "Token type is auto-detected from prefix. Press enter to skip.")
	fmt.Fprintln(ios.Out)

	p := prompter.New(ios)
	for !storedBot || !storedUser {
		var label string
		switch {
		case !storedBot && !storedUser:
			label = "Paste a token (bot or user):"
		case !storedBot:
			label = "Paste your bot (xoxb-) token:"
		default:
			label = "Paste your user (xoxp-) token:"
		}
		token, err := p.Password(label)
		if err != nil {
			return
		}
		token = strings.TrimSpace(token)
		if token == "" {
			if !storedBot || !storedUser {
				fmt.Fprintln(ios.Out, "Skipped. Add the rest later with: slackbuzz auth login")
			}
			return
		}
		// Re-pasting a token we already have (stale clipboard) would
		// silently overwrite it and leave the other side unset.
		if (storedBot && auth.IsBotToken(token)) || (storedUser && auth.IsUserToken(token)) {
			fmt.Fprintln(ios.ErrOut, "That's the same token type you already pasted. Check your clipboard and try again.")
			continue
		}
		storedBot, storedUser = storeDetectedToken(ios, cs, token, storedBot, storedUser)
	}
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
