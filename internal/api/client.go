package api

import (
	"context"
	"encoding/json"

	"github.com/triptechtravel/slackbuzz-cli/internal/build"
	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
)

// Client is the unified Slack API client used across commands.
//
// Backed entirely by our generated slackapi package (transport +
// per-method typed wrappers). slack-go is no longer in the dependency
// tree — every operation is generated from the OpenAPI spec or
// hand-augmented under internal/slackapi/.
type Client struct {
	API         *slackapi.Client
	RateLimiter *RateLimiter
	token       string
}

// NewClient creates a new Slack API client with the given token.
func NewClient(token string) *Client {
	rl := NewRateLimiter()
	httpClient := NewHTTPClient(rl)

	apiClient := slackapi.NewWithHTTP(token, httpClient)

	// Surface Version usage so the build tag isn't optimised away.
	_ = build.Version

	return &Client{
		API:         apiClient,
		RateLimiter: rl,
		token:       token,
	}
}

// Token returns the API token used by this client.
func (c *Client) Token() string {
	return c.token
}

// AuthUserID calls auth.test to get the user ID for this client's token.
// Returns empty string on failure.
func (c *Client) AuthUserID() string {
	resp, err := slackapi.AuthTest(context.Background(), c.API)
	if err != nil {
		return ""
	}
	var info struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(resp.Raw, &info); err != nil {
		return ""
	}
	return info.UserID
}
