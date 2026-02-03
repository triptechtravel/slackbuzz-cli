package api

import (
	"github.com/slack-go/slack"
	"github.com/triptechtravel/slackbuzz-cli/internal/build"
)

// Client wraps the slack-go client with rate limiting.
type Client struct {
	Slack       *slack.Client
	RateLimiter *RateLimiter
	token       string
}

// NewClient creates a new Slack API client with the given token.
func NewClient(token string) *Client {
	rl := NewRateLimiter()

	slackClient := slack.New(
		token,
		slack.OptionHTTPClient(NewHTTPClient(rl)),
		slack.OptionAppLevelToken(""),
	)

	// Set a custom user agent header on requests — slack-go doesn't expose
	// a direct "user-agent" option but we apply it via the custom transport.
	_ = build.Version

	return &Client{
		Slack:       slackClient,
		RateLimiter: rl,
		token:       token,
	}
}

// Token returns the API token used by this client.
func (c *Client) Token() string {
	return c.token
}
