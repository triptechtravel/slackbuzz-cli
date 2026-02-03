package text

import (
	"fmt"
	"os"
	"strings"
)

// SlackDeeplink generates a clickable slack:// deeplink for opening items
// directly in the Slack desktop/mobile app.
//
// Supported link types:
//   - Channel:  SlackDeeplink(teamID, channelID, "", "")
//   - Message:  SlackDeeplink(teamID, channelID, messageTS, "")
//   - Thread:   SlackDeeplink(teamID, channelID, "", threadTS)
func SlackDeeplink(teamID, channelID, messageTS, threadTS string) string {
	if teamID == "" || channelID == "" {
		return ""
	}

	base := fmt.Sprintf("slack://channel?team=%s&id=%s", teamID, channelID)

	if threadTS != "" {
		return base + "&message=" + threadTS
	}
	if messageTS != "" {
		return base + "&message=" + messageTS
	}
	return base
}

// SlackWebLink generates a web URL that redirects to the Slack app.
// This works as a fallback when the slack:// protocol isn't registered.
func SlackWebLink(teamID, channelID, messageTS string) string {
	if channelID == "" {
		return ""
	}

	if teamID != "" {
		base := fmt.Sprintf("https://app.slack.com/client/%s/%s", teamID, channelID)
		if messageTS != "" {
			return base + "/" + messageTS
		}
		return base
	}

	return ""
}

// SlackDMLink generates a deeplink to a DM conversation.
func SlackDMLink(teamID, userID string) string {
	if teamID == "" || userID == "" {
		return ""
	}
	return fmt.Sprintf("slack://user?team=%s&id=%s", teamID, userID)
}

// HyperlinkSupported returns true if the terminal likely supports
// OSC 8 hyperlinks (clickable links with display text).
func HyperlinkSupported() bool {
	// Most modern terminals support OSC 8: iTerm2, kitty, WezTerm,
	// GNOME Terminal, Windows Terminal, etc.
	term := os.Getenv("TERM_PROGRAM")
	switch term {
	case "iTerm.app", "WezTerm", "kitty":
		return true
	}
	// Check for general support hints
	if os.Getenv("COLORTERM") == "truecolor" {
		return true
	}
	return false
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence
// so the display text is clickable. Falls back to "text (url)" if the
// terminal doesn't support it.
func Hyperlink(url, displayText string) string {
	if url == "" {
		return displayText
	}
	if HyperlinkSupported() {
		return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, displayText)
	}
	return displayText
}

// FormatDeeplink returns a formatted deeplink string for display.
// Shows the link as a clickable URL or a "→ Open in Slack" hint.
func FormatDeeplink(teamID, channelID, messageTS string) string {
	link := SlackDeeplink(teamID, channelID, messageTS, "")
	if link == "" {
		return ""
	}
	return link
}

// PermalinkOrDeeplink returns the Slack deeplink if possible, falling
// back to the web permalink.
func PermalinkOrDeeplink(teamID, channelID, messageTS, permalink string) string {
	dl := SlackDeeplink(teamID, channelID, messageTS, "")
	if dl != "" {
		return dl
	}
	return permalink
}

// FormatMessageTS converts a Slack message timestamp for use in deeplinks.
// Slack deeplinks use the "p" prefix format: "1706000000.123456" → "p1706000000123456"
func FormatMessageTS(ts string) string {
	return "p" + strings.ReplaceAll(ts, ".", "")
}
