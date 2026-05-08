package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/triptechtravel/slackbuzz-cli/internal/slackapi"
)

// AuthExpiredError indicates the token has been revoked or expired (401 response).
// Detail preserves the original API response body for debugging.
type AuthExpiredError struct {
	Detail string
}

func (e *AuthExpiredError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("authentication/permission error (HTTP 401): %s", e.Detail)
	}
	return "authentication expired or revoked. Run 'slackbuzz auth login' to re-authenticate"
}

// IsAuthExpired checks if an error is an AuthExpiredError.
func IsAuthExpired(err error) bool {
	var ae *AuthExpiredError
	return errors.As(err, &ae)
}

// IsAuthRelatedSlackError returns true if the given Slack API error code
// indicates an authentication problem. Uses the typed code constants from
// the slackapi package so codes can't drift between definitions.
func IsAuthRelatedSlackError(slackErr string) bool {
	switch slackErr {
	case slackapi.CodeInvalidAuth,
		slackapi.CodeTokenRevoked,
		slackapi.CodeAccountInactive,
		slackapi.CodeNotAuthed:
		return true
	}
	return false
}

// IsMissingScopeError reports whether err is (or wraps) Slack's
// missing_scope response. Uses errors.Is against the typed sentinel
// rather than string-matching the formatted error text — the sentinel
// matches by Code regardless of how Error() formats.
func IsMissingScopeError(err error) bool {
	return errors.Is(err, slackapi.ErrMissingScope)
}

// IsChannelNotFoundError reports whether err is (or wraps) Slack's
// channel_not_found response.
func IsChannelNotFoundError(err error) bool {
	return errors.Is(err, slackapi.ErrChannelNotFound)
}

// IsRateLimitedError reports whether err is (or wraps) Slack's
// ratelimited response.
func IsRateLimitedError(err error) bool {
	return errors.Is(err, slackapi.ErrRatelimited)
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
				"    slackbuzz app update   (updates an existing app's scopes)\n"+
				"  Workaround: slackbuzz send --as-bot %s <text>",
			target, target,
		)
	}

	if code := slackCode(err); code != "" {
		return fmt.Sprintf("failed to resolve %q: %s", target, code)
	}
	return fmt.Sprintf("failed to resolve %q: %s", target, err.Error())
}

// slackCode pulls Slack's machine-readable error code out of err. Returns
// "" when the error isn't from a slackapi call. Used by every formatter
// here so we look up canned messages by code, never by formatted text.
func slackCode(err error) string {
	var apiErr *slackapi.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// FormatError converts a Slack API error to a user-friendly message.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	if code := slackCode(err); code != "" {
		return SlackErrorMessage(code)
	}
	return err.Error()
}

// FormatSendError produces a context-aware error message for message
// send failures. The target is the original channel/user argument from
// the command. Falls back to the generic formatter for non-slackapi
// errors.
func FormatSendError(err error, target string) string {
	if err == nil {
		return ""
	}

	if IsMissingScopeError(err) {
		isDM := strings.HasPrefix(target, "@") || isUserID(target) ||
			(!strings.HasPrefix(target, "#") && !isChannelID(target))
		if isDM {
			return "Missing Slack scope for DMs. Ensure your user token has: im:write, chat:write\n" +
				"  Check scopes at api.slack.com/apps > OAuth & Permissions"
		}
		return "Missing scope for posting. Ensure your token has: chat:write\n" +
			"  Check scopes at api.slack.com/apps > OAuth & Permissions"
	}

	if code := slackCode(err); code != "" {
		return SlackErrorMessage(code)
	}
	return err.Error()
}

