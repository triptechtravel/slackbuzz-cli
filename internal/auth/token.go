package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
)

// TokenInfo holds the result of validating a token via auth.test.
type TokenInfo struct {
	UserID string
	User   string
	TeamID string
	Team   string
	BotID  string
	IsBot  bool
}

// ValidateToken checks that a token is valid by calling Slack's auth.test.
func ValidateToken(token string) (*TokenInfo, error) {
	client := slackapi.New(token)

	resp, err := slackapi.AuthTest(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	var raw struct {
		UserID string `json:"user_id"`
		User   string `json:"user"`
		TeamID string `json:"team_id"`
		Team   string `json:"team"`
		BotID  string `json:"bot_id"`
	}
	if err := json.Unmarshal(resp.Raw, &raw); err != nil {
		return nil, fmt.Errorf("decode auth.test: %w", err)
	}

	info := &TokenInfo{
		UserID: raw.UserID,
		User:   raw.User,
		TeamID: raw.TeamID,
		Team:   raw.Team,
		BotID:  raw.BotID,
	}

	// Bot tokens start with xoxb- and have a BotID
	info.IsBot = strings.HasPrefix(token, "xoxb-") || raw.BotID != ""

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
