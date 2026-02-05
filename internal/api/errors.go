package api

import (
	"errors"
	"fmt"
	"strings"
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

// IsMissingScopeError checks if an error is (or wraps) a Slack "missing_scope" error.
func IsMissingScopeError(err error) bool {
	if err == nil {
		return false
	}
	// Walk the error chain
	for e := err; e != nil; e = errors.Unwrap(e) {
		if e.Error() == "missing_scope" {
			return true
		}
	}
	return strings.Contains(err.Error(), "missing_scope")
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
		"missing_scope":        "Token is missing required scopes. Run 'slackbuzz auth status' to check capabilities, or try --as-bot.",
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

// FormatResolveError produces an actionable error message when channel/user
// resolution fails. It detects missing_scope errors and suggests fixes.
func FormatResolveError(err error, target string) string {
	if err == nil {
		return ""
	}

	if IsMissingScopeError(err) {
		return fmt.Sprintf(
			"Channel resolution for %q failed: missing_scope (likely channels:read).\n"+
				"  Fix: Re-install your Slack app with updated scopes, or run:\n"+
				"    slackbuzz app create   (creates app with correct scopes)\n"+
				"  Workaround: slackbuzz send --as-bot %s <text>",
			target, target,
		)
	}

	return fmt.Sprintf("failed to resolve %q: %s", target, err.Error())
}

// FormatError converts a Slack API error to a user-friendly message.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return SlackErrorMessage(err.Error())
}

// FormatSendError produces a context-aware error message for message send failures.
// The target is the original channel/user argument from the command.
func FormatSendError(err error, target string) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()

	if errStr == "missing_scope" {
		isDM := strings.HasPrefix(target, "@") || isUserID(target) ||
			(!strings.HasPrefix(target, "#") && !isChannelID(target))
		if isDM {
			return "Missing Slack scope for DMs. Ensure your user token has: im:write, chat:write\n" +
				"  Check scopes at api.slack.com/apps > OAuth & Permissions"
		}
		return "Missing scope for posting. Ensure your token has: chat:write\n" +
			"  Check scopes at api.slack.com/apps > OAuth & Permissions"
	}

	return SlackErrorMessage(errStr)
}

