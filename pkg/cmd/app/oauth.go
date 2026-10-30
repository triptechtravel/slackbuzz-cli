package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/triptechtravel/slackbuzz-cli/internal/browser"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
)

// oauthRedirectURL is the redirect URL registered in the app manifest
// (the generated manifest.go references it). Slack only accepts HTTPS
// redirect URLs, so it points at a static relay page on the docs site
// (docs/public/oauth/callback/index.html) that forwards the
// ?code=…&state=… query to the CLI's loopback listener. The listener's
// port travels in `state`; the code is useless without the app's client
// secret, which never leaves this machine.
const oauthRedirectURL = "https://triptechtravel.github.io/slackbuzz-cli/oauth/callback/"

// oauthTokenURL, openBrowser and oauthTimeout are vars so tests can stub them.
var (
	oauthTokenURL = "https://slack.com/api/oauth.v2.access"
	openBrowser   = browser.Open
	oauthTimeout  = 5 * time.Minute
)

// oauthApp identifies the app to install. TeamID is optional; when set,
// Slack skips the workspace picker (non-distributed apps can only be
// installed on their own workspace).
type oauthApp struct {
	ClientID     string
	ClientSecret string
	TeamID       string
}

// oauthResult is the subset of oauth.v2.access we use.
type oauthResult struct {
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
	AppID       string `json:"app_id"`
	AccessToken string `json:"access_token"` // bot token (xoxb-)
	AuthedUser  struct {
		AccessToken string `json:"access_token"` // user token (xoxp-)
	} `json:"authed_user"`
}

// installResult reports what the OAuth install managed to store, so a
// partial result can be finished by pasting the missing token.
type installResult struct {
	AppID      string
	StoredBot  bool
	StoredUser bool
}

// newOAuthState returns "<port>-<32 hex chars>". The relay page parses
// this exact shape (see TestOAuthStateMatchesRelay).
func newOAuthState(port int) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return strconv.Itoa(port) + "-" + hex.EncodeToString(nonce), nil
}

// runOAuthInstall installs the app via Slack's OAuth flow and stores both
// tokens, so the user clicks Allow instead of copying tokens by hand.
// The result is meaningful even with an error when tokens were partly stored.
func runOAuthInstall(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, app oauthApp) (installResult, error) {
	var out installResult

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return out, fmt.Errorf("could not start local callback listener: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	state, err := newOAuthState(port)
	if err != nil {
		ln.Close()
		return out, err
	}

	params := url.Values{
		"client_id":    {app.ClientID},
		"scope":        {strings.Join(botScopes, ",")},
		"user_scope":   {strings.Join(userScopes, ",")},
		"redirect_uri": {oauthRedirectURL},
		"state":        {state},
	}
	if app.TeamID != "" {
		params.Set("team", app.TeamID)
	}
	authURL := "https://slack.com/oauth/v2/authorize?" + params.Encode()

	code, err := waitForCode(ios, cs, ln, authURL, state)
	if err != nil {
		return out, err
	}

	res, err := exchangeCode(app.ClientID, app.ClientSecret, code)
	if err != nil {
		return out, err
	}
	out.AppID = res.AppID

	if res.AccessToken != "" {
		out.StoredBot, out.StoredUser = storeDetectedToken(ios, cs, res.AccessToken, out.StoredBot, out.StoredUser)
	}
	if res.AuthedUser.AccessToken != "" {
		out.StoredBot, out.StoredUser = storeDetectedToken(ios, cs, res.AuthedUser.AccessToken, out.StoredBot, out.StoredUser)
	}
	if !out.StoredBot || !out.StoredUser {
		return out, fmt.Errorf("slack completed the install but not both tokens were stored")
	}
	return out, nil
}

// waitForCode serves the loopback callback, opens the authorize URL and
// blocks until Slack (via the relay) delivers a code, an error, or we time out.
func waitForCode(ios *iostreams.IOStreams, cs *iostreams.ColorScheme, ln net.Listener, authURL, state string) (string, error) {
	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	var once sync.Once
	host := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// Host check defeats DNS-rebinding pages; state ties the request
		// to this run (stale tab, or something else poking the port).
		if r.Host != host || q.Get("state") != state {
			http.Error(w, "This sign-in link doesn't match the running slackbuzz command. Return to the terminal and try again.", http.StatusBadRequest)
			return
		}
		res := result{code: q.Get("code")}
		if e := q.Get("error"); e != "" {
			res = result{err: fmt.Errorf("slack authorization failed: %s", e)}
		} else if res.code == "" {
			res = result{err: fmt.Errorf("slack redirect had no code")}
		}

		accepted := false
		once.Do(func() {
			accepted = true
			done <- res
		})

		msg := "slackbuzz is authorized. You can close this tab and return to your terminal."
		switch {
		case !accepted:
			msg = "This sign-in was already handled. Check your terminal."
		case res.err != nil:
			msg = res.err.Error()
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!doctype html><title>slackbuzz</title><body style=\"font-family:system-ui;margin:3rem\"><p>%s</p></body>", html.EscapeString(msg))
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	fmt.Fprintf(ios.Out, "\n%s Opening Slack in your browser — click %s to install and authorize.\n", cs.Blue("→"), cs.Bold("Allow"))
	fmt.Fprintf(ios.Out, "  If it doesn't open, visit:\n  %s\n\n", authURL)
	_ = openBrowser(authURL)
	fmt.Fprintf(ios.Out, "Waiting for authorization (up to %d minutes; Ctrl-C to cancel)...\n", int(oauthTimeout.Minutes()))
	fmt.Fprintln(ios.Out, "  If Slack shows \"Request to install\", a workspace admin must approve the app first — run this again once they have.")

	select {
	case r := <-done:
		return r.code, r.err
	case <-time.After(oauthTimeout):
		return "", fmt.Errorf("timed out waiting for Slack authorization (if Slack's redirect showed an error page, check %s loads)", oauthRedirectURL)
	}
}

func exchangeCode(clientID, clientSecret, code string) (*oauthResult, error) {
	form := url.Values{
		"code":         {code},
		"redirect_uri": {oauthRedirectURL},
	}
	req, err := http.NewRequest("POST", oauthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Slack prefers client credentials via Basic auth over form params.
	req.SetBasicAuth(clientID, clientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	var res oauthResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse token exchange response: %w", err)
	}
	if !res.OK {
		if res.Error == "invalid_team_for_non_distributed_app" {
			return nil, fmt.Errorf("token exchange failed: %s — pick the workspace the app was created in", res.Error)
		}
		return nil, fmt.Errorf("token exchange failed: %s", res.Error)
	}
	return &res, nil
}
