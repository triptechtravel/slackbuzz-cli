package app

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
)

// startWait runs waitForCode against a fresh listener and returns the
// callback base URL plus a channel with the outcome.
func startWait(t *testing.T, state string) (string, <-chan [2]interface{}) {
	t.Helper()
	opened := make(chan string, 1)
	orig := openBrowser
	openBrowser = func(u string) error { opened <- u; return nil }
	t.Cleanup(func() { openBrowser = orig })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ios := iostreams.Test()
	out := make(chan [2]interface{}, 1)
	go func() {
		code, err := waitForCode(ios, ios.ColorScheme(), ln, "https://slack.example/authorize", state)
		out <- [2]interface{}{code, err}
	}()
	<-opened // listener is serving once the browser would open
	return "http://" + ln.Addr().String() + "/callback", out
}

func TestWaitForCodeDeliversCode(t *testing.T) {
	base, out := startWait(t, "1234-abc")

	// A request with the wrong state is rejected and doesn't end the wait.
	resp, err := http.Get(base + "?state=other&code=evil")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong-state status = %d, want 400", resp.StatusCode)
	}

	resp, err = http.Get(base + "?state=1234-abc&code=good")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	r := <-out
	if r[1] != nil || r[0] != "good" {
		t.Fatalf("got code=%v err=%v, want good", r[0], r[1])
	}
}

func TestWaitForCodeReportsDenial(t *testing.T) {
	base, out := startWait(t, "1234-abc")
	resp, err := http.Get(base + "?state=1234-abc&error=access_denied")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	r := <-out
	if err, _ := r[1].(error); err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("err = %v, want access_denied", r[1])
	}
}

func TestExchangeCode(t *testing.T) {
	var got url.Values
	var user, pass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		user, pass, _ = r.BasicAuth()
		_, _ = w.Write([]byte(`{"ok":true,"app_id":"A1","access_token":"xoxb-1","authed_user":{"access_token":"xoxp-1"}}`))
	}))
	defer srv.Close()
	orig := oauthTokenURL
	oauthTokenURL = srv.URL
	defer func() { oauthTokenURL = orig }()

	res, err := exchangeCode("cid", "secret", "code1")
	if err != nil {
		t.Fatal(err)
	}
	if res.AppID != "A1" || res.AccessToken != "xoxb-1" || res.AuthedUser.AccessToken != "xoxp-1" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if user != "cid" || pass != "secret" {
		t.Fatalf("basic auth = %q/%q, want cid/secret", user, pass)
	}
	if got.Get("client_secret") != "" || got.Get("code") != "code1" || got.Get("redirect_uri") != oauthRedirectURL {
		t.Fatalf("unexpected form: %v", got)
	}
}

func TestExchangeCodeSlackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_code"}`))
	}))
	defer srv.Close()
	orig := oauthTokenURL
	oauthTokenURL = srv.URL
	defer func() { oauthTokenURL = orig }()

	if _, err := exchangeCode("cid", "secret", "x"); err == nil || !strings.Contains(err.Error(), "invalid_code") {
		t.Fatalf("err = %v, want invalid_code", err)
	}
}

func TestManifestRegistersRedirect(t *testing.T) {
	oc := appManifest["oauth_config"].(map[string]interface{})
	urls := oc["redirect_urls"].([]string)
	if len(urls) != 1 || urls[0] != oauthRedirectURL {
		t.Fatalf("redirect_urls = %v, want [%s]", urls, oauthRedirectURL)
	}
}

func TestWaitForCodeRejectsForeignHost(t *testing.T) {
	base, out := startWait(t, "1234-abc")
	req, _ := http.NewRequest("GET", base+"?state=1234-abc&code=x", nil)
	req.Host = "attacker.example:80"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign host status = %d, want 400", resp.StatusCode)
	}
	select {
	case r := <-out:
		t.Fatalf("wait ended on a foreign-host request: %v", r)
	default:
	}
	// Unblock the waiter so the goroutine ends.
	resp, err = http.Get(base + "?state=1234-abc&code=ok")
	if err == nil {
		resp.Body.Close()
	}
	<-out
}

func TestWaitForCodeTimesOut(t *testing.T) {
	orig := oauthTimeout
	oauthTimeout = 50 * time.Millisecond
	t.Cleanup(func() { oauthTimeout = orig })

	_, out := startWait(t, "1234-abc")
	r := <-out
	if err, _ := r[1].(error); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timeout", r[1])
	}
}

// The relay page parses state with its own regex; keep the two in step.
func TestOAuthStateMatchesRelay(t *testing.T) {
	page, err := os.ReadFile("../../../docs/public/oauth/callback/index.html")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`/(\^[^/]+\$)/\.exec`).FindSubmatch(page)
	if m == nil {
		t.Fatal("state regex not found in relay page")
	}
	relay := regexp.MustCompile(string(m[1]))
	for _, port := range []int{1024, 49152, 65535} {
		state, err := newOAuthState(port)
		if err != nil {
			t.Fatal(err)
		}
		if !relay.MatchString(state) {
			t.Errorf("relay regex %s rejects state %q", m[1], state)
		}
	}
}

// Windows' URL handler has historically truncated at ~2083 characters.
func TestManifestCreateURLFitsWindows(t *testing.T) {
	u, err := manifestCreateURL()
	if err != nil {
		t.Fatal(err)
	}
	if len(u) > 2000 {
		t.Fatalf("create URL is %d chars; Windows may truncate it", len(u))
	}
}
