package api

import (
	"errors"
	"fmt"
)

// AuthExpiredError indicates the token has been revoked or expired (401 response).
type AuthExpiredError struct{}

func (e *AuthExpiredError) Error() string {
	return "authentication expired or revoked. Run 'slackbuzz auth login' to re-authenticate"
}

// IsAuthExpired checks if an error is an AuthExpiredError.
func IsAuthExpired(err error) bool {
	var ae *AuthExpiredError
	return errors.As(err, &ae)
}

// IsAuthRelatedSlackError returns true if the Slack API error string indicates
// an authentication problem (invalid_auth, token_revoked, account_inactive, not_authed).
func IsAuthRelatedSlackError(slackErr string) bool {
	switch slackErr {
	case "invalid_auth", "token_revoked", "account_inactive", "not_authed":
		return true
	}
	return false
}

// SlackErrorMessage maps Slack API error codes to user-friendly messages.
func SlackErrorMessage(slackErr string) string {
	messages := map[string]string{
		"channel_not_found":    "Channel not found. Check the channel name or ID.",
		"not_in_channel":       "Bot is not a member of this channel. Invite the bot first.",
		"is_archived":          "Channel is archived and cannot receive messages.",
		"msg_too_long":         "Message is too long. Slack messages are limited to 40,000 characters.",
		"no_text":              "Message text cannot be empty.",
		"rate_limited":         "Rate limited. Please wait and try again.",
		"invalid_auth":         "Authentication failed. Run 'slackbuzz auth login' to re-authenticate.",
		"account_inactive":     "Account is inactive or token has been revoked.",
		"token_revoked":        "Token has been revoked. Run 'slackbuzz auth login' to re-authenticate.",
		"missing_scope":        "Token is missing required scopes. Check your Slack app configuration.",
		"not_authed":           "Not authenticated. Run 'slackbuzz auth login' to authenticate.",
		"user_not_found":       "User not found. Check the username or ID.",
		"thread_not_found":     "Thread not found. Check the thread timestamp.",
		"too_many_attachments": "Too many attachments. Reduce the number and try again.",
	}

	if msg, ok := messages[slackErr]; ok {
		return msg
	}
	return fmt.Sprintf("Slack API error: %s", slackErr)
}

// FormatError converts a Slack API error to a user-friendly message.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return SlackErrorMessage(err.Error())
}
