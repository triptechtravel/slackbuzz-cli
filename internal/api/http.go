package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/triptechtravel/slackbuzz-cli/internal/build"
)

// rateLimitTransport injects a User-Agent header and handles rate limiting.
type rateLimitTransport struct {
	base http.RoundTripper
	rl   *RateLimiter
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", fmt.Sprintf("slack-cli/%s", build.Version))

	t.rl.Wait()

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	t.rl.Update(resp)

	// Detect 401 — read the body before closing so we preserve the actual
	// error message. Slack can return 401 for permission issues, not just
	// expired tokens.
	if resp.StatusCode == 401 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &AuthExpiredError{Detail: string(body)}
	}

	// Retry once on 429
	if t.rl.ShouldRetry(resp) {
		resp.Body.Close()
		t.rl.Wait()
		resp, err = t.base.RoundTrip(req)
		if err != nil {
			return resp, err
		}
		t.rl.Update(resp)
	}

	return resp, nil
}

// NewHTTPClient creates an http.Client with rate limiting and User-Agent.
func NewHTTPClient(rl *RateLimiter) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &rateLimitTransport{
			base: http.DefaultTransport,
			rl:   rl,
		},
	}
}
