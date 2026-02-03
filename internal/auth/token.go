package auth

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

// TokenInfo holds the result of validating a token via auth.test.
type TokenInfo struct {
	UserID   string
	User     string
	TeamID   string
	Team     string
	BotID    string
	IsBot    bool
}

// ValidateToken checks that a token is valid by calling Slack's auth.test.
func ValidateToken(token string) (*TokenInfo, error) {
	client := slack.New(token)

	resp, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	info := &TokenInfo{
		UserID: resp.UserID,
		User:   resp.User,
		TeamID: resp.TeamID,
		Team:   resp.Team,
		BotID:  resp.BotID,
	}

	// Bot tokens start with xoxb- and have a BotID
	info.IsBot = strings.HasPrefix(token, "xoxb-") || resp.BotID != ""

	return info, nil
}

// IsBotToken returns true if the token looks like a bot token.
func IsBotToken(token string) bool {
	return strings.HasPrefix(token, "xoxb-")
}

// IsUserToken returns true if the token looks like a user token.
func IsUserToken(token string) bool {
	return strings.HasPrefix(token, "xoxp-")
}

// DetectTokenType returns "bot" or "user" based on prefix, or "unknown".
func DetectTokenType(token string) string {
	switch {
	case IsBotToken(token):
		return "bot"
	case IsUserToken(token):
		return "user"
	default:
		return "unknown"
	}
}
